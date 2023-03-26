package main

import (
	"log"
	"net"
	"net/rpc"
	. "ridehost/types"
	"strconv"
)

var ip net.IP
var numClusterNodes = 0
var ports = map[string]int{"cluster": 2233, "clusterNode": 2234}

var MembershipList = []string{}

type AcceptClientFromIntroducer bool

func main() {
	// get this machine's IP address
	address, err := net.ResolveTCPAddr("tcp", "0.0.0.0:"+strconv.Itoa(ports["cluster"]))
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
	// go rpc.ServeConn(conn)
}

// cluster node accepts an RPC call from client node,
// get the cluster using the kmeans clustering function and return.
func (a *AcceptClientFromIntroducer) FindClusterInfo(request IntroducerClusterRequest, response *IntroducerClusterResponse) error {

	MembershipList = append(MembershipList, string(request.Uuid[:]))
	// wait 1 minute
	result := kMeansClustering()
	response.Result = strconv.Itoa(result["test"])
	response.Message = "TestMsg"
	return nil
}

func kMeansClustering() map[string]int {
	// TODO Perform some operation here

	return map[string]int{"test": 0}
}
