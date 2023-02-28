.DEFAULT_GOAL := build-all

LDFLAGS_COMMON=-extldflags '-static' -X 'github.com/weeveiot/weeve-agent/internal/model.Version=$(shell git tag | sort -V | tail -n 1 | cut -c2-)'
LDFLAGS_RELEASE=$(LDFLAGS_COMMON) -w

build:
	# add -ldflags="-w" to remove debug info

	CGO_ENABLED=0 go build -a -tags netgo -ldflags="$(LDFLAGS_COMMON)" -o bin/weeve-agent ./cmd/agent/agent.go
.PHONY: build

build-x86:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags netgo -ldflags="$(LDFLAGS_COMMON)" -o bin/weeve-agent-linux-amd64 ./cmd/agent/agent.go
.PHONY: build-x86

build-arm:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -a -tags netgo -ldflags="$(LDFLAGS_COMMON)" -o bin/weeve-agent-linux-arm-v7 ./cmd/agent/agent.go
.PHONY: build-arm

build-darwin:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -a -tags netgo -ldflags="$(LDFLAGS_COMMON)" -o bin/weeve-agent-darwin ./cmd/agent/agent.go
.PHONY: build-darwin

cross:
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -a -tags netgo -ldflags="$(LDFLAGS_RELEASE)" -o bin/weeve-agent-linux-amd64    ./cmd/agent/agent.go
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build -a -tags netgo -ldflags="$(LDFLAGS_RELEASE)" -o bin/weeve-agent-linux-arm64    ./cmd/agent/agent.go
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm   go build -a -tags netgo -ldflags="$(LDFLAGS_RELEASE)" -o bin/weeve-agent-linux-arm      ./cmd/agent/agent.go
.PHONY: cross

build-all: build-arm build-x86 build-darwin
.PHONY: build-all
