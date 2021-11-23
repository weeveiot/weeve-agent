NODE_ID=7ced3826-7738-4c9f-84b8-9302cb436e89


SERVER_CERTIFICATE=./AmazonRootCA1.pem
CLIENT_CERTIFICATE=./${NODE_ID}-certificate.pem.crt
CLIENT_PRIVATE_KEY=./${NODE_ID}-private.pem.key

.DEFAULT_GOAL := build

build-arm:
	rm -rf /build/arm.tar ./build/arm/*
	cp ${SERVER_CERTIFICATE} ${CLIENT_CERTIFICATE} ${CLIENT_PRIVATE_KEY} start.sh ./build/arm
	GOOS=linux GOARCH=arm GOARM=7 go build -o build/arm/agent ./cmd/agent/agent.go
	tar -cvf ./build/arm.tar -C ./build arm
.PHONY: build-arm


build-x86:
	rm -rf ./build/x86.tar ./build/x86/*
	cp ${SERVER_CERTIFICATE} ${CLIENT_CERTIFICATE} ${CLIENT_PRIVATE_KEY} start.sh ./build/x86
	GOOS=linux GOARCH=amd64 go build -o build/x86/agent ./cmd/agent/agent.go
	tar -cvf ./build/x86.tar -C ./build x86
.PHONY: build-x86

build-darwin:
	rm -rf ./build/darwin.tar ./build/darwin/*
	cp ${SERVER_CERTIFICATE} ${CLIENT_CERTIFICATE} ${CLIENT_PRIVATE_KEY} start.sh ./build/darwin
	GOOS=darwin GOARCH=amd64 go build -o build/darwin/agent ./cmd/agent/agent.go
	tar -cvf ./build/darwin.tar -C ./build darwin
.PHONY: build-darwin

build-all: build-arm build-x86 build-darwin