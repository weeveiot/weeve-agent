#!/bin/sh

stage=$1
CI_REGISTRY=$2
CI_REGISTRY_USER=$3
CI_REGISTRY_PASSWORD=$4
listarch=$(go tool dist list)
# aix/ppc64        freebsd/amd64   linux/mipsle   openbsd/386
# android/386      freebsd/arm     linux/ppc64    openbsd/amd64
# android/amd64    illumos/amd64   linux/ppc64le  openbsd/arm
# android/arm      js/wasm         linux/s390x    openbsd/arm64
# android/arm64    linux/386       nacl/386       plan9/386
# darwin/386       linux/amd64     nacl/amd64p32  plan9/amd64
# darwin/amd64     linux/arm       nacl/arm       plan9/arm
# darwin/arm       linux/arm64     netbsd/386     solaris/amd64
# darwin/arm64     linux/mips      netbsd/amd64   windows/386
# dragonfly/amd64  linux/mips64    netbsd/arm     windows/amd64
# freebsd/386      linux/mips64le  netbsd/arm64   windows/arm

echo $stage
echo $listarch

mkdir bin
cross_pfrm=''

for arch in ${listarch[@]}
do
    # echo $arch
    cross_pfrm=$cross_pfrm,$arch
    arrIN=(${arch//// })
    echo ${arrIN[0]} ${arrIN[1]} agent_${arrIN[0]}_${arrIN[1]}
    GOOS=${arrIN[0]} GOARCH=${arrIN[1]} go build -o bin/agent_${arrIN[0]}_${arrIN[1]} cmd/agent/agent.go
done

aws s3 sync bin s3://weeve-resource-772697371069-us-east-1/agent_binaries/$stage/

echo $cross_pfrm
# export DOCKER_BUILDKIT=1
# git clone git://github.com/docker/buildx ./docker-buildx
# docker build --platform=local -o . ./docker-buildx
# mkdir -p ~/.docker/cli-plugins
# mv buildx ~/.docker/cli-plugins/docker-buildx
# docker run --rm --privileged multiarch/qemu-user-static --reset -p yes
# docker login -u $CI_REGISTRY_USER -p $CI_REGISTRY_PASSWORD $CI_REGISTRY
# docker buildx create --use --name mybuilder
# docker buildx build --platform $cross_pfrm --push -t "$CI_REGISTRY/weevenetwork/weeve_agent:1.0.0" -t "$CI_REGISTRY/weevenetwork/weeve_agent:latest" .
