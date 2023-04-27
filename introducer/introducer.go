package main

import (
	"log"
	"net"
	"net/rpc"
	"os"
	"ridehost/cll"
	"ridehost/constants"
	. "ridehost/constants"
	"ridehost/failureDetector"
	. "ridehost/types"
	"strconv"
	"sync"
)

var ip net.IP

// VM 2
var mainClustererIp = "172.22.153.8:" + strconv.Itoa(Ports["mainClusterer"]) // TODO can hard code for now
// var mainClustererIp = "0.0.0.0:" + strconv.Itoa(Ports["mainClusterer"]) // TODO can hard code for now

type IntroducerRPC bool

var mu = new(sync.Mutex)
var cond = sync.NewCond(mu)
var virtRing = new(cll.UniqueCLL)
var curClusteringNode int

func main() {
	// get this machine's IP address
	address, err := net.ResolveTCPAddr("tcp", "0.0.0.0:"+strconv.Itoa(Ports["introducer"]))
	if err != nil {
		log.Fatal(err)
	}
	introducerRPC := new(IntroducerRPC)
	rpc.Register(introducerRPC)
	conn, err := net.ListenTCP("tcp", address)
	if err != nil {
		log.Fatal("listen error:", err)
	}

	curClusteringNode = 0

	startFailureDetector()

	rpc.Accept(conn)
}

// RPC exectued by introducer when new joins occur
func (i *IntroducerRPC) ClientJoin(request JoinRequest, response *ClientIntroducerResponse) error {
	// take the requests of the cliient and imediately send to mainClusterer
	go forwardRequestToClusterer(request)
	response.Message = "ACK"
	response.IsClusteringNode = false
	// TODO add error?
	mu.Lock()
	if virtRing.GetSize() < MaxCNs && request.NodeRequest.NodeType == Driver {
		response.IsClusteringNode = true
	}
	mu.Unlock()
	return nil
}

// RPC that client calls to let Introducer know it is ready to recv clsutering requests
// func (i *IntroducerRPC) CNReady(request ClientReadyRequest, response *ClientIntroducerResponse) error {
// 	mu.Lock()
// 	virtRing.PushBack(request.RequestingNode)
// 	mu.Unlock()
// 	response.Message = "ACK"
// 	response.IsClusteringNode = true
// 	log.Println(request.RequestingNode.Uuid.String(), " added to CN pool")
// 	return nil
// }

func startFailureDetector() {
	address, err := net.ResolveTCPAddr("tcp", "0.0.0.0:"+strconv.Itoa(constants.Ports["failureDetector"]))
	if err != nil {
		log.Fatal("listen error:", err)
	}

	mu.Lock()
	virtRing.SetDefaults()
	mu.Unlock()
	failureDetectorRPC := new(failureDetector.FailureDetectorRPC)
	failureDetectorRPC.Mu = mu
	failureDetectorRPC.Cond = cond
	failureDetectorRPC.VirtRing = virtRing
	failureDetectorRPC.NodeItself = nil
	failureDetectorRPC.Joined = nil
	// failureDetectorRPC.StartPinging = nil
	rpc.Register(failureDetectorRPC)
	conn, err := net.ListenTCP("tcp", address)
	if err != nil {
		log.Fatal("listen error:", err)
	}
	go rpc.Accept(conn)
}

func forwardRequestToClusterer(request JoinRequest) {
	mu.Lock()
	for virtRing.GetSize() <= 0 {
		cond.Wait()
	}
	clustererList := virtRing.GetNodes(false)
	curClusteringNode = curClusteringNode % len(clustererList)
	clustererIP := clustererList[curClusteringNode].Ip
	curClusteringNode = (curClusteringNode + 1) % len(clustererList)
	mu.Unlock()

	conn, err := net.Dial("tcp", clustererIP+strconv.Itoa(constants.Ports["clusteringNode"]))
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		return
	}
	client := rpc.NewClient(conn)
	clustererResponse := new(MainClustererClusteringNodeResponse)
	err = client.Call("ClusteringNodeRPC.Cluster", request, &clustererResponse)
	if err != nil {
		os.Stderr.WriteString("ClusteringNodeRPC.Cluster error: " + err.Error())
	}
}

func forwardRequestToMainClusterer(request JoinRequest) {
	conn, err := net.Dial("tcp", mainClustererIp)
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		return
	}
	client := rpc.NewClient(conn)
	mainClustererResponse := new(IntroducerMainClustererResponse)
	err = client.Call("MainClustererRPC.ClusteringRequest", request, &mainClustererResponse)
	if err != nil {
		os.Stderr.WriteString("MainClustererRPC.ClusteringRequest error: " + err.Error())
	}
}
