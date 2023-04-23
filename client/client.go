package main

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"ridehost/cll"
	"ridehost/constants"
	"ridehost/failureDetector"
	"ridehost/types"
	"strconv"
	"strings"
	"sync"

	"github.com/google/uuid"
)

type ClientRPC struct{} // RPC

var clientIp string

var myIP net.IP
var myIPStr string

var mu sync.Mutex
var isRep bool
var virtRing *cll.UniqueCLL
var joined bool
var startPinging bool
var clusterRepIP string
var clusterNum int
var nodeItself types.Node

// var wg sync.WaitGroup

func main() {
	if len(os.Args) != 5 {
		fmt.Println("format: ./client nodeType introducerIp lat lng")
		os.Exit(1)
	}

	// get this machine's IP address
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}

	mu = sync.Mutex{}

	mu.Lock()
	joined = false
	startPinging = false
	mu.Unlock()

	myIP = conn.LocalAddr().(*net.UDPAddr).IP
	myIPStr = myIP.String()
	conn.Close()

	// uuid := uuid.New()
	// nodeType, _ := strconv.Atoi(os.Args[1])
	// lat, _ := strconv.ParseFloat(os.Args[3], 64)
	// lng, _ := strconv.ParseFloat(os.Args[4], 64)
	// req := types.JoinRequest{NodeRequest: types.Node{NodeType: nodeType, Ip: ip, Uuid: uuid, Lat: lat, Lng: lng}, IntroducerIp: os.Args[2]}
	// r := joinSystem(req)
	// fmt.Println("From Introducer: ", r.Message)

	acceptClusteringConnections()

	clientIp = getMyIp()
	uuid := uuid.New()
	nodeType, _ := strconv.Atoi(os.Args[1])

	lat, _ := strconv.ParseFloat(os.Args[3], 64)
	lng, _ := strconv.ParseFloat(os.Args[4], 64)

	req := types.JoinRequest{NodeRequest: types.Node{NodeType: nodeType, Ip: clientIp, Uuid: uuid, Lat: lat, Lng: lng}, IntroducerIp: os.Args[2]}
	r := joinSystem(req)
	fmt.Println("From Introducer: ", r.Message)

	go startFailureDetector()
	acceptClusteringConnections()

	// form ring and start bidding
}

// command called by a client to join the system
func joinSystem(request types.JoinRequest) types.ClientIntroducerResponse {
	// request to introducer
	conn, err := net.Dial("tcp", request.IntroducerIp+":"+strconv.Itoa(constants.Ports["introducer"]))
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}

	client := rpc.NewClient(conn)
	response := new(types.ClientIntroducerResponse)
	err = client.Call("IntroducerRPC.ClientJoin", request, &response)
	if err != nil {
		log.Fatal("IntroducerRPC.ClientJoin error: ", err)
	}
	return *response
}

func (c *ClientRPC) JoinCluster(request types.ClientClusterJoinRequest, response *types.ClientClusterJoinResponse) error {
	mu.Lock()
	defer mu.Unlock()
	clusterNum = request.ClusterNum
	clusterRepIP = request.ClusterRepIP
	isRep = clusterRepIP == myIPStr
	virtRing = &cll.UniqueCLL{}
	virtRing.SetDefaults()
	for _, member := range request.Members {
		virtRing.PushBack(member)
	}
	joined = true
	log.Println("joined cluster")
	response.Ack = true
	return nil
}

func (c *ClientRPC) RecvClusterInfo(clusterInfo types.ClusterInfo, response *types.ClientMainClustererResponse) error {
	mu.Lock()
	nodeItself = clusterInfo.NodeItself
	clusterRep := clusterInfo.ClusterRep
	clusterNum = clusterInfo.ClusterNum
	fmt.Println("this client got clusterRep and clusterNum assigned as : ", nodeItself, clusterRep, clusterNum)
	mu.Unlock()
	return nil
}

func startFailureDetector() {
	address, err := net.ResolveTCPAddr("tcp", "0.0.0.0:"+strconv.Itoa(constants.Ports["failureDetector"]))
	if err != nil {
		log.Fatal("listen error:", err)
	}
	failureDetectorRPC := new(failureDetector.FailureDetectorRPC)
	failureDetectorRPC.Mu = &mu
	failureDetectorRPC.NodeItself = &nodeItself
	failureDetectorRPC.VirtRing = virtRing
	failureDetectorRPC.Joined = &joined
	failureDetectorRPC.StartPinging = &startPinging
	rpc.Register(failureDetectorRPC)
	conn, err := net.ListenTCP("tcp", address)
	if err != nil {
		log.Fatal("listen error:", err)
	}
	go failureDetector.SendPings(&mu, &joined, &startPinging, virtRing, myIPStr, "")
	go failureDetector.AcceptPings(myIP, &mu, &joined)
	rpc.Accept(conn)
}

func acceptClusteringConnections() {
	address, err := net.ResolveTCPAddr("tcp", "0.0.0.0:"+strconv.Itoa(constants.Ports["clientRPC"]))
	if err != nil {
		log.Fatal("listen error:", err)
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
	ipstr := strings.TrimSuffix(tcpAddr.String(), ":0") + ":" + strconv.Itoa(constants.Ports["client"])
	fmt.Println(ipstr)
	return ipstr
}

// client requests introduicer
// client gets back cluster number and cluster represnteitnve
// if rep: start taking join requests
// if not rep: send req to cluster rep to join cluster
// virtual ring with pings
