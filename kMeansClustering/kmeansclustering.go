package main

import (
	"fmt"
	"math"
	. "ridehost/types"

	"github.com/google/uuid"
	"github.com/muesli/clusters"
	"github.com/muesli/kmeans"
	"golang.org/x/exp/slices"

	// "github.com/biogo/cluster/kmeans"
	"gonum.org/v1/gonum/stat/sampleuv"
)

type Point struct {
	x float64
	y float64
}
type ClusterResult struct {
	centroids          []Point
	clusters           [][]Point
	clusterMaps        map[string][]string
	representCenters   []Point
	representCenterIds []string
}

func CentralizedKMeansClustering(Nodelist []Node, k int) ClusterResult {
	data := []Point{}
	for i := 0; i < len(Nodelist); i++ {
		lat := Nodelist[i].Lat
		lng := Nodelist[i].Lng
		pair := Point{lat, lng}
		data = append(data, pair)
	}

	// k := 3 // number of clusters
	// data := []Point{{1., 1.}, {1., 2.}, {2., 2.}, {5., 5.}, {6., 5.}, {7., 7.}, {10., 10.}, {12., 10.}, {12., 12.}}

	var d clusters.Observations
	for x := 0; x < len(data); x++ {
		d = append(d, clusters.Coordinates{
			data[x].x,
			data[x].y,
		})
	}
	km := kmeans.New()
	clusters, err := km.Partition(d, k)
	centroids := []Point{}
	pointclusters := [][]Point{}
	clusterMaps := map[string][]string{}
	centerPointList := []Point{}
	centerPointIdList := []string{}
	fmt.Println("err if any: ", err)
	for _, c := range clusters {
		temp := []Point{}
		clusterUuidList := []string{}
		centerPoint := Point{c.Center[0], c.Center[1]}
		nearestPointToCenterIdx := 0
		nearestPointToCenter := Point{0, 0}
		uuidOfPoint := ""
		idxOfPoint := 0
		min_dis := 96789056.0
		for _, obs := range c.Observations {
			p := Point{obs.Coordinates()[0], obs.Coordinates()[1]}
			idxOfPoint = slices.Index(data, p)
			cur_dis := euclideanDistance(p, centerPoint)
			if cur_dis < min_dis {
				min_dis = cur_dis
				nearestPointToCenterIdx = idxOfPoint
				nearestPointToCenter = p
			}
			uuidOfPoint = string(Nodelist[idxOfPoint].Uuid[:])
			clusterUuidList = append(clusterUuidList, uuidOfPoint)
			// fmt.Printf("the index of %v in %v is %v", p, data, idx)
			temp = append(temp, p)

		}
		nearestPointToCenterUuid := string(Nodelist[nearestPointToCenterIdx].Uuid[:])
		centerPointList = append(centerPointList, nearestPointToCenter)
		centerPointIdList = append(centerPointIdList, nearestPointToCenterUuid)
		// fmt.Println("clusterUuidList idxOfPoint", clusterUuidList)
		// fmt.Println("nearestPointToCenterUuid ", nearestPointToCenterUuid)
		clusterMaps[nearestPointToCenterUuid] = clusterUuidList
		pointclusters = append(pointclusters, temp)
		centroids = append(centroids, Point{c.Center[0], c.Center[1]})
	}
	// fmt.Println("centroids", centroids)
	// fmt.Println("pointclusters", pointclusters)
	clusterresult := ClusterResult{}
	clusterresult.centroids = centroids
	clusterresult.clusters = pointclusters
	clusterresult.clusterMaps = clusterMaps
	clusterresult.representCenters = centerPointList
	clusterresult.representCenterIds = centerPointIdList

	return clusterresult
}

func IndividualKMeansClustering(Nodelist []Node, k int, t int) {

	clusterresults := CentralizedKMeansClustering(Nodelist, k)
	// return clusterresults
	// fmt.Println("clusterresults from centraulized k means: ", clusterresults)
	fmt.Println("centroids from centraulized k means: ", clusterresults.centroids)

	centroids := clusterresults.centroids
	data := []Point{}
	for i := 0; i < len(Nodelist); i++ {
		lat := Nodelist[i].Lat
		lng := Nodelist[i].Lng
		pair := Point{lat, lng}
		data = append(data, pair)
	}
	//step 2: calculate approximate costs C(Pi, Bi)
	costList := []float64{}
	cost := 0.
	for _, point := range data {
		min_dis := 87249394.
		for _, center := range centroids {
			cur_dis := euclideanDistance(point, center)
			if cur_dis < min_dis {
				min_dis = cur_dis
			}
		}
		costList = append(costList, min_dis)
		cost += min_dis * min_dis
	}

	// step 3: TODO Communicate cost to all other clustering nodes via RPC

	// as a result from step 3

	costFromAllClusteringNodes := []float64{}

	sumOfCostOfAllClusteringNodes := 0.

	for _, val := range costFromAllClusteringNodes {
		sumOfCostOfAllClusteringNodes += val
	}

	// Round 2 step 1 : Compute ti
	ti := int(float64(t) * cost / sumOfCostOfAllClusteringNodes)
	ti = 6
	// Round 2 step 2: multiply each cost by 2 to get mp
	mp := []float64{}

	for i := 0; i < len(costList); i++ {
		mp = append(mp, 2*costList[i])
	}

	mpProb := []float64{}

	for i := 0; i < len(costList); i++ {
		mpProb = append(mpProb, costList[i]/cost)
	}

	// Round 2 step 3: Non-uniform random sample 𝑆𝑖 of 𝑡𝑖 points from Pi, where for every 𝑞 ∈ 𝑃𝑖.
	// Also find weights on q, wq

	fmt.Println("mpProb: ", mpProb)
	fmt.Println("data: ", data)
	w := sampleuv.NewWeighted(mpProb, nil)
	q_indexes := []int{}
	qp := []Point{}

	fmt.Println("ti: ", ti)
	for i := 0; i < ti; i++ {
		index, _ := w.Take()
		fmt.Println("index: ", index)
		q_indexes = append(q_indexes, index)
		qp = append(qp, data[index])
	}

	wq := []float64{}
	num := 0.
	// add the mps of this clustering node to the num
	for i := 0; i < len(mp); i++ {
		num += mp[i]
	}

	// add the mps of all other clustering nodes to the num.
	// since mp=2*cost, the summation mp = 2* sum of all cost from all clustering nodes
	num += sumOfCostOfAllClusteringNodes * 2

	for _, index := range q_indexes {

		denom := mp[index] * float64(t)
		wq = append(wq, num/denom)
	}

	// Round 2 step 4: Local k-centers (𝐵𝑖 ) are included to the coreset
	// Also find Weight on b, wb
	coresets := qp
	wb := []float64{}
	for _, centroid := range centroids {
		coresets = append(coresets, centroid)
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
		sumOfWqs := 0.
		for _, in := range intersectionOfPbAndS {
			sumOfWqs += wq[in]
		}
		wb = append(wb, float64(len(pb))-sumOfWqs)

	}

}
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
	return math.Sqrt(math.Pow(p1.x-p2.x, 2) + math.Pow(p1.y-p2.y, 2))
}

func main() {
	n1 := Node{NodeType: Driver, Ip: nil, Uuid: uuid.New(), Lat: 24, Lng: 25}
	n2 := Node{NodeType: Driver, Ip: nil, Uuid: uuid.New(), Lat: 1, Lng: 2}
	n3 := Node{NodeType: Driver, Ip: nil, Uuid: uuid.New(), Lat: 12, Lng: 13}
	n4 := Node{NodeType: Driver, Ip: nil, Uuid: uuid.New(), Lat: 5, Lng: 6}
	n5 := Node{NodeType: Driver, Ip: nil, Uuid: uuid.New(), Lat: 6, Lng: 5}
	n6 := Node{NodeType: Driver, Ip: nil, Uuid: uuid.New(), Lat: 1, Lng: 1}
	n7 := Node{NodeType: Driver, Ip: nil, Uuid: uuid.New(), Lat: 15, Lng: 15}
	n8 := Node{NodeType: Driver, Ip: nil, Uuid: uuid.New(), Lat: 16, Lng: 19}
	n9 := Node{NodeType: Driver, Ip: nil, Uuid: uuid.New(), Lat: 4, Lng: 8}
	nlist := []Node{}
	nlist = append(nlist, n1)
	nlist = append(nlist, n2)
	nlist = append(nlist, n3)
	nlist = append(nlist, n4)
	nlist = append(nlist, n5)
	nlist = append(nlist, n6)
	nlist = append(nlist, n7)
	nlist = append(nlist, n8)
	nlist = append(nlist, n9)

	clusterresult := CentralizedKMeansClustering(nlist, 3)
	fmt.Println("clusterresult", clusterresult)
	// IndividualKMeansClustering(nlist, 3, 1)
	// fmt.Println("clusterresult", clusterresult)
}
