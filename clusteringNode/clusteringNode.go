package main

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	. "ridehost/constants"
	. "ridehost/kMeansClustering"
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

// VM 2
var mainClustererIp = "172.22.153.8:" + strconv.Itoa(Ports["mainClusterer"]) // TODO can hard code for now

type ClusteringNodeRPC bool

// accepts connections from main clusterer
// Cluster: takes cluster request and adds to membership list
// StartClustering: performs coreset calculation on current membership list. Locks list until done and new requests are queeue'd.
func main() {
	acceptConnections()
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
	defer conn.Close()
	rpc.Accept(conn)
}

// cluster node accepts an RPC call from client node,
// get the cluster using the kmeans clustering function and return.
func (c *ClusteringNodeRPC) Cluster(request JoinRequest, response *MainClustererClusteringNodeResponse) error {
	fmt.Println("request from: ", string(request.NodeRequest.Uuid[:]))
	go ML.Append(request.NodeRequest)
	response.Message = "ACK"
	return nil
}

func (c *ClusteringNodeRPC) StartClustering(nouse int, response *MainClustererClusteringNodeResponse) error {
	fmt.Println("Membership List: ")
	for _, elem := range ML.List {
		fmt.Print(string(elem.Uuid[:]))
	}
	go func() {
		coreset := Coreset{}

		// lock list until clustering is done (will still accept new requests which will be added after clsutering)
		ML.mu.Lock()
		if len(ML.List) >= NumClusters {
			coreset = kMeansClustering()
			// clear list after clutsering
			ML.List = nil
		}
		ML.mu.Unlock()
		// RPC call
		sendCoreset(coreset)
	}()
	response.Message = "ACK"
	return nil
}

func kMeansClustering() Coreset {
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
		return
	}
	defer conn.Close()

	client := rpc.NewClient(conn)
	clusterResponse := new(MainClustererClusteringNodeResponse)

	if client.Call("MainClustererRPC.RecvCoreset", coreset, &clusterResponse) != nil {
		os.Stderr.WriteString("MainClustererRPC.RecvCoreset error: " + err.Error())
	}
}
