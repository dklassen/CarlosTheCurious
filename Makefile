GO_BUILD_ENV := GOOS=linux GOARCH=amd64
DOCKER_BUILD=$(shell pwd)/.docker_build
DOCKER_CMD=$(DOCKER_BUILD)/carlos-the-curious


$(DOCKER_CMD): clean
	mkdir -p $(DOCKER_BUILD)
	$(GO_BUILD_ENV) go build -v -o $(DOCKER_CMD) .

build: $(DOCKER_CMD)
	docker build -t carlos-the-curious .

clean:
	rm -rf $(DOCKER_BUILD)

run:
	docker run --net=host --rm -it -e "DATABASE_URL=$(DATABASE_URL)" -e "SLACKTOKEN=$(SLACKTOKEN)" carlos-the-curious

database-up:
	docker run -d --name carlos-postgres --net=host -p 5432:5432 postgres

database-down:
	docker stop carlos-postgres

database-build:
	docker build --tag carlos/postgres --file Dockerfile.devdb .

database-clean:
	docker rm /carlos-postgres
