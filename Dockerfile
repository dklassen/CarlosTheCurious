from alpine:latest

MAINTAINER Dana Klassen <dana.klassen@shopify.com> 

WORKDIR /opt

RUN apk add --update ca-certificates && \
    rm -rf /var/cache/apk/* /tmp/*

RUN update-ca-certificates

COPY .docker_build/carlos-the-curious bin/carlos-the-curious

COPY ./script/docker-entrypoint.sh /

EXPOSE 5432
ENTRYPOINT ["/docker-entrypoint.sh"]

