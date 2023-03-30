package main

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	. "ridehost/constants"
	. "ridehost/kmeansclustering"
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

func (m *MembershipList) Clear() {
	m.mu.Lock()
	m.List = nil
	m.mu.Unlock()
}

var ip net.IP
var numClusterNodes = 0
var ML MembershipList

var mainClustererIp = "localhost:" + strconv.Itoa(Ports["mainClusterer"]) // TODO can hard code for now

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
	response.Message = "ACK"
	return nil
}

func (c *ClusteringNodeRPC) StartClustering(requestclientcount int, response *MainClustererClusteringNodeResponse) error {
	go func() {
		ML.mu.Lock() // lock membership list and start clustering
		coreset := kMeansClustering(requestclientcount)
		// clear membership list
		ML.Clear()
		// RPC call
		sendCoreset(coreset)
		ML.mu.Unlock()
	}()
	response.Message = "ACK"
	return nil
}

func kMeansClustering(clientcount int) Coreset {
	// t = clientcount //total number of clients. It should come from mainClusterer.
	// coreset := IndividualKMeansClustering(ML.List, NumClusters, clientcount)

	// Calling the centralized K means clustering for current implementation
	// it returns cluster type and we will create a coreset type from it
	clusterresult := ClusterResult{}
	clusterresult = CentralizedKMeansClustering(ML.List, NumClusters)
	coreset := Coreset{Coreset: []Point{}, CoresetNodes: []Node{}, Tempcluster: clusterresult.ClusterMaps}

	return coreset
}

// calls MainClustererRPC.RecvCoreset to give it computed coreset
func sendCoreset(coreset Coreset) {
	conn, err := net.Dial("tcp", mainClustererIp)
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}

	client := rpc.NewClient(conn)
	clusterResponse := new(MainClustererClusteringNodeResponse)

	if client.Call("MainClustererRPC.RecvCoreset", coreset, &clusterResponse) != nil {
		log.Fatal("MainClustererRPC.RecvCoreset error: ", err)
	}
}
