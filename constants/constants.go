package constants

import (
	"time"
)

var Ports = map[string]int{"introducer": 2233, "mainClusterer": 2234, "clusteringNode": 2235, "acceptPings": 2236, "client": 2237, "clientRPC": 2238}

const PingFrequency = 3 * time.Second

const UDPPingAckTimeout = 2000 * time.Millisecond

const NumClusters = 5

const ClusteringPeriod = 1 // in minutes

const TCPTimeout = time.Minute / 12

const BidTimeout = time.Second * 5 // 5 seconds

const RiderMaxCost float64 = 100 //km
