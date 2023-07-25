build:
	go build -o ./introducer cmd/introducer/introducer.go
	go build -o ./client cmd/client/client.go
	go build -o ./mainClusterer cmd/mainClusterer/mainClusterer.go

clean: 
	rm cmd/client/client
	rm cmd/introducer/introducer
	rm cmd/mainClusterer/mainClusterer