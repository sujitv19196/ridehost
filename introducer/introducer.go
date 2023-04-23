package main

import (
	"log"
	"net"
	"net/rpc"
	"os"
	. "ridehost/constants"
	. "ridehost/types"
	"strconv"
)

var ip net.IP

// VM 2
var mainClustererIp = "172.22.153.8:" + strconv.Itoa(Ports["mainClusterer"]) // TODO can hard code for now
// var mainClustererIp = "0.0.0.0:" + strconv.Itoa(Ports["mainClusterer"]) // TODO can hard code for now

type IntroducerRPC bool

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
	rpc.Accept(conn)
}

// RPC exectued by introducer when new joins occur
func (i *IntroducerRPC) ClientJoin(request JoinRequest, response *ClientIntroducerResponse) error {
	// take the requests of the cliient and imediately send to mainClusterer
	go forwardRequestToMainClusterer(request)
	response.Message = "ACK"
	// TODO add error?
	return nil
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
