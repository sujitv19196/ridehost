package types

import "net"

const (
	Driver  int = 0
	Rider   int = 1
	Cluster int = 2
)

type Node struct {
	NodeType int
	Ip       *net.TCPAddr
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

type Coreset struct {
	coreset map[string]int //TODO change, placeholder
}

type ClusterInfo struct {
	ClusterRep string
	ClusterNum int
}

type ClientMainClustererResponse struct {
	Message string
	Error   error
}

type ClientClusterJoinRequest struct {
	ClusterNum   int
	ClusterRepIP string
	Members      []string
}

type ClientClusterJoinResponse struct {
	Ack bool
}

type ClusterNodeRemovalRequest struct {
	NodeIP string
}

type ClusterNodeRemovalResponse struct {
	Ack bool
}
