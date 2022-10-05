# weeve agent
The weeve agent is a lightweight service to orchestrate edge applications.
An edge application is defined in a manifest file and consists of several interconnected docker containers (modules), building a data pipeline.
The edge applications are orchestrated by the Manager API (MAPI) over MQTT.
With the orchestration messages MAPI is able to deploy, undeploy, start, stop and remove edge applications on a node.
The agent periodically publishes status messages over MQTT back to MAPI to monitor the state of the edge apps running on the node.

## Quick-start
### Requirements
Right now weeve agent can run on Linux systems with a Docker installation on the following architectures:
- ARM
- ARM64
- x86_64

### Prerequisites
The node needs to be registered first to set a node name and acquire a node ID from the database.
For this follow our [quick setup guide](https://docs.weeve.engineering/guides/installing-the-weeve-agent).
If the node is already registered, please fill the fields `nodeId` and `nodeName` in the config file `nodeconfig.json`.

### Installation
Execute this one-line installer with the path to your node configuration file:
```bash
curl -sO http://weeve-agent-dev.s3.amazonaws.com/weeve-agent-installer.sh && sh weeve-agent-installer.sh configpath=<path-to-config-file> release=prod
```

The installer script can take the following optional parameters:
| Parameter   | Required | Description                                                  | Possible Values                       | Default   |
| ----------- | -------- | ------------------------------------------------------------ | ------------------------------------- | --------- |
| configpath  | true     | path of the JSON file with node configuration                |                                       |           |
| release     | true     | path of the JSON file with node configuration                | prod, dev                             |           |
| test        | false    | set to 'true' to build agent from local and run              | false, true                           | false     |
| broker      | false    | URL of the MQTT broker to connect                            |                                       |           |
| loglevel    | false    | level of log verbosity                                       | debug, info, warning, error           | info      |
| heartbeat   | false    | time period of heartbeat messages (sec)                      |                                       |           |

### Uninstallation
```bash
curl -sO http://weeve-agent-dev.s3.amazonaws.com/weeve-agent-uninstaller.sh && sh weeve-agent-uninstaller.sh
```

## CLI parameters for weeve agent
Also use `agent -h` or `agent --help` for help.

| Parameter   | Short | Required | Description                                            | Default         |
| ----------- | ----- | -------- | ------------------------------------------------------ | --------------- |
| broker      | b     | true     | URL of the MQTT broker to connect                      |                 |
| out         |       | false    | Print logs to stdout                                   | false           |
| heartbeat   | t     | false    | Time period of heartbeat messages (sec)                | 30              |
| mqttlogs    | m     | false    | For developer - Display detailed MQTT logging messages |                 |
| notls       |       | false    | For developer - disable TLS for MQTT                   |                 |
| loglevel    | l     | false    | Set the logging level                                  | info            |
| logfilename |       | false    | Set the name of the log file                           | Weeve_Agent.log |
| logsize     |       | false    | Set the size of each log files (MB)                    | 1               |
| logage      |       | false    | Set the time period to retain the log files (days)     | 1               |
| logbackup   |       | false    | Set the max number of log files to retain              | 5               |
| logcompress |       | false    | Compress the log files                                 | false           |
| nodeId      | i     | false    | ID of this node                                        |                 |
| name        | n     | false    | Name of the node                                       |                 |
| rootcert    | r     | false    | Path to MQTT broker (server) certificate               |                 |
| config      |       | false    | Path to the .json config file                          |<exe dir>        |
| manifest    |       | false    | Path to the .json manifest file to be deployed         |                 |

## Documentation
See the official technical documentation on https://docs.weeve.engineering/.

## Developer guide
This section is a guide for developers intending to testing and developing the agent locally.

### Application architecture
The weeve agent can be considered as a Docker orchestration layer with a purpose built business logic for a data service - multiple containers in communication with each other.
As such, the project relies on the [Golang Docker SDK](https://godoc.org/github.com/docker/docker).

The main entry command initiates logging, parses flags, and passes control to the publish and subscribe MQTT client software.
The [paho](github.com/eclipse/paho.mqtt.golang) MQTT client is used for MQTT communication.
TLS is optionally configurable, and supports server authentication, therefore a CA certificate used to sign the certificate needs to be provided.

After the initial setup the agent subscribes on the topic <nodeId>/orchestration and waits for incoming commands from MAPI.
It also publishes a status message to <nodeId>/nodestatus every `heartbeat` seconds, which includes the status of the node, the running edge apps and their modules as well as an overview of the available node ressources.

### Local setup
#### Prerequisites
GoLang is installed https://golang.org/doc/install

Run a MQTT broker on your local machine, for example:

```bash
MQTT_PORT=8083
docker run --rm --name mosquitto -p $MQTT_PORT:1883 eclipse-mosquitto:2.0.14 mosquitto -v -c /mosquitto-no-auth.conf
```

Edit `nodeconfig.json` and fill the fields `NodeId` and `NodeName` with unique values, also set the `Registered` field to `true` e.g.:
```json
{
 "RootCertPath": "",
 "NodeId": "1234567890",
 "Password": "",
 "APIkey": "",
 "NodeName": "LocalTestNode",
 "Registered": true
}
```

Build the agent binary from the project root folder;

```bash
go build -o ./bin/agent ./cmd/agent/agent.go
```

And run it locally with your preffered configuration, for example;

```bash
./bin/agent --out --notls --broker=mqtt://localhost:$MQTT_PORT --heartbeat=300 --loglevel=debug --config nodeconfig.json
```

The mosquitto client can be used to publish the messages to the agent.

Example messages can be found in `testdata`.

```bash
mosquitto_pub -L mqtt://localhost:$MQTT_PORT/<nodeId>/orchestration -f test_manifest.json
```

You can observe the agent's status messages by subscribing to the corresponding topic:

```bash
mosquitto_sub -L mqtt://localhost:$MQTT_PORT/<nodeId>/nodestatus
```

### Unit testing
To execute all unit tests run the following command from the project root directory
```bash
go test -v ./...
```

## Containerization
Weeve agent can also run in a container, given the right environment. Currently we support container orchestration in the secunet container environment. To create a container run `make secunet` in the top project directory. This will create a container `secunet-test` ready to be deployed on a secunet gateway. It can then be deployed using the repository [secunet deployment](https://github.com/weeveiot/secunet-deployment).
