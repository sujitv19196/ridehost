build:
	go build -o ./introducer introducer/introducer.go
	go build -o ./client client/client.go
	go build -o ./clusteringNode clusteringNode/clusteringNode.go

clean: 
	rm bin/client
	rm bin/introducer
	rm bin/clusteringNode
