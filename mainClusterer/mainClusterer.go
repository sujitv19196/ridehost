package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	. "ridehost/constants"
	. "ridehost/kMeansClustering"
	. "ridehost/types"
	"strconv"
	"sync"
	"time"
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

var clusteringNodes = []string{"172.22.155.51:" + strconv.Itoa(Ports["clusteringNode"])}

// var clusteringNodes = []string{"0.0.0.0:2235", "0.0.0.0:2238", "0.0.0.0:2239"}

var coresetList CoresetList

func main() {
	coresetList.cond = *sync.NewCond(&coresetList.mu)
	go acceptConnections()

	for { // send request to start clustering to all nodes every ClusteringPeriod
		time.Sleep(ClusteringPeriod * time.Minute) // TODO might want a cond var here
		start := time.Now()
		for _, node := range clusteringNodes {
			sendStartClusteringRPC(node)
		}

		// wait for all coresets to be recvd
		coresetList.mu.Lock()
		for len(coresetList.List) < len(clusteringNodes) {
			coresetList.cond.Wait()
		}
		// take union of coresets
		clusterNums := coresetUnion()
		end := time.Now()
		fmt.Println("execution time of kmeans: ", end.Sub(start))
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

		// send RPCs to clients with their cluster rep ip
		for node, clusterNum := range clusterNums {
			clusterinfo := ClusterInfo{NodeItself: node, ClusterRep: coreunion[node], ClusterNum: clusterNum}
			go sendClusterInfo(node, clusterinfo)
		}
	}
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
	clusterNum := r.Intn(len(clusteringNodes))
	fmt.Println("send cluster req to: ", clusteringNodes[clusterNum])
	conn, err := net.Dial("tcp", clusteringNodes[clusterNum])
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		return
	}

	client := rpc.NewClient(conn)
	clusterResponse := new(MainClustererClusteringNodeResponse)

	// send clustering request to clusterNum clustering Node
	if client.Call("ClusteringNodeRPC.Cluster", request, &clusterResponse) != nil {
		os.Stderr.WriteString("ClusteringNodeRPC.Cluster error: " + err.Error())
	}
}

func sendStartClusteringRPC(clusterIp string) {
	fmt.Println("start clustering at: ", clusterIp)
	conn, err := net.Dial("tcp", clusterIp)
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		return
	}

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
	restnodes := subtractnode(allnodes, coresetnodes)

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

	fmt.Println("clusterresult result: ", clusterresult.ClusterMaps)
	fmt.Println("CoreUnion result: ", len(coreunion), coreunion)

	return coreunion
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

	fmt.Println(unique) // Output: [1 3 5]
	return unique
}

// send cluster info to client nodes
func sendClusterInfo(node Node, clusterinfo ClusterInfo) {
	fmt.Println("Send cluster info to: ", node.Ip)
	conn, err := net.Dial("tcp", node.Ip)
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		return
	}
	client := rpc.NewClient(conn)
	response := new(ClientMainClustererResponse)
	fmt.Println("Node ", string(clusterinfo.NodeItself.Uuid[:]), "-> Clsuter ", clusterinfo.ClusterNum, "has cluster rep: ", string(clusterinfo.ClusterRep.Uuid[:]))

	err = client.Call("ClientRPC.RecvClusterInfo", clusterinfo, &response)
	if err != nil {
		os.Stderr.WriteString("IntroducerRPC.ClientJoin error: " + err.Error())
	}
}
