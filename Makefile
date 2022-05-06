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

secunet:
	GOOS=linux GOARCH=amd64 go build -o bin/agent_secunet -tags secunet cmd/agent/agent.go
	docker build -f Dockerfile.secunet -t secunet-test .

build-all: build-arm build-x86 build-darwin