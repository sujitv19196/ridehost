package types

const (
	Driver  int = 0
	Rider   int = 1
	Cluster int = 2
)

type Node struct {
	NodeType int
	Ip       string
	Uuid     [16]byte
	Lat      float64
	Lng      float64
}

type JoinRequest struct {
	NodeRequest  Node
	IntroducerIp string
}

type ClientIntroducerResponse struct {
	Message string
	Error   error
}

type IntroducerMainClustererResponse struct {
	Message string
	Error   error
}

type MainClustererClusteringNodeResponse struct {
	Message string
	Error   error
}

type Point struct {
	X float64
	Y float64
}

type Coreset struct {
	Coreset      []Point //TODO change, placeholder
	CoresetNodes []Node
	Tempcluster  map[Node][]Node
}

type ClusterResult struct {
	Centroids             []Point
	Clusters              [][]Point
	ClusterMaps           map[Node][]Node
	RepresentCenterPoints []Point
	RepresentCenterNodes  []Node
}

type ClusterInfo struct {
	NodeItself Node
	ClusterRep Node
	ClusterNum int
}

type ClientMainClustererResponse struct {
	Message string
	Error   error
}
