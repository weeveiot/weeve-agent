NODE_ID=7ced3826-7738-4c9f-84b8-9302cb436e89

AGENT_VERSION=0.9.0

SERVER_CERTIFICATE=./AmazonRootCA1.pem
CLIENT_CERTIFICATE=./${NODE_ID}-certificate.pem.crt
CLIENT_PRIVATE_KEY=./${NODE_ID}-private.pem.key

.DEFAULT_GOAL := build

build-x86:
	GOOS=linux GOARCH=amd64 go build -o build/weeve-agent-${AGENT_VERSION}-linux-x86 ./cmd/agent/agent.go
.PHONY: build-x86


build-arm:
	GOOS=linux GOARCH=arm GOARM=7 go build -o build/weeve-agent-${AGENT_VERSION}-linux-arm ./cmd/agent/agent.go
.PHONY: build-arm


build-darwin:
	GOOS=darwin GOARCH=amd64 go build -o build/weeve-agent-${AGENT_VERSION}-darwin ./cmd/agent/agent.go
.PHONY: build-darwin


cross:
	rm -rf build
	GOOS=linux   GOARCH=amd64 go build -o build/weeve-agent-${AGENT_VERSION}-linux-x86_64 ./cmd/agent/agent.go
	GOOS=linux   GOARCH=arm64 go build -o build/weeve-agent-${AGENT_VERSION}-linux-aarch64 ./cmd/agent/agent.go
	GOOS=linux   GOARM=6 GOARCH=arm go build -o build/weeve-agent-${AGENT_VERSION}-linux-armv6 ./cmd/agent/agent.go
	GOOS=linux   GOARM=7 GOARCH=arm go build -o build/weeve-agent-${AGENT_VERSION}-linux-armv7 ./cmd/agent/agent.go
	GOOS=darwin  GOARCH=amd64 go build -o build/weeve-agent-${AGENT_VERSION}-darwin-x86_64 ./cmd/agent/agent.go
	GOOS=darwin  GOARCH=arm64 go build -o build/weeve-agent-${AGENT_VERSION}-darwin-aarch64 ./cmd/agent/agent.go
	GOOS=windows GOARCH=amd64 go build -o build/weeve-agent-${AGENT_VERSION}-windows-x86_64.exe ./cmd/agent/agent.go
.PHONY: cross

secunet:
	GOOS=linux GOARCH=amd64 go build -o bin/agent_secunet -tags secunet cmd/agent/agent.go
	docker build -f Dockerfile.secunet -t secunet-test .

build-all: build-arm build-x86 build-darwin