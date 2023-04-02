package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"ridehost/cll"
	"ridehost/constants"
	"ridehost/types"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ClientRPC struct{} // RPC

var ip *net.TCPAddr

var myIP net.IP
var myIPStr string
var logger *log.Logger

var mu sync.Mutex
var clusterId int
var clusterRepIP string
var isRep bool
var virtRing *cll.UniqueCLL
var joined bool
var startPinging bool

func main() {
	if len(os.Args) != 5 {
		fmt.Println("format: ./client nodeType introducerIp lat lng")
		os.Exit(1)
	}

	// external log file
	f, _ := os.OpenFile("output.txt", os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0666)
	defer f.Close()
	logger = log.New(f, "OUTPUT:", log.Ldate|log.Ltime|log.Lshortfile)

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

	go acceptPings()
	go sendPings()
	acceptClusteringConnections()
}

// command called by a client to join the system
func joinSystem(request types.JoinRequest) types.ClientIntroducerResponse {
	// request to introducer
	conn, err := net.Dial("tcp", request.IntroducerIp)
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
	clusterId = request.ClusterNum
	clusterRepIP = request.ClusterRepIP
	isRep = clusterRepIP == myIPStr
	virtRing = &cll.UniqueCLL{}
	virtRing.SetDefaults()
	for _, member := range request.Members {
		virtRing.PushBack(member)
	}
	joined = true
	logger.Print("joined cluster")
	response.Ack = true
	return nil
}

func (c *ClientRPC) StartPinging(request types.ClientClusterPingingStatusRequest, response *types.ClientClusterPingingStatusResponse) error {
	mu.Lock()
	defer mu.Unlock()
	startPinging = request.Status
	logger.Printf("pinging status changed to %t\n", startPinging)
	response.Ack = true
	return nil
}

func sendPings() {
	for {
		neighborIPs := []string{}
		mu.Lock()
		if joined && startPinging {
			neighborIPs = virtRing.GetNeighbors(myIPStr)
		}
		mu.Unlock()
		for _, neighborIP := range neighborIPs {
			sendPing(neighborIP)
		}
		// ping every second
		time.Sleep(constants.PingFrequency)
	}
}

func sendPing(neighborIP string) {
	// send ping
	buffer := make([]byte, 2048)
	conn, err := net.Dial("udp", neighborIP+":"+strconv.Itoa(constants.Ports["acceptPings"]))
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		go sendListRemoval(neighborIP)
		return
	}
	fmt.Fprintf(conn, "PING")
	conn.SetReadDeadline(time.Now().Add(constants.UDPPingAckTimeout))
	bytes_read, err := bufio.NewReader(conn).Read(buffer)
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		go sendListRemoval(neighborIP)
		return
	}
	// recieve response and check if it's an ack
	logger.Printf("[sendPing] Message from %s: \"%s\"\n", neighborIP, buffer[:bytes_read])
	if strings.Compare(string(buffer[:bytes_read]), "ACK") != 0 {
		logger.Printf("[sendPing] ACK not recieved from %s\n", neighborIP)
		// remove process ID from all membership lists if ack is not recieved
		go sendListRemoval(neighborIP)
	}
	conn.Close()
}

func acceptPings() {
	// listen for ping
	buffer := make([]byte, 2048)
	addr := net.UDPAddr{
		Port: constants.Ports["acceptPings"],
		IP:   myIP,
	}
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
	defer conn.Close()
	for {
		bytes_read, addr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			os.Stderr.WriteString(err.Error() + "\n")
			continue
		}
		// recieve message and check if it's a ping
		logger.Printf("[acceptPings] Message from %v: \"%s\"\n", addr, buffer[:bytes_read])
		if strings.Compare(string(buffer[:bytes_read]), "PING") != 0 {
			logger.Printf("[acceptPings] PING not recieved from %v\n", addr)
			continue
		}
		go func(conn *net.UDPConn, addr *net.UDPAddr, joined bool) {
			var err error
			if joined {
				// if it's a ping, send an ack
				_, err = conn.WriteToUDP([]byte("ACK"), addr)
			}

			if err != nil {
				os.Stderr.WriteString(err.Error() + "\n")
			}
		}(conn, addr, func() bool {
			mu.Lock()
			defer mu.Unlock()
			return joined
		}())
	}
}

func (c *ClientRPC) SendNodeFailure(request types.ClusterNodeRemovalRequest, response *types.ClusterNodeRemovalResponse) error {
	mu.Lock()
	defer mu.Unlock()
	virtRing.RemoveNode(request.NodeIP)
	fmt.Printf("removing node %s\n", request.NodeIP)
	// removing self
	if request.NodeIP == myIPStr {
		joined = false
		logger.Print("left cluster")
	}
	response.Ack = true
	return nil
}

func sendListRemoval(neighborIp string) {
	mu.Lock()
	IPs := virtRing.GetList()
	mu.Unlock()
	for _, ip := range IPs {
		go func(ip string, neighborIp string) {
			conn, err := net.DialTimeout("tcp", ip+":"+strconv.Itoa(constants.Ports["clientRPC"]), constants.TCPTimeout)
			// only throw error when can't connect to non-failed node
			if err != nil && ip != neighborIp {
				os.Stderr.WriteString(err.Error() + "\n")
				os.Exit(1)
			}

			client := rpc.NewClient(conn)
			response := new(types.ClusterNodeRemovalResponse)
			request := types.ClusterNodeRemovalRequest{}
			request.NodeIP = neighborIp
			err = client.Call("ClientRPC.SendNodeFailure", request, response)
			if err != nil {
				log.Fatal("SendNodeFailure error: ", err)
			}
			conn.Close()
		}(ip, neighborIp)
	}
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

func getMyIp() *net.TCPAddr {
	address, err := net.ResolveTCPAddr("tcp", "0.0.0.0:"+strconv.Itoa(constants.Ports["introducer"]))
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
