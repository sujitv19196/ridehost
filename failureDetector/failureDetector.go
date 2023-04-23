package failureDetector

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
)

type FailureDetectorRPC struct {
	Mu           *sync.Mutex
	VirtRing     *cll.UniqueCLL
	Joined       *bool
	StartPinging *bool
	NodeItself   *types.Node
}

func (fdr *FailureDetectorRPC) StartPings(request types.ClientClusterPingingStatusRequest, response *types.ClientClusterPingingStatusResponse) error {
	fdr.Mu.Lock()
	defer fdr.Mu.Unlock()
	*fdr.StartPinging = request.Status
	log.Printf("pinging status changed to %t\n", *fdr.StartPinging)
	response.Ack = true
	return nil
}

func SendPings(mu *sync.Mutex, joined *bool, startPinging *bool, virtRing *cll.UniqueCLL, myIPStr string, introducerIP string) {
	for {
		neighborIPs := []string{}
		mu.Lock()
		if *joined && *startPinging {
			neighborIPs = virtRing.GetNeighbors(myIPStr)
		}
		mu.Unlock()
		var wg sync.WaitGroup
		wg.Add(len(neighborIPs))
		for _, neighborIP := range neighborIPs {
			go AttemptPings(neighborIP, &wg, mu, virtRing, myIPStr, introducerIP)
		}
		wg.Wait()
		// ping every second
		time.Sleep(constants.PingFrequency)
	}
}

func AttemptPings(neighborIP string, wg *sync.WaitGroup, mu *sync.Mutex, virtRing *cll.UniqueCLL, myIPStr string, introducerIP string) {
	defer wg.Done()
	if !sendPing(neighborIP) {
		for i := 0; i < 2; i++ {
			time.Sleep(time.Duration(math.Pow(2, float64(i+1))) * time.Second)
			if sendPing(neighborIP) {
				return
			}
		}
		RemoveNode(neighborIP, mu, virtRing, myIPStr, introducerIP)
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

func AcceptPings(myIP net.IP, mu *sync.Mutex, joined *bool) {
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
			return *joined
		}())
	}
}

func (fdr *FailureDetectorRPC) SendNodeFailure(request types.ClusterNodeRemovalRequest, response *types.ClusterNodeRemovalResponse) error {
	fdr.Mu.Lock()
	defer fdr.Mu.Unlock()
	fdr.VirtRing.RemoveNode(request.NodeIP)
	log.Printf("removing node %s\n", request.NodeIP)
	// removing self
	if request.NodeIP == fdr.NodeItself.Ip {
		*fdr.Joined = false
		log.Println("left cluster")
	}
	response.Ack = true
	return nil
}

func RemoveNode(nodeIP string, mu *sync.Mutex, virtRing *cll.UniqueCLL, myIPStr string, introducerIP string) {
	mu.Lock()
	// get list before removing node so failed node gets rpc saying it failed
	IPs := virtRing.GetIPList()
	virtRing.RemoveNode(nodeIP)
	mu.Unlock()
	if len(introducerIP) > 0 {
		IPs = append(IPs, introducerIP)
	}
	go sendListRemoval(nodeIP, IPs, myIPStr)
	log.Printf("removed node %s\n", nodeIP)
}

func sendListRemoval(neighborIp string, IPs []string, myIPStr string) {
	for _, ip := range IPs {
		if ip != myIPStr {
			go func(ip string, neighborIp string) {
				conn, err := net.DialTimeout("tcp", ip+":"+strconv.Itoa(constants.Ports["failureDetector"]), constants.TCPTimeout)
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
				err = client.Call("FailureDetectorRPC.SendNodeFailure", request, response)
				if err != nil {
					log.Fatal("SendNodeFailure error: ", err)
				}
				conn.Close()
			}(ip, neighborIp)
		}
	}
}
