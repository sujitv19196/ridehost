package types

const (
	Driver     int = 0
	Rider      int = 1
	Stationary int = 2
)

type ClientIntroducerRequest struct {
	RequestType  int
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
