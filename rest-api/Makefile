# Makefile for the Docker image stevesloka/rook-api
# MAINTAINER: Steve Sloka <steves@heptio.com>

.PHONY: all build container push clean test

TAG ?= latest
PREFIX ?= stevesloka

all: container

build:
	GOOS=linux GOARCH=amd64 go build -o _output/bin/rook-api ./main.go

container: build
	docker build -t $(PREFIX)/rook-api:$(TAG) .

push:
	docker push $(PREFIX)/rook-api:$(TAG)

clean:
	rm -f rook-api