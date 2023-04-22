package main

import (
	"bufio"
	"fmt"
	"log"
	"math"
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

	go sendPings()
	go acceptPings()

	req := types.JoinRequest{NodeRequest: types.Node{NodeType: nodeType, Ip: clientIp, Uuid: uuid, Lat: lat, Lng: lng}, IntroducerIp: os.Args[2]}
	r := joinSystem(req)
	fmt.Println("From Introducer: ", r.Message)

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

func (c *ClientRPC) StartPinging(request types.ClientClusterPingingStatusRequest, response *types.ClientClusterPingingStatusResponse) error {
	mu.Lock()
	defer mu.Unlock()
	startPinging = request.Status
	log.Printf("pinging status changed to %t\n", startPinging)
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
		var wg sync.WaitGroup
		wg.Add(len(neighborIPs))
		for _, neighborIP := range neighborIPs {
			go AttemptPings(neighborIP, &wg)
		}
		wg.Wait()
		// ping every second
		time.Sleep(constants.PingFrequency)
	}
}

func AttemptPings(neighborIP string, wg *sync.WaitGroup) {
	defer wg.Done()
	if !sendPing(neighborIP) {
		for i := 0; i < 2; i++ {
			time.Sleep(time.Duration(math.Pow(2, float64(i+1))) * time.Second)
			if sendPing(neighborIP) {
				return
			}
		}
		RemoveNode(neighborIP)
	}
}

func sendPing(neighborIP string) bool {
	// send ping
	buffer := make([]byte, 2048)
	conn, err := net.Dial("udp", neighborIP+":"+strconv.Itoa(constants.Ports["acceptPings"]))
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		return false
	}
	defer conn.Close()
	fmt.Fprintf(conn, "PING")
	conn.SetReadDeadline(time.Now().Add(constants.UDPPingAckTimeout))
	bytes_read, err := bufio.NewReader(conn).Read(buffer)
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		return false
	}
	// recieve response and check if it's an ack
	// logger.Printf("[sendPing] Message from %s: \"%s\"\n", neighborIP, buffer[:bytes_read])
	if strings.Compare(string(buffer[:bytes_read]), "ACK") != 0 {
		// remove process ID from all membership lists if ack is not recieved
		log.Printf("[sendPing] ACK not recieved from %s\n", neighborIP)
		return false
	}
	return true
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
		// logger.Printf("[acceptPings] Message from %v: \"%s\"\n", addr, buffer[:bytes_read])
		if strings.Compare(string(buffer[:bytes_read]), "PING") != 0 {
			log.Printf("[acceptPings] PING not recieved from %v\n", addr)
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
	log.Printf("removing node %s\n", request.NodeIP)
	// removing self
	if request.NodeIP == myIPStr {
		joined = false
		log.Println("left cluster")
	}
	response.Ack = true
	return nil
}

func RemoveNode(nodeIP string) {
	mu.Lock()
	// get list before removing node so failed node gets rpc saying it failed
	IPs := virtRing.GetList()
	virtRing.RemoveNode(nodeIP)
	mu.Unlock()
	go sendListRemoval(nodeIP, IPs)
	log.Printf("removed node %s\n", nodeIP)
}

func sendListRemoval(neighborIp string, IPs []string) {
	for _, ip := range IPs {
		if ip != myIPStr {
			go func(ip string, neighborIp string) {
				conn, err := net.DialTimeout("tcp", ip+":"+strconv.Itoa(constants.Ports["clientRPC"]), constants.TCPTimeout)
				// only throw error when can't connect to non-failed node
				if err != nil {
					if ip == neighborIp {
						return
					}
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
