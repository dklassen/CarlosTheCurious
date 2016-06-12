FROM golang:1.7-alpine

MAINTAINER Dana Klassen <dana.klassen@shopify.com> 

ENV PROJECT /go/src/github.com/dklassen/CarlosTheCurious
WORKDIR $PROJECT

RUN apk add --update ca-certificates

RUN update-ca-certificates

RUN apk update && \
    apk upgrade && \
    apk add bash git openssh

RUN mkdir -p $PROJECT

COPY . $PROJECT

RUN go get -u github.com/kardianos/govendor
RUN govendor sync
RUN go install -v 

EXPOSE 5432

CMD $PROJECT/script/docker-entrypoint.sh
