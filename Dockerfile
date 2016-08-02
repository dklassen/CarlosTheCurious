FROM golang:1.6-alpine

MAINTAINER Dana Klassen <dana.klassen@shopify.com> 

ENV PROJECT /go/src/github.com/dklassen/CarlosTheCurious
WORKDIR $PROJECT

RUN mkdir -p $PROJECT
COPY . $PROJECT

RUN apk add --update ca-certificates
RUN rm -rf /var/cache/apk/* /tmp/*

RUN update-ca-certificates

RUN apk update && \
    apk upgrade && \
    apk add bash \ 
            git \
            openssh


RUN go get github.com/tools/godep
RUN godep go install -v 

EXPOSE 5432

ENTRYPOINT $PROJECT/script/docker-entrypoint.sh
