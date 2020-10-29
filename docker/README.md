# node-service docker container

Maintainer: kanchen Monnin

------



## Description

Receives a manifest and instantiates a manifest into a running data service. Returns status to manager. Installed in the client's premise on many nodes and is connected to a data source.

- Repository: https://gitlab.com/weeve/edge-server/edge-pipeline-service 
- Documentation: https://gitlab.com/weeve/edge-server/edge-pipeline-service/-/blob/dev/README.md


## Build
- copy golang directory structure to /opt
- Enable experimental features in the Docker CLI, modify the file ~/.docker/config.json .
```
{
"experimental": "enabled"
}
```
- ./build.sh             (docker build/tagging/push)



## Usage

- $ dopcker run -it -p 8030:8030weevenetwork/node-service:<tag>
- $ docker ps
- $ docker logs node-service -f
- $ docker exec -ti node-service sh
- ...



## Point your browser to ...

- http://0.0.0.0:8030
