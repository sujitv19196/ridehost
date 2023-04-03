package main

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"strings"

	. "ridehost/constants"
	. "ridehost/types"
	"strconv"

	"github.com/google/uuid"
)

type Response ClientIntroducerResponse
type ClientRPC bool // RPC

var clientIp string

var clusterRep Node
var clusterNum int
var nodeItself Node

// var wg sync.WaitGroup

func main() {
	if len(os.Args) != 5 {
		fmt.Println("format: ./client nodeType introducerIp lat lng")
		os.Exit(1)
	}
	clientIp = getMyIp()
	uuid := uuid.New()
	nodeType, _ := strconv.Atoi(os.Args[1])

	lat, _ := strconv.ParseFloat(os.Args[3], 64)
	lng, _ := strconv.ParseFloat(os.Args[4], 64)
	req := JoinRequest{NodeRequest: Node{NodeType: nodeType, Ip: clientIp, Uuid: uuid, Lat: lat, Lng: lng}, IntroducerIp: os.Args[2]}
	r := joinSystem(req)
	fmt.Println("From Introducer: ", r.Message)
	// wg.Add(1)
	//acceptClusteringConnections()
	// wg.Wait()
}

// command called by a client to join the system
func joinSystem(request JoinRequest) Response {
	// request to introducer
	conn, err := net.Dial("tcp", request.IntroducerIp+":"+strconv.Itoa(Ports["introducer"]))
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

func (c *ClientRPC) RecvClusterInfo(clusterInfo ClusterInfo, response *Response) error {
	nodeItself = clusterInfo.NodeItself
	clusterRep = clusterInfo.ClusterRep
	clusterNum = clusterInfo.ClusterNum
	fmt.Println("this client got clusterRep and clusterNum assigned as : ", nodeItself, clusterRep, clusterNum)
	os.Exit(0)
	return nil
}

func acceptClusteringConnections() {
	address, err := net.ResolveTCPAddr("tcp", "0.0.0.0:"+strconv.Itoa(Ports["client"]))
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	clientRPC := new(ClientRPC)
	rpc.Register(clientRPC)
	conn, err := net.ListenTCP("tcp", address)
	if err != nil {
		log.Fatal("listen error:", err)
	}
	rpc.Accept(conn)
}

func getMyIp() string {
	ief, err := net.InterfaceByName("eth0")
	if err != nil {
		log.Fatal(err)
	}
	addrs, err := ief.Addrs()
	if err != nil {
		log.Fatal(err)
	}

	tcpAddr := &net.TCPAddr{
		IP: addrs[0].(*net.IPNet).IP,
	}
	ipstr := strings.TrimSuffix(tcpAddr.String(), ":0") + ":" + strconv.Itoa(Ports["client"])
	fmt.Println(ipstr)
	return ipstr
}

// client requests introduicer
// client gets back cluster number and cluster represnteitnve
// if rep: start taking join requests
// if not rep: send req to cluster rep to join cluster
// virtual ring with pings
