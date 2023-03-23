package introducer

import (
	"log"
	"math/rand"
	"net"
	"net/rpc"
	"ridehost/types"
	"strconv"
	"time"
)

var ip net.IP
var numClusterNodes = 0
var ports = map[string]int{"introducer": 2233,
	"clusterNode": 2234}

type AcceptClient bool

func main() {
	// get this machine's IP address
	address, err := net.ResolveTCPAddr("tcp", "0.0.0.0:"+strconv.Itoa(ports["introducer"]))
	if err != nil {
		log.Fatal(err)
	}
	acceptClient := new(AcceptClient)
	rpc.Register(acceptClient)
	conn, err := net.ListenTCP("tcp", address)
	if err != nil {
		log.Fatal("listen error:", err)
	}
	rpc.Accept(conn)
}

// RPC exectued by introducer when new joins occur
func (a *AcceptClient) clientJoin(request types.AcceptClientRequest, response *types.AcceptClientResponse) error {
	// take the request of the cliient

	// forward request to crandom lsuter node to kmeans
	seed := rand.NewSource(time.Now().UnixNano())
	r := rand.New(seed)
	clusterNum := r.Intn(numClusterNodes)

	// get clsuter group back from cluster node
	// give repsonse to client
}
