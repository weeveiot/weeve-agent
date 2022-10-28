# weeve agent

The weeve agent is a lightweight service to orchestrate edge applications.
An edge application is defined in a manifest file and consists of several interconnected docker containers (modules), building a data pipeline.
The edge applications are orchestrated by the Manager API (MAPI) over MQTT.
With the orchestration messages MAPI is able to deploy, undeploy, stop, resume and remove edge applications on a node.
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
| Parameter | Required | Description | Possible Values | Default |
| ----------- | -------- | ------------------------------------------------------------ | ------------------------------------- | --------- |
| configpath | true | path of the JSON file with node configuration | | |
| release | true | path of the JSON file with node configuration | prod, dev | |
| test | false | set to 'true' to build agent from local and run | false, true | false |
| broker | false | URL of the MQTT broker to connect | | |
| loglevel | false | level of log verbosity | debug, info, warning, error | info |
| heartbeat | false | time period of heartbeat messages (sec) | | |

### Uninstallation

```bash
curl -sO http://weeve-agent-dev.s3.amazonaws.com/weeve-agent-uninstaller.sh && sh weeve-agent-uninstaller.sh
```

## Parameters

The weeve agent depends on configuration for execution.
The configuration of the agent includes describing how the agent connects to a backend server, and the behaviour of the weeve agent.

The weeve agent can be configured using a configuration file (by specifying `--config` flag), directly with command line arguments, or a combination of both.

Configuration parameters are listed in the table below with defaults, or can be displayed with the `agent --help` command.

| Parameter   | Short | Required | Description                                                     | Default         |
| ----------- | ----- | -------- | --------------------------------------------------------------- | --------------- |
| version     | v     | false    | Print version information and exit                              |                 |
| broker      | b     | true     | URL of the MQTT broker to connect                               |                 |
| id          | i     | true     | ID of this node                                                 |                 |
| name        | n     | true     | Name of the node                                                |                 |
| notls       |       | false    | For developers - disable TLS for MQTT                           | false           |
| password    |       | false    | Password for TLS                                                | ""              |
| rootcert    |       | false    | Path to MQTT broker (server) certificate                        | ca.crt          |
| loglevel    | l     | false    | Set the logging level                                           | info            |
| logfilename |       | false    | Set the name of the log file                                    | Weeve_Agent.log |
| logsize     |       | false    | Set the size of each log files (MB)                             | 1               |
| logage      |       | false    | Set the time period to retain the log files (days)              | 1               |
| logbackup   |       | false    | Set the max number of log files to retain                       | 5               |
| logcompress |       | false    | Compress the log files                                          | false           |
| mqttlogs    |       | false    | For developers - Display detailed MQTT logging messages         | false           |
| heartbeat   | t     | false    | Time period between heartbeat messages (sec)                    | 10              |
| logsendinvl |       | false    | Time period between sending edge app logs (sec)                 | 60              |
| out         |       | false    | Print logs to stdout                                            | false           |
| config      |       | false    | Path to the .json config file                                   |                 |
| manifest    |       | false    | For developers - Path to the .json manifest file to be deployed |                 |

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

After the initial setup the agent publishes it public key to MAPI, subscribes on the topic <nodeId>/orchestration and waits for incoming commands from MAPI. It additionally subscribes to <nodeId>/orgKey to receive the secret organization key, that will be used to decrypt secret parameters shared in the manifests from MAPI.
ATTENTION: the key sharing function is meant to only be used over secure communication channel. Never use it with `--notls` option!

The agent also publishes a status message to <nodeId>/nodestatus every `heartbeat` seconds, which includes the status of the node, the running edge apps and their modules as well as an overview of the available node ressources.

### Local setup

#### Prerequisites

GoLang is installed https://golang.org/doc/install

Run a MQTT broker on your local machine, for example:

```bash
MQTT_PORT=8083
docker run --rm --name mosquitto -p $MQTT_PORT:1883 eclipse-mosquitto:2.0.14 mosquitto -v -c /mosquitto-no-auth.conf
```

Get a configuration file from the UI by creating a new node or construct your own using `agent-conf.json.example` as an example.
E.g. fill the fields `NodeId` and `NodeName` with unique values, also set the `Registered` field to `true`.

Build the agent binary from the project root folder

```bash
make build-<your arch>
```

And run it locally with your preffered configuration, for example

```bash
./bin/weeve-agent-<os>-<arch> --out --notls --broker=mqtt://localhost:$MQTT_PORT --loglevel=debug --config agent-conf.json
```

The mosquitto client can be used to publish the messages to the agent.

Example messages can be found in `testdata`.

```bash
mosquitto_pub -L mqtt://localhost:$MQTT_PORT/<nodeId>/orchestration -f testdata/test_manifest.json
```

You can observe the agent's status messages by subscribing to the corresponding topic:

```bash
mosquitto_sub -L mqtt://localhost:$MQTT_PORT/nodestatus/<nodeId>
```

### Unit testing

To execute all unit tests run the following command from the project root directory

```bash
go test -v ./...
```

## Containerization

Weeve agent can also run in a container, given the right environment. Currently we support container orchestration in the secunet container environment. To create a container run `make secunet` in the top project directory. This will create a container `secunet-test` ready to be deployed on a secunet gateway. It can then be deployed using the repository [secunet deployment](https://github.com/weeveiot/secunet-deployment).
