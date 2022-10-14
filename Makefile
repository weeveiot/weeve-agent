.DEFAULT_GOAL := build-all

build-x86:
	GOOS=linux GOARCH=amd64 go build -o bin/weeve-agent-linux-amd64 ./cmd/agent/agent.go
.PHONY: build-x86

build-arm:
	GOOS=linux GOARCH=arm GOARM=7 go build -o bin/weeve-agent-linux-arm-v7 ./cmd/agent/agent.go
.PHONY: build-arm

build-darwin:
	GOOS=darwin GOARCH=amd64 go build -o bin/weeve-agent-darwin ./cmd/agent/agent.go
.PHONY: build-darwin

cross:
	rm -rf installer-contents
	GOOS=linux   GOARCH=amd64 go build -o installer-contents/weeve-agent-linux-amd64    ./cmd/agent/agent.go
	GOOS=linux   GOARCH=arm64 go build -o installer-contents/weeve-agent-linux-arm64    ./cmd/agent/agent.go
	GOOS=linux   GOARCH=arm   go build -o installer-contents/weeve-agent-linux-arm      ./cmd/agent/agent.go
	GOOS=darwin  GOARCH=amd64 go build -o installer-contents/weeve-agent-macos-amd64    ./cmd/agent/agent.go
	GOOS=darwin  GOARCH=arm64 go build -o installer-contents/weeve-agent-macos-arm64    ./cmd/agent/agent.go
.PHONY: cross

secunet:
	GOOS=linux GOARCH=amd64 go build -o bin/agent_secunet -tags secunet cmd/agent/agent.go
	docker build -f Dockerfile.secunet -t secunet-test .
.PHONY: secunet

build-all: build-arm build-x86 build-darwin
.PHONY: build-all
