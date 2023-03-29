package main

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	. "ridehost/constants"
	. "ridehost/types"
	"strconv"

	"github.com/google/uuid"
)

type Response ClientIntroducerResponse
type ClientRPC bool // RPC

var ip *net.TCPAddr

var clusterRep string
var clusterNum int

func main() {
	if len(os.Args) != 5 {
		fmt.Println("format: ./client nodeType introducerIp lat lng")
		os.Exit(1)
	}
	ip = getMyIp()
	uuid := uuid.New()
	nodeType, _ := strconv.Atoi(os.Args[1])
	lat, _ := strconv.ParseFloat(os.Args[3], 64)
	lng, _ := strconv.ParseFloat(os.Args[4], 64)
	req := JoinRequest{NodeRequest: Node{NodeType: nodeType, Ip: ip, Uuid: uuid, Lat: lat, Lng: lng}, IntroducerIp: os.Args[2]}
	r := joinSystem(req)
	fmt.Println("From Introducer: ", r.Message)

	go acceptClusteringConnections()
}

// command called by a client to join the system
func joinSystem(request JoinRequest) Response {
	// request to introducer
	conn, err := net.Dial("tcp", request.IntroducerIp)
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}

	client := rpc.NewClient(conn)
	response := new(Response)
	err = client.Call("IntroducerRPC.ClientJoin", request, &response)
	if err != nil {
		log.Fatal("IntroducerRPC.ClientJoin error: ", err)
	}
	return *response
}

func acceptClusteringConnections() {
	clientRPC := new(ClientRPC)
	rpc.Register(clientRPC)
	conn, err := net.ListenTCP("tcp", ip)
	if err != nil {
		log.Fatal("listen error:", err)
	}
	rpc.Accept(conn)
}

func (c *ClientRPC) RecvClusterInfo(clusterInfo ClusterInfo, response *Response) {
	clusterRep = clusterInfo.ClusterRep
	clusterNum = clusterInfo.ClusterNum
}

func getMyIp() *net.TCPAddr {
	address, err := net.ResolveTCPAddr("tcp", "0.0.0.0:"+strconv.Itoa(Ports["introducer"]))
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	return address
}

// client requests introduicer
// client gets back cluster number and cluster represnteitnve
// if rep: start taking join requests
// if not rep: send req to cluster rep to join cluster
// virtual ring with pings
