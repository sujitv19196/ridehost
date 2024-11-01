package types

import (
	"github.com/google/uuid"
)

const (
	Driver        int = 0
	Rider         int = 1
	Stationary    int = 2
	Introducer    int = 3
	MainClusterer int = 4
)

type Node struct {
	NodeType  int
	Ip        string
	Uuid      uuid.UUID
	StartLat  float64
	StartLng  float64
	DestLat   float64
	DestLng   float64
	Cost      float64
	PingReady bool
}

type JoinRequest struct {
	NodeRequest  Node
	IntroducerIp string
}

type ClientIntroducerResponse struct {
	Message          string
	Error            error
	IsClusteringNode bool
}

type AckErrResponse struct {
	Message string
	Error   error
}

type ClientReadyRequest struct {
	RequestingNode Node
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
	Weights      []float64
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

type CostMsg struct {
	NodeIp   string
	Cost     float64
	LengthML int
}

type ClusteringNodeClusteringNodeResponse struct {
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
	NodeUuid string
}

type IntroducerNodeAddRequest struct {
	NodeToAdd Node
}

type IntroducerNodeAddResponse struct {
	Members []Node
}

type ClusterNodeAddRequest struct {
	NodeToAdd Node
}

type ClusterNodeAddResponse struct {
	Ack bool
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

type NodeFailureDetectingPingingStatusReq struct {
	Uuid   string
	Ip     string
	Status bool
}

type NodeFailureDetectingPingingStatusRes struct {
	Message string
	Error   error
}
