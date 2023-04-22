package types

import "github.com/google/uuid"

const (
	Driver  int = 0
	Rider   int = 1
	Cluster int = 2
)

type Node struct {
	NodeType int
	Ip       string
	Uuid     uuid.UUID
	StartLat float64
	StartLng float64
	DestLat  float64
	DestLng  float64
	Cost     float64
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

type ClientMainClustererResponse struct {
	Message string
	Error   error
}

type ClientClusterJoinRequest struct {
	ClusterNum int
	ClusterRep Node
	NodeItself Node
	Members    []Node
}

type ClientClusterJoinResponse struct {
	Ack bool
}

type ClientClusterPingingStatusRequest struct {
	Status bool
}

type ClientClusterPingingStatusResponse struct {
	Ack bool
}

type ClusterNodeRemovalRequest struct {
	NodeIP string
}

type ClusterNodeRemovalResponse struct {
	Ack bool
}

type Bid struct {
	Cost float64
}

type BidResponse struct {
	Response bool
	Accept   bool
}

type DriverInfo struct {
	PhoneNumber string
}

type RiderInfo struct {
	Response    bool
	PhoneNumber string
}
