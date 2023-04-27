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
	. "ridehost/constants"
	"ridehost/types"
	. "ridehost/types"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type ClientRPC struct{} // RPC

// rider auction state
type AuctionState struct {
	mu          sync.Mutex
	acceptedBid bool
}

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

var auctionState AuctionState
var timeStart time.Time

func main() {
	if len(os.Args) >= 3 {
		nodeType, _ := strconv.Atoi(os.Args[1])
		if nodeType == Driver && (len(os.Args) < 5) {
			fmt.Println("format for Driver: ./client nodeType introducerIp startlat startlng")
			os.Exit(1)
		} else if nodeType == Rider && (len(os.Args) < 7) {
			fmt.Println("format for Driver: ./client nodeType introducerIp startlat startlng destlat destlng")
			os.Exit(1)
		}
	}
	// get this machine's IP address
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}

	myIP = conn.LocalAddr().(*net.UDPAddr).IP
	myIPStr = myIP.String()
	conn.Close()

	clientIp = myIPStr + ":" + strconv.Itoa(Ports["clientRPC"]) //add port to IP
	fmt.Println(clientIp)
	uuid := uuid.New()
	nodeType, _ := strconv.Atoi(os.Args[1])

	startLat, _ := strconv.ParseFloat(os.Args[3], 64)
	startLng, _ := strconv.ParseFloat(os.Args[4], 64)
	req := JoinRequest{}
	if nodeType == Rider {
		destLat, _ := strconv.ParseFloat(os.Args[5], 64)
		destLng, _ := strconv.ParseFloat(os.Args[6], 64)
		req = JoinRequest{NodeRequest: Node{NodeType: nodeType, Ip: clientIp, Uuid: uuid, StartLat: startLat, StartLng: startLng, DestLat: destLat, DestLng: destLng}, IntroducerIp: os.Args[2]}
	} else {
		req = JoinRequest{NodeRequest: Node{NodeType: nodeType, Ip: clientIp, Uuid: uuid, StartLat: startLat, StartLng: startLng}, IntroducerIp: os.Args[2]}
	}
	r := joinSystem(req)
	log.Println("From Introducer: ", r.Message)

	mu = sync.Mutex{}
	mu.Lock()
	joined = false
	startPinging = false
	mu.Unlock()
	go sendPings()
	go acceptPings()

	auctionState = AuctionState{mu: sync.Mutex{}, acceptedBid: false}
	acceptClusteringConnections()
}

// command called by a client to join the system
func joinSystem(request types.JoinRequest) types.ClientIntroducerResponse {
	// request to introducer
	conn, err := net.Dial("tcp", request.IntroducerIp+":"+strconv.Itoa(constants.Ports["introducer"]))
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
	defer conn.Close()

	client := rpc.NewClient(conn)
	response := new(types.ClientIntroducerResponse)
	err = client.Call("IntroducerRPC.ClientJoin", request, &response)
	if err != nil {
		log.Fatal("IntroducerRPC.ClientJoin error: ", err)
	}
	return *response
}

func (c *ClientRPC) JoinCluster(request ClientClusterJoinRequest, response *ClientClusterJoinResponse) error {
	timeStart = time.Now()
	mu.Lock()
	defer mu.Unlock()
	nodeItself = request.NodeItself
	clusterNum = request.ClusterNum
	clusterRepIP = request.ClusterRep.Ip
	isRep = clusterRepIP == myIPStr
	virtRing = &cll.UniqueCLL{}
	virtRing.SetDefaults()
	log.Print("Cluster Membership List: ")
	for _, member := range request.Members {
		virtRing.PushBack(member)
		fmt.Print(member.Uuid.String() + ", ")
	}
	fmt.Println()
	joined = true
	log.Println("Joined Cluster #", clusterNum)
	response.Ack = true
	if nodeItself.NodeType == Driver {
		go sendBid()
	}
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

// func (c *ClientRPC) RecvClusterInfo(clusterInfo types.ClusterInfo, response *types.ClientMainClustererResponse) error {
// 	mu.Lock()
// 	nodeItself = clusterInfo.NodeItself
// 	clusterRep := clusterInfo.ClusterRep
// 	clusterNum = clusterInfo.ClusterNum
// 	fmt.Println("this client got clusterRep and clusterNum assigned as : ", nodeItself, clusterRep, clusterNum)
// 	mu.Unlock()
// 	return nil
// }

// rider receiving a bid
func (c *ClientRPC) RecvBid(bid Bid, response *BidResponse) error {
	auctionState.mu.Lock()
	defer auctionState.mu.Unlock()
	response.Response = true
	if auctionState.acceptedBid { // if the rider already accepted a bid
		response.Accept = false
		log.Println("Bid Rejected, already accepted bid")
		return nil
	}

	if bid.Cost < RiderMaxCost { // if bid is within ddrivere price range
		auctionState.acceptedBid = true
		response.Accept = true
		log.Println("Bid Accepted, cost: ", bid.Cost)
	} else {
		response.Accept = false
		log.Println("Bid Rejected, cost: ", bid.Cost)
	}
	return nil
}

func (c *ClientRPC) RecvDriverInfo(driverInfo types.DriverInfo, response *types.RiderInfo) error {
	log.Println("Driver Info Recv'd: ", driverInfo.PhoneNumber)
	response.Response = true
	response.PhoneNumber = "RiderPhoneNumber"
	// TODO graceful leave system
	log.Println("Time: ", time.Since(timeStart))
	return nil
}

// driver sending a bid to a rider
func sendBid() {
	// get current list of riders in ring
	// mu.Lock()
	biddingPool := virtRing.GetNodes(true)
	// mu.Unlock()
	// order list by cost (low to high)
	sort.Slice(biddingPool, func(i, j int) bool {
		biddingPool[i].Cost = driverCost(nodeItself, biddingPool[i])
		biddingPool[j].Cost = driverCost(nodeItself, biddingPool[j])
		return biddingPool[i].Cost < biddingPool[j].Cost
	})
	// loop thrugh list and send bid to each client
	for _, rider := range biddingPool {
		riderCost := riderCost(nodeItself, rider)
		log.Println("Send bid to ", rider.Uuid.String(), " value: ", fmt.Sprintf("%f", riderCost))
		bidResponse := sendBidRPC(rider, riderCost)
		log.Println("Bid Reponse: ", bidResponse)
		if bidResponse.Response && bidResponse.Accept {
			// driver either accepts or declines rider
			// assume driver auto accept
			driverInfo := types.DriverInfo{PhoneNumber: "drivernumber"}
			riderInfo := sendDriveInfo(rider, driverInfo)
			if riderInfo.Response { // matched, exit system on match
				log.Println("Matched! Rider number: ", riderInfo.PhoneNumber)
				// TODO graceful leave system
				return
			}
		} // if no response or declined bid, try next rider in biddingPool
	}
	// terminate if no match
	// TODO graceful leave system
	return
}

// cost functions and lat/lng distance calculations

// driver cost is distance extra distance driven to pick up passanger
func driverCost(driver types.Node, rider types.Node) float64 {
	dstart_rstart := distanceBetweenCoords(driver.StartLat, driver.StartLng, rider.StartLat, rider.StartLng)
	rstart_rend := distanceBetweenCoords(rider.StartLat, rider.StartLng, rider.DestLat, rider.DestLng)
	// assume driver end is same as riders end dest
	dstart_dend := distanceBetweenCoords(driver.StartLat, driver.StartLng, rider.DestLat, rider.DestLng)
	return (dstart_rstart + rstart_rend) - dstart_dend
}

// rider cost is from their start to end dest + extra distance that the driver had to drive to pick them up
func riderCost(driver types.Node, rider types.Node) float64 {
	return distanceBetweenCoords(rider.StartLat, rider.StartLng, rider.DestLat, rider.DestLng)
}

func distanceBetweenCoords(lat1 float64, lng1 float64, lat2 float64, lng2 float64) float64 {
	earthRadius := 6371.0 // Earth radius in kilometers

	// Convert degrees to radians
	lat1Rad := lat1 * math.Pi / 180
	lng1Rad := lng1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lng2Rad := lng2 * math.Pi / 180

	// Calculate the distance using the Haversine formula
	dLat := lat2Rad - lat1Rad
	dLng := lng2Rad - lng1Rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	distance := earthRadius * c
	return distance
}

////////////////////////////////////////////////////////////////////////////////

func sendBidRPC(node types.Node, cost float64) types.BidResponse {
	conn, err := net.DialTimeout("tcp", node.Ip, constants.BidTimeout)
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		return BidResponse{Response: false}
	}
	defer conn.Close()
	client := rpc.NewClient(conn)
	response := new(types.BidResponse)

	err = client.Call("ClientRPC.RecvBid", types.Bid{Cost: cost}, &response)
	if err != nil {
		log.Fatal("ClientRPC.RecvBid error: ", err)
	}
	return *response
}

func sendDriveInfo(node Node, driverInfo DriverInfo) RiderInfo {
	conn, err := net.DialTimeout("tcp", node.Ip, constants.BidTimeout)
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		return RiderInfo{Response: false}
	}
	defer conn.Close()
	client := rpc.NewClient(conn)
	response := new(RiderInfo)
	err = client.Call("ClientRPC.RecvDriverInfo", driverInfo, &response)
	if err != nil {
		log.Fatal("ClientRPC.RecvDriverInfo error: ", err)
	}
	log.Println("Time: ", time.Since(timeStart))
	return *response
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
	defer conn.Close()
	rpc.Accept(conn)
}

// client requests introduicer
// client gets back cluster number and cluster represnteitnve
// if rep: start taking join requests
// if not rep: send req to cluster rep to join cluster
// virtual ring with pings
