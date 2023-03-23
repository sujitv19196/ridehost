package client

import (
	"net"
	"net/rpc"
	"os"
	. "ridehost/types"
)

type Response ClientIntroducerResponse

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
