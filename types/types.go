package types

type NodeType int

const (
	Driver     NodeType = 0
	Rider      NodeType = 1
	Stationary NodeType = 2
)

type ClientIntroducerRequest struct {
	RequestType  NodeType
	IntroducerIP string
	Uuid         [16]byte
	Lat          float64
	Lng          float64
}

type ClientIntroducerResponse struct {
	ClusterNum int
	Error      error
}

type IntroducerClusterRequest struct {
	Uuid [16]byte
	Lat  float64
	Lng  float64
}

type IntroducerClusterResponse struct {
	ClusterNum int
	Result     string //json
	Error      error
	Message    string
}
