package client

import (
	"fmt"
	"net"
	"net/rpc"
	"os"
	. "ridehost/types"
	"strconv"

	"github.com/google/uuid"
)

type Response ClientIntroducerResponse

func main() {
	if len(os.Args) != 5 {
		fmt.Println("format: ./client nodeType introducerIp lat lng")
	}
	uuid := uuid.New()
	nodeType, _ := strconv.Atoi(os.Args[1])
	lat, _ := strconv.ParseFloat(os.Args[3], 64)
	lng, _ := strconv.ParseFloat(os.Args[4], 64)
	req := ClientIntroducerRequest{RequestType: nodeType, IntroducerIP: os.Args[2], Uuid: uuid, Lat: lat, Lng: lng}
	r := joinSystem(req)
	fmt.Println(r.ClusterNum)
}

// command called by a client to join the system
func joinSystem(request ClientIntroducerRequest) Response {
	// request to introducer
	conn, err := net.Dial("tcp", request.IntroducerIP)
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}

	client := rpc.NewClient(conn)
	response := new(Response)
	promise := client.Go("AcceptClient.clientJoin", request, &response, nil)
	// wait for RPC to finish
	<-promise.Done
	return *response
}

// client requests introduicer
// client gets back cluster number and cluster represnteitnve
// if rep: start taking join requests
// if not rep: send req to cluster rep to join cluster
// virtual ring with pings
