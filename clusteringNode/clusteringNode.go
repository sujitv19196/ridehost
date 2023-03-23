package clusterNode

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
func (s *AcceptClientFromIntroducer) findClusterInfo(request AcceptClientFromIntroducer, response *AcceptClientFromIntroducerResponse) error {

	s.DataList = append(s.DataList, data.Value)

	result := kMeansClustering(request.Uuid, request.Lat, request.Lng)
	response.ClusterNum = result.clusterNum
	response.Message = ""
	return nil

}

func kMeansClustering(string uuid, float lat, float lng) int {
	// Perform some operation here
	return lat
}
