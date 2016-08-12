deploy: build test

build: $(DOCKER_CMD)
	docker build -t carlos-the-curious .

local-test: 
	godep go test ./slackbot

test:
	docker run --rm  --net=host -w /go/src/github.com/dklassen/CarlosTheCurious  carlos-the-curious go test ./slackbot

clean:
	rm -rf $(DOCKER_BUILD)

run:
	docker run --net=host --rm -it -e "DATABASE_URL=$(DATABASE_URL)" -e "SLACKTOKEN=$(SLACKTOKEN)" carlos-the-curious

console:
	docker run -it --entrypoint /bin/bash carlos-the-curious

database-up:
	docker run -d --name carlos-postgres  --publish 5432:5432 postgres

database-down: database-stop database-clean

database-stop:
	docker stop carlos-postgres


database-build:
	docker build --tag carlos/postgres --file Dockerfile.devdb .

database-clean:
	docker rm /carlos-postgres
