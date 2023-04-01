package test_client

import (
	"log"
	"net"
	"net/rpc"
	"os"
	"ridehost/constants"
	"ridehost/types"
	"strconv"
)

func main() {
	clients := os.Args[1:]
	clusterRep := clients[0]

	var joinPromises []*rpc.Call
	var joinResponses []*types.ClientClusterJoinResponse

	for _, client := range clients {
		conn, err := net.DialTimeout("tcp", client+":"+strconv.Itoa(constants.Ports["clientRPC"]), constants.TCPTimeout)
		if err != nil {
			os.Stderr.WriteString(err.Error() + "\n")
			os.Exit(1)
		}

		client := rpc.NewClient(conn)
		response := new(types.ClientClusterJoinResponse)
		joinResponses = append(joinResponses, response)
		request := types.ClientClusterJoinRequest{}
		request.ClusterRepIP = clusterRep
		request.ClusterNum = 0
		request.Members = clients
		joinPromises = append(joinPromises, client.Go("ClientRPCs.JoinCluster", request, response, nil))
		if err != nil {
			log.Fatal("JoinCluster error: ", err)
		}
		conn.Close()
	}

	for i, promise := range joinPromises {
		<-promise.Done
		if joinResponses[i].Ack != true {
			log.Fatalf("%s did not join cluster", clients[i])
		}
	}

	var pingPromises []*rpc.Call
	var pingResponses []*types.ClientClusterPingingStatusResponse

	for _, client := range clients {
		conn, err := net.DialTimeout("tcp", client+":"+strconv.Itoa(constants.Ports["clientRPC"]), constants.TCPTimeout)
		if err != nil {
			os.Stderr.WriteString(err.Error() + "\n")
			os.Exit(1)
		}

		client := rpc.NewClient(conn)
		response := new(types.ClientClusterPingingStatusResponse)
		pingResponses = append(pingResponses, response)
		request := types.ClientClusterPingingStatusRequest{}
		request.Status = true
		pingPromises = append(joinPromises, client.Go("ClientRPCs.StartPinging", request, response, nil))
		if err != nil {
			log.Fatal("StartPinging error: ", err)
		}
		conn.Close()
	}

	for i, promise := range pingPromises {
		<-promise.Done
		if pingResponses[i].Ack != true {
			log.Fatalf("%s did not start pinging", clients[i])
		}
	}

}
