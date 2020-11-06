# Weeve Node Service
The Weeve Node Service is a lightweight service to orchestrate data pipelines. A data pipeline is defined in a manifest file and consists of several interconnected docker containers. The data pipeline is instantiated by POST request to the "/pipelines" endpoint. The logic of the service then pulls images from docker hub if they do not exist on the machine. The Weeve Node Service then creates and starts containers based on request manifest. A bridge networks can be instantiated to facilitate container communication. Several resources exist and are exposed in a RESTful API;
- Docker images
- Docker containers
- Data pipelines

## Getting Started for Users

The compiled binary found as a release can be executed by specifying the port to be exposed;

go run ./cmd/node-service.go -v -p 8030

The `-v` verbose flag is optional and will present the Debug level logging messages.



Go to project root path and rub below commands to build and run code in development enviroment,

`go build ./cmd/node-service.go`

`go run ./cmd/node-service.go -p 8030`

## Test pipelines endpoint

To test pipelines endpoint use below sample request,

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


# Dev
## reflex
Using the [reflex](https://github.com/cespare/reflex) file watcher;
(Install with `go get github.com/cespare/reflex`)
`reflex -r '\.go$' -s -- sh -c 'go run ./cmd/node-service.go -v -p 8030'`

Running the server;
`go run ./cmd/node-service.go --port 8030`

make build

## Docker notes
`docker container rm $(docker container ls -aq)   `

