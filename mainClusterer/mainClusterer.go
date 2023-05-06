package main

import (
	"log"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"ridehost/cll"
	"ridehost/constants"
	. "ridehost/constants"
	"ridehost/failureDetector"
	. "ridehost/kMeansClustering"
	. "ridehost/types"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
)

type MainClustererRPC bool

type CoresetList struct {
	mu   sync.Mutex
	cond sync.Cond
	List []Coreset
}

func (c *CoresetList) Append(elem Coreset) {
	c.mu.Lock()
	c.List = append(c.List, elem)
	c.mu.Unlock()
	c.cond.Signal()
}

func (c *CoresetList) Clear() {
	c.mu.Lock()
	c.List = nil
	c.mu.Unlock()
}

// var clusteringNodes = []string{"172.22.155.51:" + strconv.Itoa(Ports["clusteringNode"])}
// var clusteringNodes = []string{"172.22.155.51:" + strconv.Itoa(Ports["clusteringNode"]), "172.22.157.57:" + strconv.Itoa(Ports["clusteringNode"]), "172.22.150.239:" + strconv.Itoa(Ports["clusteringNode"])}

// var clusteringNodes = []string{"0.0.0.0:2235", "0.0.0.0:2238", "0.0.0.0:2239"}

var coresetList CoresetList

var logger = log.New(os.Stdout, "MainClusterer", log.Ldate|log.Ltime)

var ipStr string = getMyIpStr()

var mu = new(sync.Mutex)
var cond = sync.NewCond(mu)
var virtRing = new(cll.UniqueCLL)

func main() {
	coresetList.cond = *sync.NewCond(&coresetList.mu)
	go acceptConnections()
	go startFailureDetector()

	for { // send request to start clustering to all nodes every ClusteringPeriod
		time.Sleep(ClusteringPeriod * time.Minute) // TODO might want a cond var here
		start := time.Now()
		mu.Lock()
		for _, node := range virtRing.GetIPList() {
			go sendStartClusteringRPC(node + ":" + strconv.Itoa(Ports["clusteringNode"]))
		}

		// wait for all coresets to be recvd
		coresetList.mu.Lock()
		for len(coresetList.List) < virtRing.GetSize() {
			coresetList.cond.Wait()
		}
		mu.Unlock()

		// take union of coresets
		clusterNums := coresetUnion()
		end := time.Now()
		logger.Println("Execution time of kmeans: ", end.Sub(start))
		// make cope list and then clear list
		tempcorelist := coresetList.List
		coresetList.List = nil
		coresetList.mu.Unlock()

		// find ClusterRepresentation Info to send to client
		coreunion := map[Node]Node{}
		corelist := tempcorelist
		for _, core := range corelist {
			for repNode, clusterList := range core.Tempcluster {
				for _, node := range clusterList {
					coreunion[node] = repNode
				}
			}
		}

		clusterMemberes := make(map[int][]Node, len(coreunion))
		for k, v := range clusterNums {
			clusterMemberes[v] = append(clusterMemberes[v], k)
		}
		// TODO List of members of cluster
		// send RPCs to clients with their cluster rep ip
		for node, clusterNum := range clusterNums {
			clusterinfo := ClientClusterJoinRequest{NodeItself: node, ClusterRep: coreunion[node], ClusterNum: clusterNum,
				Members: clusterMemberes[clusterNum]}
			go sendClusterInfo(node, clusterinfo)
		}
	}
}

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
	failureDetectorRPC.NodeItself = &Node{Uuid: uuid.New(), Ip: ipStr, NodeType: MainClusterer}
	failureDetectorRPC.Joined = nil
	// failureDetectorRPC.StartPinging = nil
	rpc.Register(failureDetectorRPC)
	conn, err := net.ListenTCP("tcp", address)
	if err != nil {
		log.Fatal("listen error:", err)
	}
	rpc.Accept(conn)
}

func acceptConnections() {
	// get my ip
	address, err := net.ResolveTCPAddr("tcp", "0.0.0.0:"+strconv.Itoa(Ports["mainClusterer"]))
	if err != nil {
		log.Fatal(err)
	}
	mainClustererRPC := new(MainClustererRPC)
	rpc.Register(mainClustererRPC)
	conn, err := net.ListenTCP("tcp", address)
	if err != nil {
		log.Fatal("listen error:", err)
	}
	defer conn.Close()
	rpc.Accept(conn)
}

func (m *MainClustererRPC) ClusteringRequest(request JoinRequest, response *IntroducerMainClustererResponse) error {
	go sendClusteringRPC(request)
	response.Message = "ACK"
	return nil
}

func sendClusteringRPC(request JoinRequest) {
	// randomly chose clusteringNode to forward request to
	seed := rand.NewSource(time.Now().UnixNano())
	r := rand.New(seed)
	mu.Lock()
	clusterNum := r.Intn(virtRing.GetSize())
	conn, err := net.Dial("tcp", virtRing.GetIPList()[clusterNum]+":"+strconv.Itoa(Ports["clusteringNode"]))
	mu.Unlock()
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		return
	}
	defer conn.Close()

	client := rpc.NewClient(conn)
	clusterResponse := new(MainClustererClusteringNodeResponse)

	// send clustering request to clusterNum clustering Node
	if client.Call("ClusteringNodeRPC.Cluster", request, &clusterResponse) != nil {
		os.Stderr.WriteString("ClusteringNodeRPC.Cluster error: " + err.Error())
	}
}

func sendStartClusteringRPC(clusterIp string) {
	logger.Println("Starting clustering at: ", clusterIp)
	conn, err := net.Dial("tcp", clusterIp)
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		return
	}
	defer conn.Close()

	client := rpc.NewClient(conn)
	// var clusterResponse *MainClustererClusteringNodeResponse
	clusterResponse := new(MainClustererClusteringNodeResponse)

	// send start clustering request to clusterNum clustering Node
	if client.Call("ClusteringNodeRPC.StartClustering", 0, &clusterResponse) != nil {
		os.Stderr.WriteString("ClusteringNodeRPC.StartClustering error: " + err.Error())
	}
}

func (m *MainClustererRPC) RecvCoreset(coreset Coreset, response *MainClustererClusteringNodeResponse) error {
	// add coreset to list
	go coresetList.Append(coreset)
	response.Message = "ACK"
	return nil
}

// func WeightedDistance(a Point, b Point, weights []float64) float64 {
// 	var dist float64
// 	for i := range weights {
// 		diff := a.X - b.X
// 		dist += weights[i] * diff * diff
// 		diff = a.Y - b.Y
// 		dist += weights[i] * diff * diff
// 	}
// 	return dist
// }

// func Weight(i int, weights []float64) float64 {
// 	return weights[i]
// }
// func KMeansWeighted(points []Point, weights []float64, k int) []Point {
// 	// Initialize the centroids randomly
// 	centroids := make([]Point, k)
// 	for i := range centroids {
// 		centroids[i] = points[rand.Intn(len(points))]
// 	}
// 	count := 0
// 	// Repeat until convergence
// 	for {
// 		count += 1
// 		// Assign each point to the nearest centroid
// 		assignments := make([]int, len(points))
// 		for i, p := range points {
// 			minDist := WeightedDistance(p, centroids[0], weights)
// 			minIndex := 0
// 			for j := 1; j < k; j++ {
// 				dist := WeightedDistance(p, centroids[j], weights)
// 				if dist < minDist {
// 					minDist = dist
// 					minIndex = j
// 				}
// 			}
// 			assignments[i] = minIndex
// 		}

// 		// Compute the new centroids
// 		newCentroids := make([]Point, k)
// 		counts := make([]float64, k)
// 		for i, p := range points {
// 			index := assignments[i]
// 			counts[index] += Weight(i, weights)
// 			newCentroids[index].X += Weight(i, weights) * p.X
// 			newCentroids[index].Y += Weight(i, weights) * p.Y
// 		}
// 		for i := range newCentroids {
// 			if counts[i] > 0 {
// 				newCentroids[i].X /= counts[i]
// 				newCentroids[i].Y /= counts[i]
// 			}
// 		}

// 		// Check for convergence
// 		converged := true
// 		for i := range centroids {
// 			if centroids[i].X != newCentroids[i].X || centroids[i].Y != newCentroids[i].Y {
// 				converged = false
// 				break
// 			}
// 		}
// 		if converged {
// 			break
// 		}

// 		// Update the centroids
// 		centroids = newCentroids

// 		if count == 500 {
// 			break
// 		}
// 	}

// 	return centroids
// }

func coresetUnion() map[Node]int {
	// outputs Node -> clusterNum
	coreunion := map[Node]int{}
	finalnodemap := map[Node]Node{}
	tempnodemap := map[Node]Node{}
	corelist := coresetList.List

	// for weighted-k-means
	// weights := []float64{}
	// tpoints := []Point{}
	// centroids := []Point{}
	// fmt.Println("corelist returned to main  : ")
	// fmt.Println(corelist)
	// for _, core := range corelist {
	// 	for i := 0; i < len(core.Coreset); i++ {
	// 		tpoints = append(tpoints, core.Coreset[i])
	// 		weights = append(weights, core.Weights[i])
	// 	}
	// }
	// fmt.Println("tpoints :", tpoints)
	// fmt.Println("weights :", weights)
	// // Run the weighted k-means algorithm
	// if len(tpoints) >= NumClusters {
	// 	centroids = KMeansWeighted(tpoints, weights, NumClusters)
	// }

	//create list of nodes from coresets to apply kmeans on them
	coresetnodes := []Node{}
	allnodes := []Node{}
	for _, core := range corelist {
		for i := 0; i < len(core.CoresetNodes); i++ {
			coresetnodes = append(coresetnodes, core.CoresetNodes[i])
		}
		// to find the rest of the nodes other than coresetnodes, we need all nodes
		mp := core.Tempcluster
		for k, v := range mp {
			for j := 0; j < len(v); j++ {
				allnodes = append(allnodes, v[j])
				tempnodemap[v[j]] = k
			}
		}
	}
	logger.Println("length of allnodes: ", len(allnodes))
	logger.Println("length of allnodes: ", len(dedupeList(allnodes)))
	logger.Println("length of coresetnodes: ", len(coresetnodes))
	restnodes := subtractnode(dedupeList(allnodes), coresetnodes)
	logger.Println("length of restnodes: ", len(restnodes))
	clusterresult := ClusterResult{}
	clusterresult = CentralizedKMeansClustering(coresetnodes, NumClusters)
	num := 0

	for n, clusterList := range clusterresult.ClusterMaps {
		for _, node := range clusterList {
			coreunion[node] = num
			finalnodemap[node] = n
		}
		num += 1
	}

	// also add rest of the nodes to finalnodemap and coreunion
	for _, node := range restnodes {
		parent := tempnodemap[node]
		parentnum := coreunion[parent]
		finalnodemap[node] = parent
		coreunion[node] = parentnum
	}

	logger.Println("CoreUnion result: ", len(coreunion), coreunion)

	return coreunion
}
func dedupeList(list []Node) []Node {
	deduped := make([]Node, 0, len(list))
	seen := make(map[Node]bool)

	for _, item := range list {
		if !seen[item] {
			deduped = append(deduped, item)
			seen[item] = true
		}
	}
	return deduped
}
func subtractnode(list1 []Node, list2 []Node) []Node {
	// Input lists

	// Create a map to store the elements of list2
	map2 := make(map[Node]bool)
	for _, num := range list2 {
		map2[num] = true
	}

	// Create a new slice to store the unique elements of list1
	unique := []Node{}
	for _, num := range list1 {
		if !map2[num] {
			unique = append(unique, num)
		}
	}
	return unique
}

// send cluster info to client nodes
func sendClusterInfo(node Node, clusterinfo ClientClusterJoinRequest) {
	conn, err := net.Dial("tcp", node.Ip+":"+strconv.Itoa(constants.Ports["clientRPC"]))
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		return
	}
	defer conn.Close()
	client := rpc.NewClient(conn)
	response := new(ClientClusterJoinResponse)
	logger.Println("Node ", clusterinfo.NodeItself.Uuid.String(), " -> Cluster ", clusterinfo.ClusterNum, " has cluster rep: ", clusterinfo.ClusterRep.Uuid.String())

	err = client.Call("ClientRPC.JoinCluster", clusterinfo, &response)
	if err != nil {
		os.Stderr.WriteString("ClientRPC.ClientJoin error: " + err.Error())
	}
}

// get this machine's IP address
func getMyIpStr() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
	defer conn.Close()
	myIP := conn.LocalAddr().(*net.UDPAddr).IP
	return myIP.String()
}
