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
	. "ridehost/types"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type ClientRPC struct{} // RPC

var ip *net.TCPAddr

var myIP string
var logger *log.Logger

var mu sync.Mutex
var clusterId int
var clusterRepIP string
var isRep bool
var virtRing *cll.UniqueCLL
var joined bool

func SetJoined(new_join bool) {
	mu.Lock()
	joined = new_join
	mu.Unlock()
}

func ReadJoined() bool {
	mu.Lock()
	defer mu.Unlock()
	return joined
}

func SetIsRep(new_is_rep bool) {
	mu.Lock()
	isRep = new_is_rep
	mu.Unlock()
}

func ReadIsRep() bool {
	mu.Lock()
	defer mu.Unlock()
	return isRep
}

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

	SetJoined(false)

	myIP = conn.LocalAddr().(*net.UDPAddr).IP.String()
	conn.Close()

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
func joinSystem(request JoinRequest) ClientIntroducerResponse {
	// request to introducer
	conn, err := net.Dial("tcp", request.IntroducerIp)
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}

	client := rpc.NewClient(conn)
	response := new(ClientIntroducerResponse)
	err = client.Call("IntroducerRPC.ClientJoin", request, &response)
	if err != nil {
		log.Fatal("IntroducerRPC.ClientJoin error: ", err)
	}
	return *response
}

func (c *ClientRPC) JoinCluster(request ClientClusterJoinRequest, response *ClientClusterJoinResponse) error {
	mu.Lock()
	defer mu.Unlock()
	clusterId = request.ClusterNum
	clusterRepIP = request.ClusterRepIP
	isRep = clusterRepIP == myIP
	virtRing = &cll.UniqueCLL{}
	virtRing.SetDefaults()
	for _, member := range request.Members {
		virtRing.PushBack(member)
	}
	joined = true

	response.Ack = true
	return nil
}

func sendPings() {
	for {
		neighborIPs := []string{}
		mu.Lock()
		if joined {
			neighborIPs = virtRing.GetNeighbors(myIP)
		}
		mu.Unlock()
		for _, neighborIP := range neighborIPs {
			sendPing(neighborIP)
		}
		// ping every second
		time.Sleep(time.Second)
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
	conn.SetReadDeadline(time.Now().Add(constants.UDPTimeoutMillseconds * time.Millisecond)) // 1.5 seconds
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

func (c *ClientRPC) SendNodeFailure(request ClusterNodeRemovalRequest, response *ClusterNodeRemovalResponse) error {
	mu.Lock()
	defer mu.Unlock()
	virtRing.RemoveNode(request.NodeIP)
	response.Ack = true
	return nil
}

func sendListRemoval(neighborIp string) {
	mu.Lock()
	IPs := virtRing.GetList()
	mu.Unlock()
	for _, ip := range IPs {
		go func(ip string) {
			conn, err := net.Dial("tcp", ip)
			if err != nil {
				os.Stderr.WriteString(err.Error() + "\n")
				os.Exit(1)
			}

			client := rpc.NewClient(conn)
			response := new(ClusterNodeRemovalResponse)
			request := ClusterNodeRemovalRequest{}
			request.NodeIP = neighborIp
			err = client.Call("ClientRPCs.SendNodeFailure", request, &response)
			if err != nil {
				log.Fatal("SendNodeFailure error: ", err)
			}
			conn.Close()
		}(ip)
	}
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
