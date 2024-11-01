package main

import (
	"fmt"
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
	var joinConns []*net.Conn

	for _, client := range clients {
		conn, err := net.DialTimeout("tcp", client+":"+strconv.Itoa(constants.Ports["failureDetector"]), constants.TCPTimeout)
		if err != nil {
			os.Stderr.WriteString(err.Error() + "\n")
			os.Exit(1)
		}

		client := rpc.NewClient(conn)
		response := new(types.ClientClusterJoinResponse)
		joinResponses = append(joinResponses, response)
		request := types.ClientClusterJoinRequest{}
		request.ClusterRep.Ip = clusterRep
		request.ClusterNum = 0
		request.Members = clients
		joinPromises = append(joinPromises, client.Go("ClientRPC.JoinCluster", request, response, nil))
		if err != nil {
			log.Fatal("JoinCluster error: ", err)
		}
		joinConns = append(joinConns, &conn)
	}

	for i, promise := range joinPromises {
		<-promise.Done
		if joinResponses[i].Ack != true {
			log.Fatalf("%s did not join cluster: %s", clients[i], promise.Error)
		}
		(*joinConns[i]).Close()
	}

	var pingPromises []*rpc.Call
	var pingResponses []*types.ClientClusterPingingStatusResponse
	var pingConns []*net.Conn

	for _, client := range clients {
		conn, err := net.DialTimeout("tcp", client+":"+strconv.Itoa(constants.Ports["failureDetector"]), constants.TCPTimeout)
		if err != nil {
			os.Stderr.WriteString(err.Error() + "\n")
			os.Exit(1)
		}

		client := rpc.NewClient(conn)
		response := new(types.ClientClusterPingingStatusResponse)
		pingResponses = append(pingResponses, response)
		request := types.ClientClusterPingingStatusRequest{}
		request.Status = true
		pingPromises = append(pingPromises, client.Go("FailureDetectorRPC.StartPings", request, response, nil))
		if err != nil {
			log.Fatal("StartPinging error: ", err)
		}
		pingConns = append(pingConns, &conn)
	}

	for i, promise := range pingPromises {
		fmt.Printf("waiting for %d ping promise\n", i)
		<-promise.Done
		if pingResponses[i].Ack != true {
			log.Fatalf("%s did not start pinging", clients[i])
		}
		fmt.Printf("closing %d start pinging conn\n", i)
		(*pingConns[i]).Close()
		fmt.Printf("closed %d start pinging conn\n", i)
	}
	fmt.Println("test client done")
}
