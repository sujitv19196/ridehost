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
		// take union of coresets
		clusterNums := coresetUnion()
		// make cope list and then clear list
		tempcorelist := coresetList.List
		coresetList.List = nil
		coresetList.mu.Unlock()

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
		// TODO List of members of cluster
		// send RPCs to clients with their cluster rep ip
		for node, clusterNum := range clusterNums {
			clusterinfo := ClientClusterJoinRequest{NodeItself: node, ClusterRep: coreunion[node], ClusterNum: clusterNum}
			go sendClusterInfo(node, clusterinfo)
		}
	}
}

func acceptConnections() {
	// get my ip
	address, err := net.ResolveTCPAddr("tcp", "0.0.0.0:"+strconv.Itoa(Ports["mainClusterer"]))
	if err != nil {
		log.Fatal(err)
	}
	mainClustererRPC := new(MainClustererRPC)
	rpc.Register(mainClustererRPC)
	conn, err := net.ListenTCP("tcp", address)
	if err != nil {
		log.Fatal("listen error:", err)
	}
	defer conn.Close()
	rpc.Accept(conn)
}

func (m *MainClustererRPC) ClusteringRequest(request JoinRequest, response *IntroducerMainClustererResponse) error {
	go sendClusteringRPC(request)
	response.Message = "ACK"
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
		return
	}
	defer conn.Close()

	client := rpc.NewClient(conn)
	clusterResponse := new(MainClustererClusteringNodeResponse)

	// send clustering request to clusterNum clustering Node
	if client.Call("ClusteringNodeRPC.Cluster", request, &clusterResponse) != nil {
		os.Stderr.WriteString("ClusteringNodeRPC.Cluster error: " + err.Error())
	}
}

func sendStartClusteringRPC(clusterIp string) {
	fmt.Println("start clustering at: ", clusterIp)
	conn, err := net.Dial("tcp", clusterIp)
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		return
	}
	defer conn.Close()

	client := rpc.NewClient(conn)
	// var clusterResponse *MainClustererClusteringNodeResponse
	clusterResponse := new(MainClustererClusteringNodeResponse)

	// send start clustering request to clusterNum clustering Node
	if client.Call("ClusteringNodeRPC.StartClustering", 0, &clusterResponse) != nil {
		os.Stderr.WriteString("ClusteringNodeRPC.StartClustering error: " + err.Error())
	}
}

func (m *MainClustererRPC) RecvCoreset(coreset Coreset, response *MainClustererClusteringNodeResponse) error {
	// add coreset to list
	go coresetList.Append(coreset)
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
}

// send cluster info to client nodes
func sendClusterInfo(node Node, clusterinfo ClientClusterJoinRequest) {
	fmt.Println("Send cluster info to: ", node.Ip)
	conn, err := net.Dial("tcp", node.Ip)
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		return
	}
	defer conn.Close()
	client := rpc.NewClient(conn)
	response := new(ClientMainClustererResponse)
	fmt.Println("Node ", clusterinfo.NodeItself.Uuid.String(), "-> Clsuter ", clusterinfo.ClusterNum, "has cluster rep: ", clusterinfo.ClusterRep.Uuid.String())

	err = client.Call("ClientRPC.JoinCluster", clusterinfo, &response)
	if err != nil {
		os.Stderr.WriteString("ClientRPC.ClientJoin error: " + err.Error())
	}
}
