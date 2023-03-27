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
	"time"
)

var ip net.IP
var cluseringNodes = []string{"localhost:2234"} // TODO can hard code these for now

type AcceptClient bool

func main() {
	// get this machine's IP address
	address, err := net.ResolveTCPAddr("tcp", "0.0.0.0:"+strconv.Itoa(Ports["introducer"]))
	if err != nil {
		log.Fatal(err)
	}
	acceptClient := new(AcceptClient)
	rpc.Register(acceptClient)
	conn, err := net.ListenTCP("tcp", address)
	if err != nil {
		log.Fatal("listen error:", err)
	}
	rpc.Accept(conn)

	// clusterResponse := sendClusteringRPC(0, IntroducerClusterRequest{Uuid: [16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
	// 	Lat: 23,
	// 	Lng: 24}) // get assinged clsuter group back

	// // give repsonse to client
	// r := ClientIntroducerResponse{ClusterNum: clusterResponse.ClusterNum, Error: clusterResponse.Error}
	// fmt.Println(r.ClusterNum)
}

// RPC exectued by introducer when new joins occur
func (a *AcceptClient) ClientJoin(request ClientIntroducerRequest, response *ClientIntroducerResponse) error {
	// take the requests of the cliient and imediately send to a clusteringNode
	// randomly chose clusteringNode to forward request to
	seed := rand.NewSource(time.Now().UnixNano())
	r := rand.New(seed)
	clusterNum := r.Intn(len(cluseringNodes))

	// RPC to clusterNum for kmeans
	clusterResponse := sendClusteringRPC(clusterNum, IntroducerClusterRequest{Uuid: request.Uuid,
		Lat: request.Lat,
		Lng: request.Lng}) // get assinged clsuter group back

	// give repsonse to client
	*response = ClientIntroducerResponse{ClusterNum: clusterResponse.ClusterNum, Error: clusterResponse.Error}
	return nil
}

func sendClusteringRPC(clusterNum int, request IntroducerClusterRequest) IntroducerClusterResponse {
	conn, err := net.Dial("tcp", cluseringNodes[clusterNum])
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}

	client := rpc.NewClient(conn)
	clusterResponse := new(IntroducerClusterResponse)
	if client.Call("ClusteringNodeMembershipList.FindClusterInfo", request, &clusterResponse) != nil {
		log.Fatal("FindClusterInfo error: ", err)
	}
	return *clusterResponse
}
