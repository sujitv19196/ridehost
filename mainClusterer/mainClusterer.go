package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	. "ridehost/constants"
	. "ridehost/types"
	"strconv"
	"sync"
	"time"
)

type MainClustererRPC bool

type CoresetList struct {
	mu   sync.Mutex
	cond sync.Cond
	List []Coreset
}

func (c *CoresetList) Append(elem Coreset) {
	c.mu.Lock()
	c.List = append(c.List, elem)
	c.mu.Unlock()
	c.cond.Signal()
}

func (c *CoresetList) Clear() {
	c.mu.Lock()
	c.List = nil
	c.mu.Unlock()
}

var clusteringNodes = []string{"172.22.155.51:" + strconv.Itoa(Ports["clusteringNode"])}
var coresetList CoresetList

func main() {
	coresetList.cond = *sync.NewCond(&coresetList.mu)
	go acceptConnections()

	for { // send request to start clustering to all nodes every ClusteringPeriod
		time.Sleep(ClusteringPeriod * time.Minute) // TODO might want a cond var here

		for _, node := range clusteringNodes {
			sendStartClusteringRPC(node)
		}

		// wait for all coresets to be recvd
		coresetList.mu.Lock()
		for len(coresetList.List) < len(clusteringNodes) {
			coresetList.cond.Wait()
		}
		coresetList.mu.Unlock()

		// TODO union of coresets
		clusterNums := coresetUnion()
		tempcorelist := coresetList.List
		coresetList.Clear()

		// find ClusterRepresentation Info to send to client
		coreunion := map[Node]Node{}
		corelist := tempcorelist
		for _, core := range corelist {
			for repNode, clusterList := range core.Tempcluster {
				for _, node := range clusterList {
					coreunion[node] = repNode
				}
			}
		}

		// send RPCs to clients with their cluster rep ip
		for node, clusterNum := range clusterNums {
			clusterinfo := ClusterInfo{NodeItself: node, ClusterRep: coreunion[node], ClusterNum: clusterNum}
			go sendClusterInfo(node, clusterinfo)
		}
	}
}

func acceptConnections() {
	// get my ip
	address, err := net.ResolveTCPAddr("tcp", "0.0.0.0:"+strconv.Itoa(Ports["mainClusterer"]))
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	mainClustererRPC := new(MainClustererRPC)
	rpc.Register(mainClustererRPC)
	conn, err := net.ListenTCP("tcp", address)
	if err != nil {
		log.Fatal("listen error:", err)
	}
	rpc.Accept(conn)
}

func (m *MainClustererRPC) ClusteringRequest(request JoinRequest, response *IntroducerMainClustererResponse) error {
	fmt.Println("hello")
	go sendClusteringRPC(request)
	response.Message = "ACK"
	// totalclient.Increment() //increment count of t when a client request is sent to a clusteringnode
	// fmt.Println("Total Client Count", totalclient.count)
	return nil
}

func sendClusteringRPC(request JoinRequest) {
	// randomly chose clusteringNode to forward request to
	seed := rand.NewSource(time.Now().UnixNano())
	r := rand.New(seed)
	clusterNum := r.Intn(len(clusteringNodes))
	fmt.Println("send cluster req to: ", clusteringNodes[clusterNum])
	conn, err := net.Dial("tcp", clusteringNodes[clusterNum])
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}

	client := rpc.NewClient(conn)
	clusterResponse := new(MainClustererClusteringNodeResponse)

	// send clustering request to clusterNum clustering Node
	if client.Call("ClusteringNodeRPC.Cluster", request, &clusterResponse) != nil {
		log.Fatal("ClusteringNodeRPC.Cluster error: ", err)
	}
}

func sendStartClusteringRPC(clusterIp string) {
	fmt.Println("start clustering at: ", clusterIp)
	conn, err := net.Dial("tcp", clusterIp)
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}

	client := rpc.NewClient(conn)
	// var clusterResponse *MainClustererClusteringNodeResponse
	clusterResponse := new(MainClustererClusteringNodeResponse)

	// send start clustering request to clusterNum clustering Node
	if client.Call("ClusteringNodeRPC.StartClustering", 0, &clusterResponse) != nil {
		log.Fatal("ClusteringNodeRPC.StartClustering error: ", err)
	}
}

func (m *MainClustererRPC) RecvCoreset(coreset Coreset, response *MainClustererClusteringNodeResponse) error {
	// add coreset to list
	coresetList.Append(coreset)
	response.Message = "ACK"
	return nil
}

// {
// List: [  => corelist
// core => {
// Tempcluster  map[Node][]Node
// }
// ]
// }
func coresetUnion() map[Node]int {
	//TODO
	// outputs Node -> clusterNum
	coreunion := map[Node]int{}
	corelist := coresetList.List
	num := 0
	for _, core := range corelist {
		for _, clusterList := range core.Tempcluster {
			for _, node := range clusterList {
				coreunion[node] = num
			}
			num += 1
		}
	}

	fmt.Println("CoreUnion: ", coreunion)
	return coreunion
	// n := Node{NodeType: Driver, Ip: nil, Uuid: [16]byte{}, Lat: 24, Lng: 25}
	// return map[Node]int{n: 1}
}

func sendClusterInfo(node Node, clusterinfo ClusterInfo) {
	fmt.Println("Send cluster info to: ", node.Ip)
	conn, err := net.Dial("tcp", node.Ip)
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
	client := rpc.NewClient(conn)
	response := new(ClientMainClustererResponse)
	fmt.Println("send this clusterRep and clusterNum from main: ", clusterinfo)

	err = client.Call("ClientRPC.RecvClusterInfo", clusterinfo, &response)
	if err != nil {
		log.Fatal("IntroducerRPC.ClientJoin error: ", err)
	}
}
