package kMeansClustering

import (
	"log"
	"math"
	"os"
	. "ridehost/types"

	"github.com/muesli/clusters"
	"github.com/muesli/kmeans"
	// "github.com/biogo/cluster/kmeans"
)

var logger = log.New(os.Stdout, "ClusteringNode-kmeansclustering ", log.Ldate|log.Ltime)

func CentralizedKMeansClustering(Nodelist []Node, k int) ClusterResult {
	data := []Point{}
	for i := 0; i < len(Nodelist); i++ {
		lat := Nodelist[i].StartLat
		lng := Nodelist[i].StartLng
		pair := Point{X: lat, Y: lng}
		data = append(data, pair)
	}

	// k := 3 // number of clusters
	// data := []Point{{1., 1.}, {1., 2.}, {2., 2.}, {5., 5.}, {6., 5.}, {7., 7.}, {10., 10.}, {12., 10.}, {12., 12.}}

	var d clusters.Observations
	for i := 0; i < len(data); i++ {
		d = append(d, clusters.Coordinates{
			data[i].X,
			data[i].Y,
		})
	}
	km := kmeans.New()
	clusters, err := km.Partition(d, k)
	centroids := []Point{}
	pointclusters := [][]Point{}
	clusterMaps := map[Node][]Node{}
	centerPointList := []Point{}
	centerPointNodeList := []Node{}
	logger.Println("err if any: ", err)
	for _, c := range clusters {
		temp := []Point{}
		clusterList := []Node{}
		centerPoint := Point{X: c.Center[0], Y: c.Center[1]}
		nearestPointToCenterIdx := 0
		nearestPointToCenter := Point{X: 0, Y: 0}
		pointnode := Node{}
		idxOfPoint := 0
		min_dis := 999999999.0
		indexes_visited := make(map[int]int, len(data))
		for i := 0; i < len(data); i++ {
			indexes_visited[i] = 0
		}
		for _, obs := range c.Observations {
			p := Point{X: obs.Coordinates()[0], Y: obs.Coordinates()[1]}
			// logger.Println("data : ", data)
			// logger.Println("p : ", p)
			// indexes := make([]int, 0)
			for i, v := range data {
				if v == p {
					// indexes = append(indexes, i)
					if indexes_visited[i] == 0 {
						idxOfPoint = i
						indexes_visited[i] = 1
						break
					}

				}
			}

			// idxOfPoint = slices.Index(data, p)
			cur_dis := euclideanDistance(p, centerPoint)
			if cur_dis < min_dis {
				min_dis = cur_dis
				nearestPointToCenterIdx = idxOfPoint
				nearestPointToCenter = p
			}
			pointnode = Nodelist[idxOfPoint]
			clusterList = append(clusterList, pointnode)
			temp = append(temp, p)

		}
		nearestPointNodeToCenter := Nodelist[nearestPointToCenterIdx]
		centerPointList = append(centerPointList, nearestPointToCenter)
		centerPointNodeList = append(centerPointNodeList, nearestPointNodeToCenter)
		clusterMaps[nearestPointNodeToCenter] = clusterList
		pointclusters = append(pointclusters, temp)
		centroids = append(centroids, Point{X: c.Center[0], Y: c.Center[1]})
	}
	// fmt.Println("centroids", centroids)
	// fmt.Println("pointclusters", pointclusters)
	// logger.Println("clusterMaps : ", clusterMaps)
	clusterresult := ClusterResult{}
	clusterresult.Centroids = centroids
	clusterresult.Clusters = pointclusters
	clusterresult.ClusterMaps = clusterMaps
	clusterresult.RepresentCenterPoints = centerPointList
	clusterresult.RepresentCenterNodes = centerPointNodeList

	return clusterresult
}

// func IndividualKMeansClustering(Nodelist []Node, k int) Coreset {

// 	clusterresults := CentralizedKMeansClustering(Nodelist, k)
// 	// return clusterresults
// 	// fmt.Println("clusterresults from centraulized k means: ", clusterresults)
// 	fmt.Println("centroids from centraulized k means: ", clusterresults.Centroids)

// 	centroids := clusterresults.Centroids
// 	data := []Point{}
// 	for i := 0; i < len(Nodelist); i++ {
// 		lat := Nodelist[i].Lat
// 		lng := Nodelist[i].Lng
// 		pair := Point{X: lat, Y: lng}
// 		data = append(data, pair)
// 	}
// 	//step 2: calculate approximate costs C(Pi, Bi)
// 	costList := []float64{}
// 	cost := 0.
// 	for _, point := range data {
// 		min_dis := 87249394.
// 		for _, center := range centroids {
// 			cur_dis := euclideanDistance(point, center)
// 			if cur_dis < min_dis {
// 				min_dis = cur_dis
// 			}
// 		}
// 		costList = append(costList, min_dis)
// 		cost += min_dis * min_dis
// 	}
// 	fmt.Println("cost : ", cost)
// 	fmt.Println("costList : ", costList)

// 	// step 3: TODO Communicate cost to all other clustering nodes via RPC
// 	// also communicate length of membership lists to calculate t

// 	//k-means-clustering calls this in step 3.
// 	// get this machine's IP address
// 	add, err := net.LookupIP("ispycode.com")
// 	for _, node := range clusteringNodes {
// 		if node.Ip != add {
// 			sendCostMsg(node, Cost, lenML)
// 		}
// 	}
// 	// wait for all responses to be recvd
// 	clusteringNodesResponseList.mu.Lock()
// 	for len(clusteringNodesResponseList.List) < len(clusteringNodes) {
// 		clusteringNodesResponseList.cond.Wait()
// 	}

// 	t := 9
// 	// as a result from step 3

// 	costFromAllClusteringNodes := []float64{}

// 	sumOfCostOfAllClusteringNodes := 0.

// 	for _, val := range costFromAllClusteringNodes {
// 		sumOfCostOfAllClusteringNodes += val
// 	}

// 	// Round 2 step 1 : Compute ti
// 	ti := int(float64(t) * cost / sumOfCostOfAllClusteringNodes)
// 	ti = 4
// 	// Round 2 step 2: multiply each cost by 2 to get mp
// 	mp := []float64{}

// 	for i := 0; i < len(costList); i++ {
// 		mp = append(mp, 2*costList[i])
// 	}

// 	mpProb := []float64{}

// 	for i := 0; i < len(costList); i++ {
// 		mpProb = append(mpProb, costList[i]/cost)
// 	}

// 	// Round 2 step 3: Non-uniform random sample 𝑆𝑖 of 𝑡𝑖 points from Pi, where for every 𝑞 ∈ 𝑃𝑖.
// 	// Also find weights on q, wq

// 	fmt.Println("mpProb: ", mpProb)
// 	fmt.Println("data: ", data)
// 	w := sampleuv.NewWeighted(mpProb, nil)
// 	q_indexes := []int{}
// 	qp := []Point{}

// 	fmt.Println("ti: ", ti)
// 	for i := 0; i < ti; i++ {
// 		index, _ := w.Take()
// 		fmt.Println("index: ", index)
// 		q_indexes = append(q_indexes, index)
// 		qp = append(qp, data[index])
// 	}

// 	wq := []float64{}
// 	num := 0.
// 	// add the mps of this clustering node to the num
// 	for i := 0; i < len(mp); i++ {
// 		num += mp[i]
// 	}
// 	fmt.Println("idhar aaya 1")
// 	// add the mps of all other clustering nodes to the num.
// 	// since mp=2*cost, the summation mp = 2* sum of all cost from all clustering nodes
// 	num += sumOfCostOfAllClusteringNodes * 2

// 	for _, index := range q_indexes {

// 		denom := mp[index] * float64(t)
// 		wq = append(wq, num/denom)
// 	}
// 	fmt.Println("idhar aaya 2")
// 	// Round 2 step 4: Local k-centers (𝐵𝑖 ) are included to the coreset
// 	// Also find Weight on b, wb
// 	core := Coreset{}
// 	coreset := qp
// 	wb := []float64{}
// 	for _, centroid := range centroids {
// 		coreset = append(coreset, centroid)
// 		pb := []Point{}
// 		pb_indexes := []int{}
// 		for idx, point := range data {
// 			euc_dist := euclideanDistance(centroid, point)
// 			if euc_dist == costList[idx] {
// 				pb = append(pb, point)
// 				pb_indexes = append(pb_indexes, idx)
// 			}
// 		}

// 		// Finding Pb 𝑃 = {𝑝∈𝑃: 𝑑(𝑝,𝑏) = 𝑑(𝑝,𝐵)}
// 		intersectionOfPbAndS := intersect(pb_indexes, q_indexes)
// 		fmt.Println("(pb_indexes, q_indexes, wq)", pb_indexes, q_indexes, wq)
// 		fmt.Println("idhar aaya 3", intersectionOfPbAndS)

// 		sumOfWqs := 0.
// 		for _, in := range intersectionOfPbAndS {
// 			sumOfWqs += wq[in]
// 		}
// 		wb = append(wb, float64(len(pb))-sumOfWqs)
// 		fmt.Println("idhar aaya 4")
// 	}
// 	core.Coreset = coreset
// 	return core
// }

func intersect(slice1, slice2 []int) []int {
	var intersect []int
	for _, element1 := range slice1 {
		for _, element2 := range slice2 {
			if element1 == element2 {
				intersect = append(intersect, element1)
			}
		}
	}
	return intersect //return slice after intersection
}

func euclideanDistance(p1, p2 Point) float64 {
	return math.Sqrt(math.Pow(p1.X-p2.X, 2) + math.Pow(p1.Y-p2.Y, 2))
}

// func main() {
// 	n1 := Node{NodeType: Driver, Ip: "35", Uuid: uuid.New(), Lat: 24, Lng: 25}
// 	n2 := Node{NodeType: Driver, Ip: "890", Uuid: uuid.New(), Lat: 1, Lng: 2}
// 	n3 := Node{NodeType: Driver, Ip: "234", Uuid: uuid.New(), Lat: 12, Lng: 13}
// 	n4 := Node{NodeType: Driver, Ip: "trfh", Uuid: uuid.New(), Lat: 5, Lng: 6}
// 	n5 := Node{NodeType: Driver, Ip: "5676", Uuid: uuid.New(), Lat: 6, Lng: 5}
// 	n6 := Node{NodeType: Driver, Ip: "23478", Uuid: uuid.New(), Lat: 1, Lng: 1}
// 	n7 := Node{NodeType: Driver, Ip: "1279", Uuid: uuid.New(), Lat: 15, Lng: 15}
// 	n8 := Node{NodeType: Driver, Ip: "11234", Uuid: uuid.New(), Lat: 16, Lng: 19}
// 	n9 := Node{NodeType: Driver, Ip: "qwerty", Uuid: uuid.New(), Lat: 4, Lng: 8}
// 	nlist := []Node{}
// 	nlist = append(nlist, n1)
// 	nlist = append(nlist, n2)
// 	nlist = append(nlist, n3)
// 	nlist = append(nlist, n4)
// 	nlist = append(nlist, n5)
// 	nlist = append(nlist, n6)
// 	nlist = append(nlist, n7)
// 	nlist = append(nlist, n8)
// 	nlist = append(nlist, n9)

// 	// clusterresult := CentralizedKMeansClustering(nlist, 3)
// 	// fmt.Println("clusterresult", clusterresult)
// 	clusterresult := IndividualKMeansClustering(nlist, 3)
// 	fmt.Println("clusterresult", clusterresult)
// }
