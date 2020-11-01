#/bin/bash

export DOCKER_REGISTRY='weevenetwork'
export NODE_SERVICE_VERSION='v0.0.1'
export DOCKER_CLI_EXPERIMENTAL=enabled
export DOCKER_BUILDKIT=1
export DOCKER_USER='weevenetwork'
sum=$(sha256sum Dockerfile| cut -f1 -d' ')


docker build --platform=local -o . git://github.com/docker/buildx
mkdir -p ~/.docker/cli-plugins && mv buildx ~/.docker/cli-plugins/docker-buildx

docker run --rm --privileged docker/binfmt:a7996909642ee92942dcd6cff44b9b95f08dad64

# Inspection
ls -al /proc/sys/fs/binfmt_misc/
cat /proc/sys/fs/binfmt_misc/qemu-aarch64

docker buildx create --name mybuilder
docker buildx use mybuilder
docker buildx inspect --bootstrap

docker buildx build --platform linux/amd64,linux/arm64,linux/386,linux/arm/v7,linux/arm/v6 -t "${DOCKER_USER}/node-service:${NODE_SERVICE_VERSION}" -t "${DOCKER_USER}/node-service:latest" -t "${DOCKER_USER}/manager-backend:${sum}" --push .

echo
docker images | grep node-service
echo
