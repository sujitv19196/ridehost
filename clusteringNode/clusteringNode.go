package main

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	. "ridehost/constants"
	. "ridehost/types"
	"strconv"
	"sync"
)

type MembershipList struct {
	mu   sync.Mutex
	List []Node
}

func (m *MembershipList) Append(elem Node) {
	m.mu.Lock()
	m.List = append(m.List, elem)
	m.mu.Unlock()
}

var ip net.IP
var numClusterNodes = 0
var ML MembershipList

type ClusteringNodeRPC bool

func main() {
	go acceptConnections()
}

func acceptConnections() {
	// get this machine's IP address
	address, err := net.ResolveTCPAddr("tcp", "0.0.0.0:"+strconv.Itoa(Ports["clusteringNode"]))
	if err != nil {
		log.Fatal(err)
	}
	clusteringNodeRPC := new(ClusteringNodeRPC)
	rpc.Register(clusteringNodeRPC)
	conn, err := net.ListenTCP("tcp", address)
	if err != nil {
		log.Fatal("listen error:", err)
	}
	rpc.Accept(conn)
}

// cluster node accepts an RPC call from client node,
// get the cluster using the kmeans clustering function and return.
func (c *ClusteringNodeRPC) Cluster(request JoinRequest, response *MainClustererClusteringNodeResponse) error {
	fmt.Println("request from: ", request.NodeRequest.Uuid)
	ML.Append(request.NodeRequest)
	// wait for Kmeans to finish
	response.Message = "ACK"
	return nil
}

func (c *ClusteringNodeRPC) StartClustering(request string, response *MainClustererClusteringNodeResponse) error {
	go func() {
		ML.mu.Lock() // lock membership list and start clustering
		kMeansClustering()
		ML.mu.Unlock()
	}()
	response.Message = "ACK"
	return nil
}

func kMeansClustering() map[string]int {
	// TODO Perform some operation here

	// clear membership list
	return map[string]int{"test": 0}
}
