package clusteringNode

import (
	"log"
	"math"
	"net"
	"net/rpc"
	"os"
	"ridehost/cll"
	"ridehost/constants"
	. "ridehost/constants"
	"ridehost/failureDetector"
	. "ridehost/kMeansClustering"
	"ridehost/types"
	. "ridehost/types"
	"strconv"
	"sync"

	"gonum.org/v1/gonum/stat/sampleuv"
)

type MembershipList struct {
	mu   sync.Mutex
	List []Node
}

func (m *MembershipList) Append(elem Node) {
	m.mu.Lock()
	m.List = append(m.List, elem)
	m.mu.Unlock()
}

func (m *MembershipList) Clear() {
	m.mu.Lock()
	m.List = nil
	m.mu.Unlock()
}

var ip net.IP
var numClusterNodes = 0
var ML MembershipList

// VM 2
var mainClustererIp = constants.MainClustererIp + ":" + strconv.Itoa(Ports["mainClusterer"]) // TODO can hard code for now
// var mainClustererIp = "0.0.0.0:" + strconv.Itoa(Ports["mainClusterer"]) // TODO can hard code for now
var introducerIp string
var nodeItself Node

type ClusteringNodeRPC bool

// VM 3, 4, 5
// var clusteringNodes = []string{"172.22.155.51:" + strconv.Itoa(Ports["clusteringNode"]), "172.22.157.57:" + strconv.Itoa(Ports["clusteringNode"]), "172.22.150.239:" + strconv.Itoa(Ports["clusteringNode"])}

var mu = new(sync.Mutex)
var virtualRing = new(cll.UniqueCLL)
var joined = new(bool)
var startPinging = new(bool)

var clusteringNodesResponseList ResponseList
var logger = log.New(os.Stdout, "ClusteringNode ", log.Ldate|log.Ltime)

// accepts connections from main clusterer
// Cluster: takes cluster request and adds to membership list
// StartClustering: performs coreset calculation on current membership list. Locks list until done and new requests are queeue'd.
func Start(thisNode Node, introducer string) {
	mu.Lock()
	virtualRing.SetDefaults()
	nodeItself = thisNode
	introducerIp = introducer
	mu.Unlock()
	go acceptConnections()
	startFailureDetector()
}

func startFailureDetector() {
	address, err := net.ResolveTCPAddr("tcp", "0.0.0.0:"+strconv.Itoa(constants.Ports["failureDetector"]))
	if err != nil {
		log.Fatal("listen error:", err)
	}
	failureDetectorRPC := new(failureDetector.FailureDetectorRPC)
	failureDetectorRPC.Mu = mu
	failureDetectorRPC.NodeItself = &nodeItself
	failureDetectorRPC.VirtRing = virtualRing
	*joined = true
	failureDetectorRPC.Joined = joined
	// failureDetectorRPC.StartPinging = startPinging
	rpc.Register(failureDetectorRPC)
	conn, err := net.ListenTCP("tcp", address)
	if err != nil {
		log.Fatal("listen error:", err)
	}
	tellIntroducerFDReady()
	go failureDetector.SendPings(mu, joined, virtualRing, nodeItself.Ip, []string{introducerIp, constants.MainClustererIp}, nodeItself.Uuid.String())
	go failureDetector.AcceptPings(ip, mu, joined)
	tellIntroducerFDPingingReady()
	rpc.Accept(conn)
}

func tellIntroducerFDPingingReady() {
	conn, err := net.Dial("tcp", introducerIp+":"+strconv.Itoa(constants.Ports["failureDetector"]))
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
	defer conn.Close()

	client := rpc.NewClient(conn)
	response := new(types.NodeFailureDetectingPingingStatusRes)
	nodeItself.PingReady = true
	virtualRing.GetNode(nodeItself.Uuid.String()).PingReady = true
	err = client.Call("FailureDetectorRPC.StartPingingNode", NodeFailureDetectingPingingStatusReq{Uuid: nodeItself.Uuid.String(), Ip: nodeItself.Ip, Status: true}, &response)
	if err != nil {
		log.Fatal("FailureDetectorRPC.StartPingingNode error: ", err)
	}
}

func tellIntroducerFDReady() {
	conn, err := net.Dial("tcp", introducerIp+":"+strconv.Itoa(constants.Ports["failureDetector"]))
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
	defer conn.Close()

	client := rpc.NewClient(conn)
	response := new(types.IntroducerNodeAddResponse)
	err = client.Call("FailureDetectorRPC.IntroducerAddNode", IntroducerNodeAddRequest{NodeToAdd: nodeItself}, &response)
	if err != nil {
		log.Fatal("FailureDetectorRPC.IntroducerAddNode error: ", err)
	}
	log.Println("Tell Introdcuer FDR Ready")
	for _, node := range response.Members {
		virtualRing.PushBack(node)
	}
}

// sends RPC to introdcuer to indicate that it is ready to recv clustering requests
// func tellIntroducerReady() {
// 	conn, err := net.Dial("tcp", introducerIp+":"+strconv.Itoa(constants.Ports["introducer"]))
// 	if err != nil {
// 		os.Stderr.WriteString(err.Error() + "\n")
// 		os.Exit(1)
// 	}
// 	defer conn.Close()

// 	client := rpc.NewClient(conn)
// 	response := new(types.ClientIntroducerResponse)
// 	err = client.Call("IntroducerRPC.CNReady", ClientReadyRequest{RequestingNode: nodeItself}, &response)
// 	if err != nil {
// 		log.Fatal("IntroducerRPC.CNReady error: ", err)
// 	}
// }

func acceptConnections() {
	// get this machine's IP address
	address, err := net.ResolveTCPAddr("tcp", "0.0.0.0:"+strconv.Itoa(Ports["clusteringNode"]))
	if err != nil {
		log.Fatal(err)
	}
	clusteringNodeRPC := new(ClusteringNodeRPC)
	rpc.Register(clusteringNodeRPC)
	conn, err := net.ListenTCP("tcp", address)
	if err != nil {
		log.Fatal("listen error:", err)
	}
	defer conn.Close()
	// tellIntroducerReady()
	rpc.Accept(conn)
}

// cluster node accepts an RPC call from client node,
// get the cluster using the kmeans clustering function and return.
func (c *ClusteringNodeRPC) Cluster(request JoinRequest, response *MainClustererClusteringNodeResponse) error {
	logger.Println("request from: ", request.NodeRequest.Uuid.String())
	go ML.Append(request.NodeRequest)
	response.Message = "ACK"
	return nil
}

func (c *ClusteringNodeRPC) StartClustering(nouse int, response *MainClustererClusteringNodeResponse) error {
	logger.Println("Membership List: ", len(ML.List))
	for _, elem := range ML.List {
		logger.Print(elem.Uuid.String())
	}
	go func() {
		coreset := Coreset{}

		// lock list until clustering is done (will still accept new requests which will be added after clsutering)
		ML.mu.Lock()
		if len(ML.List) >= NumClusters {
			coreset = kMeansClustering()
			// clear list after clutsering
			ML.List = nil
		} else { // call send CostMsg so that other clustering nodes stop waiting
			add, _ := net.LookupIP("ispycode.com")

			mu.Lock()
			for _, node := range virtualRing.GetNodes(false) {
				nodeIP := node.Ip
				if nodeIP != add[0].String() {
					go sendCostMsg(nodeIP, 0.0, 0)
				}
			}
			mu.Unlock()
		}

		ML.mu.Unlock()
		// RPC call
		sendCoreset(coreset)
	}()
	response.Message = "ACK"
	return nil
}

func kMeansClustering() Coreset {
	// Calling the centralized K means clustering for current implementation
	// it returns cluster type and we will create a coreset type from it
	// clusterresult := ClusterResult{}
	// clusterresult = CentralizedKMeansClustering(ML.List, NumClusters)
	// coreset := Coreset{Coreset: []Point{}, CoresetNodes: []Node{}, Tempcluster: clusterresult.ClusterMaps}
	coreset := Coreset{}
	coreset = IndividualKMeansClustering(ML.List, NumClusters)
	return coreset
}

// calls MainClustererRPC.RecvCoreset to give it computed coreset
func sendCoreset(coreset Coreset) {
	conn, err := net.Dial("tcp", mainClustererIp)
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		return
	}
	defer conn.Close()

	client := rpc.NewClient(conn)
	clusterResponse := new(MainClustererClusteringNodeResponse)

	if client.Call("MainClustererRPC.RecvCoreset", coreset, &clusterResponse) != nil {
		os.Stderr.WriteString("MainClustererRPC.RecvCoreset error: " + err.Error())
	}
}

type ResponseList struct {
	mu   sync.Mutex
	cond sync.Cond
	List []CostMsg
}

func (c *ResponseList) Append(elem CostMsg) {
	c.mu.Lock()
	c.List = append(c.List, elem)
	c.mu.Unlock()
	c.cond.Signal()
}

func (c *ResponseList) Clear() {
	c.mu.Lock()
	c.List = nil
	c.mu.Unlock()
}

func (m *ClusteringNodeRPC) RecvCostMsg(data CostMsg, response *ClusteringNodeClusteringNodeResponse) error {
	// add data to list
	go clusteringNodesResponseList.Append(data)
	response.Message = "ACK"
	return nil
}

// calls ClusteringNodes.RecvCostMsg to give it computed cost and length of membershiplist
func sendCostMsg(nodeIP string, cost float64, length int) {
	conn, err := net.Dial("tcp", nodeIP+":"+strconv.Itoa(Ports["clusteringNode"]))
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		return
	}

	client := rpc.NewClient(conn)
	costResponse := new(ClusteringNodeClusteringNodeResponse)
	data := CostMsg{NodeIp: nodeIP, Cost: cost, LengthML: length}
	logger.Println("Data to send to other clustering node : ", data)
	if err := client.Call("ClusteringNodeRPC.RecvCostMsg", data, &costResponse); err != nil {
		os.Stderr.WriteString("ClusteringNodeRPC.RecvCostMsg error: " + err.Error())
	}
}

// K MEANS CLUSTERING

func IndividualKMeansClustering(Nodelist []Node, k int) Coreset {

	clusterresults := CentralizedKMeansClustering(Nodelist, k)
	logger.Println("centroids from centralized k means: ", clusterresults.Centroids)

	centroids := clusterresults.Centroids
	data := []Point{}
	for i := 0; i < len(Nodelist); i++ {
		lat := Nodelist[i].StartLat
		lng := Nodelist[i].StartLng
		pair := Point{X: lat, Y: lng}
		data = append(data, pair)
	}
	//step 2: calculate approximate costs C(Pi, Bi)
	costList := []float64{}
	cost := 0.
	for _, point := range data {
		min_dis := 9999999999999999999999.
		for _, center := range centroids {
			cur_dis := euclideanDistance(point, center)
			if cur_dis < min_dis {
				min_dis = cur_dis
			}
		}
		costList = append(costList, min_dis)
		cost += min_dis * min_dis
	}
	logger.Println("cost for this clustering node : ", cost)
	logger.Println("costList for this clustering node : ", costList)

	// step 3: TODO Communicate cost to all other clustering nodes via RPC
	// also communicate length of membership lists to calculate t

	//k-means-clustering calls this in step 3.
	// get this machine's IP address
	add, _ := net.LookupIP("ispycode.com")
	// add := []string{"0.0.0.0:2235"}
	currMLLen := len(ML.List)
	mu.Lock()
	for _, node := range virtualRing.GetNodes(false) {
		nodeIP := node.Ip
		if nodeIP != add[0].String() {
			go sendCostMsg(nodeIP, cost, currMLLen)
		}
	}
	mu.Unlock()
	clusteringNodesResponseList.cond = *sync.NewCond(&clusteringNodesResponseList.mu)
	// wait for all responses to be recvd
	clusteringNodesResponseList.mu.Lock()
	mu.Lock()
	for len(clusteringNodesResponseList.List) < virtualRing.GetSize()-1 {
		clusteringNodesResponseList.cond.Wait()
	}
	mu.Unlock()

	// as a result from step 3
	tempclusteringNodesResponseList := clusteringNodesResponseList.List
	clusteringNodesResponseList.List = nil
	clusteringNodesResponseList.mu.Unlock()

	costFromAllClusteringNodes := []float64{}
	t := 0
	for _, response := range tempclusteringNodesResponseList {
		costFromAllClusteringNodes = append(costFromAllClusteringNodes, response.Cost)
		t = t + response.LengthML
	}
	t = t + len(ML.List) //add the length of ML of the node itself.
	logger.Println("value of t from all clustering nodes: ", t)
	T := int(float64(t) / DivisorT)
	logger.Println("costFromAllClusteringNodes: ", costFromAllClusteringNodes)
	sumOfCostOfAllClusteringNodes := 0.

	for _, val := range costFromAllClusteringNodes {
		sumOfCostOfAllClusteringNodes += val
	}
	sumOfCostOfAllClusteringNodes += cost
	// Round 2 step 1 : Compute ti
	ti := int(math.Floor((float64(T) * cost) / sumOfCostOfAllClusteringNodes))
	logger.Println("value of ti : ", ti)
	if ti == 0 {
		ti = 1
	} else if ti >= currMLLen {
		ti = currMLLen - 1
	}
	logger.Println("updated value of ti : ", ti)
	// ti = 4
	// Round 2 step 2: multiply each cost by 2 to get mp
	mp := []float64{}

	for i := 0; i < len(costList); i++ {
		mp = append(mp, 2*costList[i]+1e-31)
	}

	mpProb := []float64{}

	for i := 0; i < len(costList); i++ {
		mpProb = append(mpProb, (costList[i]+1e-31)/cost)
	}
	logger.Println("Finished round 2 step 2 ")

	// Round 2 step 3: Non-uniform random sample 𝑆𝑖 of 𝑡𝑖 points from Pi, where for every 𝑞 ∈ 𝑃𝑖.
	// Also find weights on q, wq

	w := sampleuv.NewWeighted(mpProb, nil)
	q_indexes := []int{}
	qp := []Point{}
	for i := 0; i < ti; i++ {
		index, _ := w.Take()
		q_indexes = append(q_indexes, index)
		qp = append(qp, data[index])
	}

	wq := []float64{}
	num := 0.
	// add the mps of all other clustering nodes to the num.
	// since mp=2*cost, the summation mp = 2* sum of all cost from all clustering nodes
	num += (sumOfCostOfAllClusteringNodes * 2)

	for _, index := range q_indexes {

		denom := mp[index] * float64(T)
		wq = append(wq, num/denom)
	}
	logger.Println("Finished round 2 step 3 ")

	// Round 2 step 4: Local k-centers (𝐵𝑖 ) are included to the coreset
	// Also find Weight on b, wb

	core := Coreset{}
	coreset := qp
	for _, point := range clusterresults.RepresentCenterPoints {
		coreset = append(coreset, point)
	}
	weights := wq
	wb := []float64{}
	for _, centroid := range centroids {
		// coreset = append(coreset, centroid)
		pb := []Point{}
		pb_indexes := []int{}
		for idx, point := range data {
			euc_dist := euclideanDistance(centroid, point)
			if euc_dist == costList[idx] {
				pb = append(pb, point)
				pb_indexes = append(pb_indexes, idx)
			}
		}

		// Finding Pb 𝑃 = {𝑝∈𝑃: 𝑑(𝑝,𝑏) = 𝑑(𝑝,𝐵)}
		intersectionOfPbAndS := intersect(pb_indexes, q_indexes)
		logger.Println("Round 2 step 4: checking (pb_indexes, q_indexes, wq) ", pb_indexes, q_indexes, wq)

		sumOfWqs := 0.
		for _, in := range intersectionOfPbAndS {
			sumOfWqs += wq[in]
		}
		wb = append(wb, float64(len(pb))-sumOfWqs)
		weights = append(weights, float64(len(pb))-sumOfWqs)
	}
	logger.Println("Finished round 2 step 4 ")

	// logger.Println("Final data : ", data)
	// logger.Println("Final coreset : ")
	// logger.Println("b : ", len(centroids), centroids)
	// logger.Println("bp : ", len(clusterresults.RepresentCenterPoints), clusterresults.RepresentCenterPoints)
	// logger.Println("qp : ", len(qp), qp)
	// logger.Println("Final weights : ")
	// logger.Println("wb : ", len(wb), wb)
	// logger.Println("wq : ", len(wq), wq)

	core.Coreset = coreset
	core.Weights = weights
	core.CoresetNodes = clusterresults.RepresentCenterNodes
	core.Tempcluster = clusterresults.ClusterMaps

	logger.Println("Resultant Local Coreset : ", core)
	return core

}
func intersect(slice1, slice2 []int) []int {
	var intersect []int
	for _, element1 := range slice1 {
		for idx, element2 := range slice2 {
			if element1 == element2 {
				intersect = append(intersect, idx)
			}
		}
	}
	return intersect //return slice after intersection
}

func euclideanDistance(p1, p2 Point) float64 {
	return math.Sqrt(math.Pow(p1.X-p2.X, 2) + math.Pow(p1.Y-p2.Y, 2))
}

// func main() {
// 	acceptConnections()
// n1 := Node{NodeType: Driver, Ip: "35", Uuid: uuid.New(), Lat: 24, Lng: 25}
// n2 := Node{NodeType: Driver, Ip: "890", Uuid: uuid.New(), Lat: 1, Lng: 2}
// n3 := Node{NodeType: Driver, Ip: "234", Uuid: uuid.New(), Lat: 12, Lng: 13}
// n4 := Node{NodeType: Driver, Ip: "trfh", Uuid: uuid.New(), Lat: 5, Lng: 6}
// n5 := Node{NodeType: Driver, Ip: "5676", Uuid: uuid.New(), Lat: 6, Lng: 5}
// n6 := Node{NodeType: Driver, Ip: "23478", Uuid: uuid.New(), Lat: 1, Lng: 1}
// n7 := Node{NodeType: Driver, Ip: "1279", Uuid: uuid.New(), Lat: 15, Lng: 15}
// n8 := Node{NodeType: Driver, Ip: "11234", Uuid: uuid.New(), Lat: 16, Lng: 19}
// n9 := Node{NodeType: Driver, Ip: "qwerty", Uuid: uuid.New(), Lat: 4, Lng: 8}
// nlist := []Node{}
// nlist = append(nlist, n1)
// nlist = append(nlist, n2)
// nlist = append(nlist, n3)
// nlist = append(nlist, n4)
// nlist = append(nlist, n5)
// nlist = append(nlist, n6)
// nlist = append(nlist, n7)
// nlist = append(nlist, n8)
// nlist = append(nlist, n9)

// // clusterresult := CentralizedKMeansClustering(nlist, 3)
// // fmt.Println("clusterresult", clusterresult)
// clusterresult := IndividualKMeansClustering(nlist, 3)
// fmt.Println("clusterresult", clusterresult)
// }
