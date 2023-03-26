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
	"time"
)

type MembershipList struct {
	mu   sync.Mutex
	List []string
}

func (m *MembershipList) Append(elem string) {
	m.mu.Lock()
	m.List = append(m.List, elem)
	m.mu.Unlock()
}

var ip net.IP
var numClusterNodes = 0
var ML MembershipList

type AcceptClientFromIntroducer bool

func main() {
	go acceptConnections()
	// every ClusteringPeriod minute, run K means
	for {
		time.Sleep(ClusteringPeriod * time.Minute) // TODO might want a cond var here
		ML.mu.Lock()                               // prevent any more adds to the membership list
		result := kMeansClustering()
		ML.mu.Unlock()
		fmt.Println(result)
	}
}

func acceptConnections() {
	// get this machine's IP address
	address, err := net.ResolveTCPAddr("tcp", "0.0.0.0:"+strconv.Itoa(Ports["clusteringNode"]))
	if err != nil {
		log.Fatal(err)
	}
	acceptClient := new(AcceptClientFromIntroducer)
	rpc.Register(acceptClient)
	conn, err := net.ListenTCP("tcp", address)
	if err != nil {
		log.Fatal("listen error:", err)
	}
	rpc.Accept(conn)
}

// cluster node accepts an RPC call from client node,
// get the cluster using the kmeans clustering function and return.
func (a *AcceptClientFromIntroducer) FindClusterInfo(request IntroducerClusterRequest, response *IntroducerClusterResponse) error {
	fmt.Println("request from: ", request.Uuid)
	ML.Append(string(request.Uuid[:]))
	// wait for Kmeans to finish
	response.Result = strconv.Itoa(0) // for testing only
	response.Message = "TestMsg"
	return nil
}

func kMeansClustering() map[string]int {
	// TODO Perform some operation here

	// clear membership list
	return map[string]int{"test": 0}
}
