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

#git remote remove origin
#git remote add origin git@github.com:sujitv19196/ridehost.git
#git remote set-url --add --push origin git@github.com:sujitv19196/ridehost.git
#git remote set-url --add --push origin pjain15@sp23-cs525-0301.cs.illinois.edu:/home/pjain15/ridehost
#git push --set-upstream origin feature/improve-logging
#git push
