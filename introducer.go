package main

import (
	"math/rand"
	"net"
	"os"
	unsafe "unsafe"
)

var ip net.IP
var numClusterNodes = 0
var ports = map[string]int{"introducer": 2233,
	"clusterNode": 2234}

type NodeType int

const (
	Driver     NodeType = 0
	Rider      NodeType = 1
	Stationary NodeType = 2
)

type request struct {
	requestType NodeType
	uuid        [16]byte
	lat         float64
	lng         float64
}

func main() {
	// get this machine's IP address
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
	ip = conn.LocalAddr().(*net.UDPAddr).IP
	conn.Close()

	go acceptClients() // start accepting clients
}

func acceptClients() {
	// listen for join requests
	buffer := make(request, unsafe.Sizeof(request))
	addr := net.UDPAddr{
		Port: ports["acceptPings"],
		IP:   ip,
	}
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
	defer conn.Close()

	// read join request

	bytes_read, addr, err := conn.ReadFromUDP(buffer)
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		continue
	}

	// handle join
	go func(conn *net.UDPConn, addr *net.UDPAddr) {
		var err error

		// randomly assign to 1 or N cluster procs
		clusterNum := rand.Intn(numClusterNodes)

		// send node info to cluster ClusterNum
		// TODO goroutine

		// send ACK to requesting node
		_, err = conn.WriteToUDP([]byte("ACK"), addr)
		if err != nil {
			os.Stderr.WriteString(err.Error() + "\n")
		}
	}(conn, addr)

}
