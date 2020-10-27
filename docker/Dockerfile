#########################################
### BUILDER
#########################################
FROM golang:1.15-alpine3.12 as builder

RUN apk add --no-cache git tree \
 && mkdir -p /opt/node-service

COPY opt/ /opt/node-service/

RUN cd /opt/node-service \
 && go build cmd/node-service.go


#########################################
### DIST IMAGE
#########################################
FROM alpine

MAINTAINER Kanchen Monnin

LABEL multi.author="Kanchen Monnin" \
      multi.maintainer="kanchen.monnin@weeve.network" \
      multi.department="DevOps" \
      multi.service="node-service" \
      multi.description="node-service on edge"

LABEL service="node-service"

# install deps
RUN apk add --no-cache --no-progress curl tini ca-certificates

# copy node-service binary
COPY --from=builder /opt/node-service/node-service /usr/bin/node-service

ENTRYPOINT ["/sbin/tini", "--"]
CMD ["node-service", "-p", "8030"]
