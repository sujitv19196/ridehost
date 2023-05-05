package main

import (
	"fmt"
	"log"
	"math"
	"net"
	"net/rpc"
	"os"
	"ridehost/cll"
	"ridehost/clusteringNode"
	"ridehost/constants"
	. "ridehost/constants"
	"ridehost/types"
	. "ridehost/types"
	"sort"
	"strconv"
	"sync"

	"github.com/google/uuid"
)

type ClientRPC struct{} // RPC

// rider auction state
type RiderAuctionState struct {
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
var clusterRepIP string
var clusterNum int
var nodeItself types.Node
var introducerIp string

var riderAuctionState RiderAuctionState

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

	introducerIp = os.Args[2]

	myIP = conn.LocalAddr().(*net.UDPAddr).IP
	myIPStr = myIP.String()
	conn.Close()

	clientIp = myIPStr
	fmt.Println(clientIp)
	uuid := uuid.New()
	nodeType, _ := strconv.Atoi(os.Args[1])

	startLat, _ := strconv.ParseFloat(os.Args[3], 64)
	startLng, _ := strconv.ParseFloat(os.Args[4], 64)
	req := JoinRequest{}
	if nodeType == Rider {
		destLat, _ := strconv.ParseFloat(os.Args[5], 64)
		destLng, _ := strconv.ParseFloat(os.Args[6], 64)
		nodeItself = Node{NodeType: nodeType, Ip: clientIp, Uuid: uuid, StartLat: startLat, StartLng: startLng, DestLat: destLat, DestLng: destLng}
		req = JoinRequest{NodeRequest: nodeItself, IntroducerIp: introducerIp}
	} else {
		nodeItself = Node{NodeType: nodeType, Ip: clientIp, Uuid: uuid, StartLat: startLat, StartLng: startLng}
		req = JoinRequest{NodeRequest: nodeItself, IntroducerIp: introducerIp}
	}
	r := joinSystem(req)
	log.Println("From Introducer: ", r.Message)

	mu = sync.Mutex{}
	mu.Lock()
	joined = false
	mu.Unlock()

	riderAuctionState = RiderAuctionState{mu: sync.Mutex{}, acceptedBid: false}
	// acceptClusteringConnections()
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
	if response.IsClusteringNode {
		log.Println("Assigned as CN, spinning up CN proc")
		go clusteringNode.Start(nodeItself, introducerIp)
	}
	return *response
}

func (c *ClientRPC) JoinCluster(request ClientClusterJoinRequest, response *ClientClusterJoinResponse) error {
	mu.Lock()
	defer mu.Unlock()
	// nodeItself = request.NodeItself
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

// rider receiving a bid
func (c *ClientRPC) RecvBid(bid Bid, response *BidResponse) error {
	riderAuctionState.mu.Lock()
	defer riderAuctionState.mu.Unlock()
	response.Response = true
	if riderAuctionState.acceptedBid { // if the rider already accepted a bid
		response.Accept = false
		log.Println("Bid Rejected, already accepted bid")
		return nil
	}

	if bid.Cost < RiderMaxCost { // if bid is within ddrivere price range
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
	return nil
}

// driver sending a bid to a rider
func sendBid() {
	// get current list of riders in ring
	mu.Lock()
	biddingPool := virtRing.GetNodes(true)
	mu.Unlock()
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
				return
			}
		} // if no response or declined bid, try next rider in biddingPool
	}
	// terminate if no match
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
	conn, err := net.DialTimeout("tcp", node.Ip+":"+strconv.Itoa(constants.Ports["clientRPC"]), constants.BidTimeout)
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
	conn, err := net.DialTimeout("tcp", node.Ip+":"+strconv.Itoa(constants.Ports["clientRPC"]), constants.BidTimeout)
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
// client gets back cluster number and membership list
// start bidding
