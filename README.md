# weeve agent
The weeve agent is a lightweight service to orchestrate data pipelines. A data pipeline is defined in a manifest file and consists of several interconnected docker containers. The data pipeline is instantiated by subscription to an MQTT topic for the stage and node. The logic of the service then pulls images from docker hub if they do not exist on the machine. The weeve agent then creates and starts containers based on request manifest. A bridge networks can be instantiated to facilitate container communication. The agent publishes status messages on a defined interval to monitor the state of the IOT edge comprised of multiple edge nodes running weeve agents.

## Architecture
The weeve agent can be considered as a Docker orchestration layer with a purpose built business logic for a data service - multiple containers in communication with each other. As such, the project relies on the [Golang Docker SDK](https://godoc.org/github.com/docker/docker).

The main entry command initiates logging, parses flags, and passes control to the web server. The web server is implemented with the [Gorilla MUX package](github.com/gorilla/mux).

A data model for the manifest object and supporting structures is found in the internal library.

## Getting started for Users

### Compiled binary
The latest binary can be downloaded from S3 bucket :
https://weeve-binaries-release.s3.eu-central-1.amazonaws.com/node-service/node-service-0-1-1

The compiled binary found as a release can be executed by specifying the port to be exposed;

`./node-service -v -p 8030`
The `-v` verbose flag is optional and will present the Debug level logging messages.

### Docker container
Currently, running the project with Docker is not supported. Since the main function of the Weeve Node Service is to orchestrate a set of docker containers, running the project inside docker presents additional complexities due to the interaction with the host machine. A docker file is present to facilitate unit testing only.

## Documentation
A postman collection is found in the /docs folder. The collection is (published)[https://documenter.getpostman.com/view/12141960/TVYQ3ubM].

## Getting started for Developers

### Build the Golang project
The project can be compiled and run from source. The root of the command is the project root directory.
`go build -o ./build/node-service ./cmd/node-service.go`

### Run the Golang project
`go run ./cmd/node-service.go -v -p 8030`

The root of the command is the project root directory.

The `-v` verbose flag is optional and will present the Debug level logging messages.

### Unit-test the Golang project

`go test -v ./...`

Currently, unit testing does not cover the project.

## Developer environment

Several developer features are supported in the project.

### Dependencies

### Enhanced golang terminal
The `go` command may be replaced with the `richgo` command to provide more colorful output at the terminal. The project is found at [richgo](https://github.com/kyoh86/richgo) and installed with `go get -u github.com/kyoh86/richgo
`.

### File watcher reflex
A file watcher is employed to automate the restart of the server and run tests on code change.
The file selected watcher is [reflex](https://github.com/cespare/reflex), and is installed with `go get -u github.com/cespare/reflex`.

The server can be started with;
`reflex -r '\.go$' -s -- sh -c 'go run ./cmd/node-service.go -v -p 8030'`

Similarly for unit tests;
`reflex -r '\.go$' -s -- sh -c 'richgo test -v ./...'`


**Endpoint**

POST: {EDGE_PIPELINE_URL}/pipelines

Request Body:

```
{
    "ID":"manifest2",
	"Name": "ManifestSingleContainerWithParameters",
	"Modules": [
    {
		"Index": 0,
		"Name": "c1",
		"Tag": "working",
		"ImageID": "sha256:a99a6700a30478ce4af059543a0aaac139eea3c85ff62b2603c9d53b4cc42657",
		"ImageName": "weevenetwork/go-mqtt-gobot",
        "options": [
            {"opt":"network", "val":"host"}
            ],
        "arguments": [
            {"arg":"InBroker", "val":"localhost:1883"},
            {"arg":"ProcessName", "val":"container-1"},
            {"arg":"InTopic", "val":"topic/source"},
            {"arg":"InClient", "val":"weevenetwork/go-mqtt-gobot"},
            {"arg":"OutBroker", "val":"localhost:1883"},
            {"arg":"OutTopic", "val":"topic/c2"},
            {"arg":"OutClient", "val":"weevenetwork/go-mqtt-gobot"}
        ]
	}
    ]
}
```

go run ./listener/node_listener.go -v -i demo_edge_node1 -b tls://asnhp33z3nubs-ats.iot.us-east-1.amazonaws.com:8883 -f efbb87beed -s nodes/awsdev -c manager/awsdev -t CheckVersion -u hssss -p 8030

go run ./listener/node_listener.go -v \
    --nodeId demo_edge_node1 \ # ID of this node \
    --broker tls://asnhp33z3nubs-ats.iot.us-east-1.amazonaws.com:8883 \ # Broker to connect to \
    --cert adcdbef7432bc42cdcae27b5e9b720851a9963dc0251689ae05e0f7f524b128c \ # Certificate to connect Broker \
    --subClientId nodes/awsdev \ # Subscriber ClientId \
    --pubClientId manager/awsdev \ # Publisher ClientId \
    --publish CheckVersion \ # Topic Name \
    --publicurl hssss \ # Public URL to connect from public \
    --nodeport 8030 \ # Port where edge node api is listening


# [WIP] Developer testing - listener

In one terminal, run a local broker with logs enabled to confirm subscription and publish; `mosquitto -v -p 8080`.

In a second terminal, subscribe to all topics for that broker; `mosquitto_sub -t '#' -p 8080`.

Run the weeve agent in a third terminal, with the local broker as the target. Disable TLS with the `--notls` flag.

```bash
go run ./listener/node_listener.go -v --notls --heartbeat 3 \
    --nodeId local-test-node-1 \ # ID of this node \
    --broker localhost:8080 \ # Broker to connect to \
    --cert adcdbef7432bc42cdcae27b5e9b720851a9963dc0251689ae05e0f7f524b128c \ # Certificate to connect Broker \
    --subClientId nodes/localtest \ # Subscriber ClientId \
    --pubClientId manager/localtest \ # Publisher ClientId \
    --publish CheckVersion \ # Topic Name \
    --publicurl hssss \ # Public URL to connect from public \
    --nodeport 8030 \ # Port where edge node api is listening
```

## Testing with TLS to IOT core

go run ./listener/node_listener.go -v  --heartbeat 10 \
    --nodeId awsdev-test-node-1 \ # ID of this node \
    --broker tls://asnhp33z3nubs-ats.iot.us-east-1.amazonaws.com:8883 \ # Broker to connect to \
    --cert adcdbef7432bc42cdcae27b5e9b720851a9963dc0251689ae05e0f7f524b128c \ # Certificate to connect Broker \
    --subClientId nodes/awsdev \ # Subscriber ClientId \
    --pubClientId manager/awsdev \ # Publisher ClientId \
    --publish CheckVersion \ # Topic Name \
    --publicurl hssss \ # Public URL to connect from public \
    --nodeport 8030 \ # Port where edge node api is listening

mosquitto_pub \
    -h asnhp33z3nubs-ats.iot.us-east-1.amazonaws.com -p 8883 \
    --cafile ~/weeve/edge-pipeline-service/adcdbef7432bc42cdcae27b5e9b720851a9963dc0251689ae05e0f7f524b128c-certificate.pem.crt \
    -t test -m testing


mosquitto_pub -h asnhp33z3nubs-ats.iot.us-east-1.amazonaws.com -p 8883 --cafile AmazonRootCA1.pem -t test -m testing