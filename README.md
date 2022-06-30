# weeve agent
The weeve agent is a lightweight service to orchestrate data pipelines. A data pipeline is defined in a manifest file and consists of several interconnected docker containers. The data pipeline is instantiated by subscription to an MQTT topic for the stage and node. The logic of the service then pulls images from docker hub if they do not exist on the machine. The weeve agent creates and starts containers based on a specified manifest. A bridge network is instantiated to facilitate container communication. The agent publishes status messages over MQTT on a defined interval to monitor the state of the IOT edge comprised of multiple edge nodes running weeve agents.

# Installing weeve agent using installer script
## Requirements

The Github personal access key is required to download the contents for the agents.
Please make sure there is a file in the local machine containing the Github Personal Access Token.

## Installation

```bash
curl -s https://raw.githubusercontent.com/weeveiot/weeve-agent/<BRANCH>/weeve-agent-installer.sh > weeve-agent-installer.sh
```

```bash
sudo sh weeve-agent-installer.sh tokenpath=<path to the file containing the token>
```
| Parameter   | Required | Description                                                 | Possible Values            | Default   |
| ----------- | -------- | ------------------------------------------------------------| ---------------------------|-----------|
| tokenpath   | true     | takes the path of the file containing the access token      |                            |           |
| configpath  | false    | takes the path of the JSON file with node configuration     |                            |           |
| environment | false    | name of the environment where the agent is to be registered | dev, demo, sandbox, wohnio |           |
| release     | false    | to select which release of agent is to be installed         | stable, dev                |           |
| nodename    | false    | takes the name of the node                                  |                            |           |
| test        | false    | set to 'true' to build agent from local and run             |                            | false     |

## Un-installation

```bash
curl -s https://raw.githubusercontent.com/weeveiot/weeve-agent/<BRANCH>/weeve-agent-uninstaller.sh | sudo sh

```

## Architecture
The weeve agent can be considered as a Docker orchestration layer with a purpose built business logic for a data service - multiple containers in communication with each other. As such, the project relies on the [Golang Docker SDK](https://godoc.org/github.com/docker/docker).

The main entry command initiates logging, parses flags, and passes control to the publish and subscribe MQTT client software. The [paho](github.com/eclipse/paho.mqtt.golang) MQTT client is used for MQTT communication. TLS is optionally configurable, and has both server and client authentication, and therefore requires the private and public key of the device as well as the certificate of the server.

A data model for the manifest object and supporting structures is found in the internal library.

[WIP] A simple web server is implemented with the [Gorilla MUX package](github.com/gorilla/mux) for debugging and inspection of the weeve agent and the device.

## Command Parameters

| Parameter   | Short | Required | Description                                            | Default  |
| ----------- | ----- | -------- | ------------------------------------------------------ | -------- |
| out         |       | false    | Print logs to stdout                                   | false    |
| broker      | b     | true     | Broker to connect                                      |          |
| heartbeat   | t     | false    | Heartbeat time in seconds                              | 30       |
| mqttlogs    | m     | false    | For developer - Display detailed MQTT logging messages |          |
| notls       |       | false    | For developer - disable TLS for MQTT                   |          |
| loglevel    | l     | false    | Set the logging level                                  | info     |
| logfilename |       | false    | Set the name of the log file                           | logs     |
| logsize     |       | false    | Set the size of each log files (MB)                    | 1        |
| logage      |       | false    | Set the time period to retain the log files (days)     | 1        |
| logbackup   |       | false    | Set the max number of log files to retain              | 5        |
| logcompress |       | false    | To compress the log files                              |          |
| nodeId      | i     | false    | ID of this node                                        | register |
| name        | n     | false    | Name of this node to be registered                     |          |
| rootcert    | r     | false    | Path to MQTT broker (server) certificate               |          |
| config      |       | false    | Path to the .json config file                          |<exe dir> |

## Getting started

### Without TLS
The binary or the source code, (with `go run`) the agent can be started with the following options.

#### Terminal: Broker
This recipe requires a local MQTT broker, for example, the Mosquitto broker with logs enabled to confirm subscription and publish; `mosquitto -v -p 8080`.

#### Terminal: Publishing to weeve agent
With the mosquitto client tools, we can publish to the weeve agent in interactive mode (enter to send a message).
`mosquitto_pub -t nodes/localtest/local-test-node-1/deploy -p 8080 -l`

Of to send a sample manifest file, the `-f` option can be used.
`mosquitto_pub -f ./testdata/manifest/mvp-manifest.json -p 8080 -t nodes/localtest/local-test-node-1/deploy`

#### Terminal: Subscription to agent status
In a new terminal instance, subscribe to all topics for that broker; `mosquitto_sub -t '#' -p 8080`.

#### Terminal: Weeve agent
In a final terminal, run the weeve agent and connect to the broker to publish status messages;
```bash
go run <path-to-agent>/agent.go --out --notls \
	--broker tcp://localhost:8080 \ # Broker to connect to \
	--heartbeat 3 # Status message publishing interval
	--nodeId local-test-node-1 \ # ID of this node optional \
	--config <path-to-config>
```

The `--out` flag enables logs in terminal, and the `--notls` flag disables TLS configuation. Further logs from the `paho` MQTT client can be enabled with the `--mqttlogs` flag, and the `--loglevel` flag enables to set desired logging level.

### With TLS

#### Manual Node Provisioning
Start with registering a new Node with GraphQL or with weeve UI. Then download corresponding pem certificate and key from AWS S3 to your device as well as AmazonRootCA1.pem from Google.
Since TLS configuration requires the full path to all secrets and certificates, execute the following code:

```bash
SERVER_CERTIFICATE=<path-to-root-cert>/ca.crt
go run <path-to-agent>/agent.go --out \
	--nodeId awsdev-test-node-1 \ # ID of this node (optional here)\
	--name awsdev-test-node-1 \ # Name of this node (optional here)\
	--broker tls://<broker host url>:8883 \ # Broker to connect to \
	--heartbeat 10   # Status message publishing interval \
	--config <path-to-config>
	# --mqttlogs # Enable detailed debug logs for the MQTT connection
```

The `tls` protocol is strictly checked in the Broker url.

#### Automatic Node Provisioning
Clone weeve Agent code to your device and place bootstrap (group) certificates in its directory.
Download AmazonRootCA1.pem from Google. Then, follow the steps:

1) Set up nodeconfig.json with bootstrap details (see [Config Options](#config-options)):
```json
{
	"AWSRootCert": "/path/to/AmazonRootCA1.pem",
	"Certificate": "/path/to/<bootstrap_id>-certificate.pem.crt",
	"NodeId": "",
	"NodeName": "Node-Sample-1",
	"PrivateKey": "/path/to/<bootstrap_id>-private.pem.key"
}
```

1) Run command:
```bash
go run <path-to-agent>/agent.go --out \
	--broker tls://<broker host url>:8883 \ # Broker to connect to \
	--heartbeat 60 # Status message publishing interval \
	--config <path-to-config>
```

# Config options
All the below params can be updated into json instead of arguments as above
```json
{
	"AWSRootCert": "/path/to/AmazonRootCA1.pem",
	"PrivateKey": "/path/to/<node private key/bootstrap private key file name>",
	"Certificate": "/path/to/<node certificate/bootstrap certificate file name>",
	"NodeId": "<node id>", //Empty initially for auto registration
	"NodeName": "<node name>" //Node name for auto registration
}
```

# Containerization
Weeve agent can also run in a container given the right environment. Currently we support container orchestration in the secunet container environment. To create a container run `make secunet` in the top project directory. This will create a container `secunet-test` ready to be deployed on a secunet gateway. It can then be deployed using the repository [secunet deployment](https://github.com/weeveiot/secunet-deployment).

# Developer guide
```
go build -o ./build/agent ./cmd/agent/agent.go
```

# [BELOW IS WIP]

### Docker container
Currently, running the project with Docker is not supported. Since the main function of the Weeve Node Service is to orchestrate a set of docker containers, running the project inside docker presents additional complexities due to the interaction with the host machine. A docker file is present to facilitate unit testing only.

## Getting started for Developers

### Prerequisites
GoLang is installed https://golang.org/doc/install

### Build the Golang project
The project can be compiled and run from source. The root of the command is the project root directory.
Build Node API Service mode
`go build -o ./build/node-service ./cmd/node-service.go`

Build Node MQTT listener mode
`go build ./cmd/node_listener.go`

### Unit-test the Golang project

TO clear all docker resoures before running tests, run below command,

>docker system prune -a

To get latest test result first clear cached result if any, using below command,

>go clean --testcache

Then run below command to run tests,
`go test -v ./...` OR 'watchtests.sh'

Currently, unit testing does not cover the project.

## Developer environment

Several developer features are supported in the project.

### Manually running the weeve agent as systemd service on a edge-node

1. Install docker [docker installation](https://docs.docker.com/engine/install/)
2. Create a new directory and copy the following to the directory
	1. weeve agent binary (AWS: s3 bucket)
	2. nodeconfig.json and bootstrap certificates (Github repository: weeve-agent-dependencies)
3. Make the binary executable `chmod u+x weeve-agent/<agent-binary-name>`
4. Create weeve-agent.service file which will define the service
```bash
[Unit]
Description=Weeve Agent
[Install]
WantedBy=multi-user.target
[Service]
Type=simple
Restart=always
RestartSec=60s
WorkingDirectory=<path to the directory containing weeve agent contents>
ExecStart=<weeve agent binary path> $ARG_STDOUT $ARG_BROKER $ARG_PUBLISH $ARG_HEARTBEAT <add more arguments as required >
```
1. Move .service file to `/lib/systemd/system/`
2. Enable the service to start at start-up `sudo systemctl enable weeve-agent`
3. Start the service `sudo systemctl start weeve-agent`

Upon first execution;
1. The weeve agent bootstraps.
2. The thing name will be the environment followed by the ID, for example; `awsdev_f5adbd1a-d4b7-4485-b5f4-2b901a92c80f`.
3. The certificate is created and uploaded to S3

NOTE:
It is possible to run multiple instances of the weeve agent in a single host. Each process would be running independently and be bootstrapped with as a dedicated IoT thing.

### Deleting a node

To delete the IoT thing, call the API - deleteNode.
This will remove the following:

- Things from IoT core
- Node and deployments from DB
- Certificate from s3 bucket

### Dependencies

### Enhanced golang terminal
The `go` command may be replaced with the `richgo` command to provide more colorful output at the terminal. The project is found at [richgo](https://github.com/kyoh86/richgo) and installed with `go get -u github.com/kyoh86/richgo`.

### File watcher reflex
A file watcher is employed to automate the restart of the server and run tests on code change.
The file selected watcher is [reflex](https://github.com/cespare/reflex), and is installed with `go get -u github.com/cespare/reflex`.

For unit tests;
`reflex -r '\.go$' -s -- sh -c 'richgo test -v ./...'`


# [WIP] Developer testing - listener


## Testing with TLS to IOT core
Traffic between the weeve agent and the MQTT broker is encrypted with TLS.

PEM or Privacy Enhanced Mail is a Base64 encoded DER certificate.

### Server authentication
The MQTT broker presents it's certificate, which the device validates against the a

### Client authentication

The server.key is likely your private key, and the .crt file is the returned, signed, x509 certificate.
(openssl x509 -noout -modulus -in certificate.pem.crt | openssl md5 ; openssl rsa -noout -modulus -in private.pem.key | openssl md5) | uniq
#### Testing with IOT core

Status heartbeat messages can be confirmed by subscribing to the NodeID topic. A test subscription [can be viewed](https://console.aws.amazon.com/iot/home?region=us-east-1#/test) if using AWS IOT Core.

The same testing UI can be used to publish back to the agent.

### Testing with mosquitto
mosquitto_pub \
	-h asnhp33z3nubs-ats.iot.us-east-1.amazonaws.com -p 8883 \
	--cafile ~/weeve/edge-pipeline-service/adcdbef7432bc42cdcae27b5e9b720851a9963dc0251689ae05e0f7f524b128c-certificate.pem.crt \
	-t test -m testing


mosquitto_pub -h asnhp33z3nubs-ats.iot.us-east-1.amazonaws.com -p 8883 --cafile AmazonRootCA1.pem -t test -m testing

## Testing with Wireshark

nslookup
ifconfig

## Testing the manifest
```bash
Local: mosquitto_pub -t nodes/localtest/{nodeId}/deploy -p 8080 -l

MANIFEST='{
	"id": "3ab346d8-55d3-44bb-a4bf-3f44ba6baa1e",
	"created": 1632981575399,
	"version": "1.0.0",
	"organizationId": "",
	"services": [
		{
			"image": {
				"name": "weevenetwork/dev-random",
				"tag": "latest"
			},
			"environments": [
				{
					"default": "",
					"name": "Volume Container",
					"options": null,
					"description": "Volume mount container.",
					"type": "string",
					"value": "/mnt/random",
					"key": "VOLUME_CONTAINER"
				}
			],
			"document": "{\"ports\":[],\"mounts\":[{\"Type\":\"bind\",\"Source\":\"/dev/urandom\",\"Target\":\"/mnt/random\",\"ReadOnly\":true}],\"restart_policy\":{\"condition\":\"on-failure\",\"delay\":\"10s\",\"max_attempts\":3,\"window\":\"120s\"}}",
			"name": "dev-randomv2",
			"icon": "https://icons-020-demo.s3.eu-central-1.amazonaws.com/USB.png",
			"description": "Ingress module mounting the dev/random device to generate a SHA256 hash string with bind type mounting.",
			"id": "b9830bc7-a0af-4672-b8de-d570414cabb2",
			"categories": [
				{
					"name": "Experimental",
					"id": "category"
				}
			],
			"type": "input",
			"version": "0.0.1",
			"commands": [
				{
					"default": "",
					"name": "Hash",
					"options": [
						"md5",
						"sha1",
						"sha256"
					],
					"description": "Hash type.",
					"type": "enum",
					"value": "sha256",
					"key": "hash"
				},
				{
					"default": "",
					"name": "Interval",
					"options": null,
					"description": "Sleep interval.",
					"type": "integer",
					"value": "30",
					"key": "interval"
				}
			],
			"tags": [
				"dev",
				"random"
			]
		},
		{
			"image": {
				"name": "weevenetwork/hash-to-int",
				"tag": "latest"
			},
			"environments": null,
			"document": "{\"ports\":[],\"volumes\":[],\"restart_policy\":{\"condition\":\"on-failure\",\"delay\":\"10s\",\"max_attempts\":3,\"window\":\"120s\"}}",
			"name": "hash-to-int",
			"icon": "https://icons-020-demo.s3.eu-central-1.amazonaws.com/USB.png",
			"description": "Return the integer representation of a byte string.",
			"id": "6bc53b83-eb48-4cb5-bb25-a772c4a1ae85",
			"categories": [
				{
					"name": "Experimental",
					"id": "category"
				}
			],
			"type": "process",
			"version": "0.0.1",
			"commands": null,
			"tags": [
				"hash",
				"integer",
				"byte",
				"string"
			]
		},
		{
			"image": {
				"name": "weevenetwork/http-egress",
				"tag": "latest"
			},
			"environments": [
				{
					"default": "",
					"name": "HTTP URL",
					"options": null,
					"description": "Paste the HTTP address.",
					"type": "string",
					"value": "https://hookb.in/2qla9bN0JLFdzq88zqko",
					"key": "EGRESS_WEBHOOK_URL"
				},
				{
					"default": "",
					"name": "Method",
					"options": [
						"POST",
						"GET"
					],
					"description": "HTTP ReST method (recommended POST).",
					"type": "enum",
					"value": "POST",
					"key": "METHOD"
				},
				{
					"default": "",
					"name": "Input Labels",
					"options": null,
					"description": "List of comma (,) separated labels to read from a previous module. Leave empty (\"\") to keep all data.",
					"type": "string",
					"value": "",
					"key": "LABELS"
				},
				{
					"default": "",
					"name": "Timestamp Field",
					"options": null,
					"description": "Label for timestamp field for incoming data. If left empty, this module will add the timestamp.",
					"type": "string",
					"value": "Time",
					"key": "TIMESTAMP"
				}
			],
			"document": "{\"ports\":[],\"volumes\":[],\"restart_policy\":{\"condition\":\"on-failure\",\"delay\":\"10s\",\"max_attempts\":3,\"window\":\"120s\"}}",
			"name": "HTTP Egress",
			"icon": "https://icons-020-demo.s3.eu-central-1.amazonaws.com/HTTP.png",
			"description": "Send your data to a third party via HTTP",
			"id": "88352247-1ba8-4cf7-8e9c-7b5242e34cde",
			"categories": [
				{
					"name": "Egress",
					"id": "category"
				}
			],
			"type": "output",
			"version": "1.0.0",
			"commands": [],
			"tags": [
				"HTTP Egress",
				"output",
				"Send",
				"data",
				"third",
				"party",
				"http"
			]
		}
	],
	"networks": {
		"driver": "bridge",
		"name": "tick-network"
	},
	"description": "MVPDataServiceV2",
	"document": "",
	"name": "MVPDataServiceV2",
	"modified": 1632981575399
}'

```
