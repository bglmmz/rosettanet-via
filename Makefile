# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
BINARY_NAME=via-gm

build:
	$(GOBUILD) -o bin/$(BINARY_NAME) cmd/via_server.go -v
