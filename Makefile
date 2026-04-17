BINARY_NAME=neeto-cli-gen
VERSION=$(shell cat VERSION 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-s -w -X github.com/neetozone/neeto-cli-template/internal/version.Version=$(VERSION) -X github.com/neetozone/neeto-cli-template/internal/version.Commit=$(COMMIT) -X github.com/neetozone/neeto-cli-template/internal/version.Date=$(DATE)"

.PHONY: build test lint install clean fmt vet setup check

build:
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/neeto-cli-gen/

test:
	go test ./...

lint:
	golangci-lint run

fmt:
	gofmt -w .

vet:
	go vet ./...

check: fmt vet test

install: build
	cp $(BINARY_NAME) /usr/local/bin/

setup:
	git config core.hooksPath .githooks

clean:
	rm -f $(BINARY_NAME)
