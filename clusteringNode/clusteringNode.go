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

type ClusteringNodeMembershipList struct {
	MembershipList []string
}

// cluster node accepts an RPC call from client node,
// get the cluster using the kmeans clustering function and return.
func (s *ClusteringNodeMembershipList) findClusterInfo(request IntroducerClusterRequest, response *IntroducerClusterResponse) error {

	s.MembershipList = append(s.MembershipList, string(request.Uuid[:]))
	result := kMeansClustering(s)
	response.Result = result
	response.Message = ""
	return nil
}

func kMeansClustering(member *ClusteringNodeMembershipList) []string {
	// Perform some operation here
	memberList := member.MembershipList

	return memberList
}
