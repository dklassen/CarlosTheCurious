LOCAL_BUILD=$(shell pwd)/.docker_build
LOCAL_CMD=$(LOCAL_BUILD)/carlos-the-curious

container-build: 
	docker build -t carlos-the-curious .

container-test: container-build
	docker run --rm  --net=host -w /go/src/github.com/dklassen/CarlosTheCurious -e DEBUG=$(DEBUG) carlos-the-curious go test ./slackbot

container-run:
	docker run --net=host --rm -it carlos-the-curious

local-build: local-clean 
	go get -u github.com/kardianos/govendor
	govendor sync
	mkdir -p $(LOCAL_BUILD)
	go build -v -o $(LOCAL_CMD) . 

local-run:
	$(LOCAL_CMD)

local-test: 
	go test ./slackbot

local-clean:
	rm -rf $(DOCKER_BUILD)

console:
	docker run -it --entrypoint /bin/bash carlos-the-curious

database-up:
	docker run -d --name carlos-postgres  --publish 5432:5432 carlos/postgres

database-down: database-stop database-clean

database-stop:
	docker stop carlos-postgres

database-build:
	docker build --tag carlos/postgres --file Dockerfile.devdb .

database-clean:
	docker rm /carlos-postgres
