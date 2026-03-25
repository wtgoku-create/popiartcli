BINARY := dist/popiart
GOFILES := $(shell find cmd internal -name '*.go' | sort)
ARGS ?= --help

.PHONY: tidy fmt build run help

tidy:
	go mod tidy

fmt:
	gofmt -w $(GOFILES)

build:
	mkdir -p dist
	go build -o $(BINARY) ./cmd/popiart

run:
	go run ./cmd/popiart $(ARGS)

help:
	go run ./cmd/popiart --help
