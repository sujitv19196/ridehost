package types

type NodeType int

const (
	Driver     NodeType = 0
	Rider      NodeType = 1
	Stationary NodeType = 2
)

type AcceptClientRequest struct {
	RequestType  NodeType
	IntroducerIP string
	Uuid         [16]byte
	Lat          float64
	Lng          float64
}

type AcceptClientResponse struct {
	ClusterNum int
	Error      error
}

type AcceptClientFromIntroducerRequest struct {
	RequestType  NodeType
	IntroducerIP string
	DataList     []string
	Uuid         [16]byte
	Lat          float64
	Lng          float64
}

type AcceptClientFromIntroducerResponse struct {
	ClusterNum int
	Error      error
	Message    string
}
