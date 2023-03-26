package introducer

import (
	"log"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	. "ridehost/types"
	"strconv"
	"time"
)

var ip net.IP
var ports = map[string]int{"introducer": 2233,
				"clusterNode": 2234}
var cluseringNodes []string // TODO can hard code these for now

type AcceptClient bool

func main() {
	// get this machine's IP address
	address, err := net.ResolveTCPAddr("tcp", "0.0.0.0:"+strconv.Itoa(ports["introducer"]))
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
}

// RPC exectued by introducer when new joins occur
func (a *AcceptClient) clientJoin(request ClientIntroducerRequest, response *ClientIntroducerResponse) error {
	// TODO make go routine
	// take the request of the cliient

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
	promise := client.Go("ClusteringNodeMembershipList.findClusterInfo", request, &clusterResponse, nil)
	// wait for RPC to finish
	<-promise.Done
	return *clusterResponse
}
