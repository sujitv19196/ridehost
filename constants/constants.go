package constants

import (
	"time"
)

var Ports = map[string]int{"introducer": 2233, "mainClusterer": 2234, "clusteringNode": 2235, "acceptPings": 2236, "clientRPC": 2237}

const PingFrequency = 2 * time.Second

const UDPPingAckTimeout = 1500 * time.Millisecond

const NumClusters = 5

const ClusteringPeriod = 1 // in minutes

const TCPTimeout = time.Minute / 12
