MODULE := github.com/neuland-ingolstadt/discord-bot
BIN := bin/bot
IMG ?= ghcr.io/neuland-ingolstadt/discord-bot
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.PHONY: all build test tidy fmt vet docker-build

all: tidy fmt vet test build

build:
	mkdir -p bin
	CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=$(VERSION) -X main.commit=$$(git rev-parse --short HEAD 2>/dev/null || echo unknown)" -o $(BIN) ./cmd/bot

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

tidy:
	go mod tidy

docker-build:
	docker build --build-arg VERSION=$(VERSION) --build-arg COMMIT=$$(git rev-parse HEAD 2>/dev/null || echo unknown) -t $(IMG):$(VERSION) .
