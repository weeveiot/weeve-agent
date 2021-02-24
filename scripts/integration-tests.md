# Integration with the docker client
With Docker installed in the local machine, the Docker daemon will be running as the `dockerd` process. The Docker daemon is a server, exposing the Docker API. The node server component interacts with the Docker daemon through this API, using the Docker Golang SDK. The node server is acting as the client.

In this series of integration tests, the node server will interact with the installed and running Docker software.

There will be no integration with Docker hub, or any container registry.

## Preparation
This series of tests relies on having images locally on the test machine. In this series of tests, no docker container registry is available.

```
docker image pull weevenetwork/mosquitto_broker:demo
```

## Tests
Given a known and simple working manifest file as follows:

```json
{
    "id": 42222,
    "manifestId": "515cd0631639793a593c1002acf2ca978a61cc835eda75ea334079557045a41c",
    "version": "1.0.0",
    "name": "MVP Manifest",
    "organizationId": "2d83e5aa819747ff3b5d055adda4b2257a5ec8d96634ad0f8923d3b9bbde75e3",
    "groupId": "245536acb08b0958f1aa78d391d5efc773e60fa2c39d942318aa04d6085bef40",
    "description": "Minimum Viable Manifest",
    "compose": {
        "network":
            {
                "name": "mosquitto-net",
                "driver": "bridge"
            },
        "services": [
            {
                "moduelId": "245536acb08b0958f1aa78d391d5efc773e60fa2c39d942318aa04d6085bef42",
                "name": "broker",
                "description": "Standard Mosquitto broker",
                "version": "1.0.0",
                "network": "mosquitto-net",
                "image": {
                    "name": "weevenetwork/mosquitto_broker",
                    "tag": "demo"
                },
                "command": [
                    {
                        "key": "p",
                        "value": "1880"
                    }
                ]
            }
        ]
    }
}
```

1. Assert that a known working manifest file with a known local image
1.


# Integration with the docker registry
In this series of tests, the docker registry will be running and tested.