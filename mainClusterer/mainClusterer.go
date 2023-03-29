package main

import (
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

var cluseringNodes = []string{"localhost:" + strconv.Itoa(Ports["clusteringNode"])}
var coresetList CoresetList

func main() {
	coresetList.cond = *sync.NewCond(&coresetList.mu)
	go acceptConnections()

	for { // send request to start clsutering to all nodes every ClusteringPeriod
		time.Sleep(ClusteringPeriod * time.Minute) // TODO might want a cond var here
		for _, node := range cluseringNodes {
			sendStartClusteringRPC(node)
		}
		// wait for all coresets to be recvd
		coresetList.mu.Lock()
		for len(coresetList.List) < len(cluseringNodes) {
			coresetList.cond.Wait()
		}
		coresetList.mu.Unlock()

		// TODO union of coresets
		clusterNums := coresetUnion()
		// send RPCs to clients with their cluster rep ip
		for node, cluserNum := range clusterNums {
			go sendClusterInfo(node, ClusterInfo{ClusterRep: "TODO", ClusterNum: cluserNum})
		}
	}
}

func acceptConnections() {
	// get my ip
	address, err := net.ResolveTCPAddr("tcp", "0.0.0.0:"+strconv.Itoa(Ports["introducer"]))
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
	go sendClusteringRPC(request)
	response.Message = "ACK"
	return nil
}

func sendClusteringRPC(request JoinRequest) {
	// randomly chose clusteringNode to forward request to
	seed := rand.NewSource(time.Now().UnixNano())
	r := rand.New(seed)
	clusterNum := r.Intn(len(cluseringNodes))
	conn, err := net.Dial("tcp", cluseringNodes[clusterNum])
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
	conn, err := net.Dial("tcp", clusterIp)
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}

	client := rpc.NewClient(conn)
	clusterResponse := new(MainClustererClusteringNodeResponse)

	// send clustering request to clusterNum clustering Node
	if client.Call("ClusteringNodeRPC.StartClustering", nil, &clusterResponse) != nil {
		log.Fatal("ClusteringNodeRPC.StartClustering error: ", err)
	}
}

func (m *MainClustererRPC) RecvCoreset(coreset Coreset, response *MainClustererClusteringNodeResponse) error {
	// add coreset to list
	coresetList.Append(coreset)
	response.Message = "ACK"
	return nil
}

func coresetUnion() map[Node]int {
	//TODO
	// outputs Node -> clusterNum
	n := Node{NodeType: Driver, Ip: nil, Uuid: [16]byte{}, Lat: 24, Lng: 25}
	return map[Node]int{n: 1}
}

func sendClusterInfo(node Node, clusterinfo ClusterInfo) {
	conn, err := net.Dial("tcp", node.Ip.String()) // TODO MIHGT BE WRONG IP
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
	client := rpc.NewClient(conn)
	response := new(ClientMainClustererResponse)
	err = client.Call("ClientRPC.RecvClusterInfo", clusterinfo, &response)
	if err != nil {
		log.Fatal("IntroducerRPC.ClientJoin error: ", err)
	}
}
