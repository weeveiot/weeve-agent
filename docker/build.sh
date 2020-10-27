#/bin/bash

export DOCKER_REGISTRY='weevenetwork'
export NODE_SERVICE_VERSION='v0.0.1'


## Build
docker buildx build --platform linux/arm/v7,linux/amd64 -t ${DOCKER_REGISTRY}/node-service:latest .

## Tagging
docker tag ${DOCKER_REGISTRY}/node-service:latest ${DOCKER_REGISTRY}/node-service:${NODE_SERVICE_VERSION}

## Push
docker push ${DOCKER_REGISTRY}/node-service:latest
docker push ${DOCKER_REGISTRY}/node-service:${NODE_SERVICE_VERSION}

echo
docker images | grep node-service
echo
