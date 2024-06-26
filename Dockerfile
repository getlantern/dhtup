# syntax=docker/dockerfile:1.3

# i sleep
FROM alpine:latest as builder

RUN apk add go
RUN apk add gcompat

WORKDIR /app
RUN apk add make gcc
RUN apk add musl-dev
RUN apk add g++
COPY Makefile .
ENV GOWORK=off
ENV GOMODCACHE=/gomodcache
ENV GOCACHE=/gocache
RUN \
	--mount=type=cache,target=$GOMODCACHE \
	--mount=type=cache,target=$GOCACHE \
	make deps


# real shit
FROM alpine

RUN apk add curl aws-cli
RUN apk add make
RUN apk add gcompat
RUN apk add bash

WORKDIR /app
# We need GNU timeout
RUN apk add coreutils
COPY --from=builder /app/bin/ bin/
COPY Makefile .
# TODO take from secrets/env
COPY dht-private-key .
COPY run ./
# This lets us override stuff with local builds. Git doesn't let us retain empty directories, and I don't like the idea of using a turdfile that gets dumped in the image root to keep it alive. Uncomment this when you want to override some contents of the image.
#COPY docker-build/ /

ENTRYPOINT ["make"]
CMD ["publish", "seed"]
