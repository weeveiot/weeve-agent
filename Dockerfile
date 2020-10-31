#########################################
### BUILDER
#########################################
FROM golang:1.15-alpine3.12 as builder

RUN apk add --no-cache git tree
#\
#  && mkdir -p /opt/node-service

# RUN mkdir /app
# COPY ./cmd ./internal /app/
# COPY go.mod go.sum /app/
COPY . /app/
WORKDIR /app/

# RUN go get -d -v
RUN go build ./cmd/node-service.go

#########################################
### DIST IMAGE
#########################################
FROM alpine

LABEL service="node-service"

# install deps
# RUN apk add --no-cache --no-progress curl tini ca-certificates

# copy node-service binary
COPY --from=builder /app/node-service /usr/bin/node-service

# ENTRYPOINT ["/sbin/tini", "--"]
# CMD ["node-service", "-p", "8030"]
