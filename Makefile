all:  clean build

#pull: 
        #git pull
        #git checkout feature/improve-logging

build:
	go build -o ./introducer introducer/introducer.go
	go build -o ./client client/client.go
	go build -o ./clusteringNode clusteringNode/clusteringNode.go
	go build -o ./mainClusterer mainClusterer/mainClusterer.go

clean: 
	rm client/client
	rm introducer/introducer
	rm clusteringNode/clusteringNode
	rm mainClusterer/mainClusterer