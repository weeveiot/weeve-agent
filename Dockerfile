FROM --platform=${BUILDPLATFORM} golang:latest AS build
WORKDIR /
ENV CGO_ENABLED=0
COPY . .
ARG TARGETOS
ARG TARGETARCH


# ADD / /app
# WORKDIR /
# go env -w GO111MODULE=auto
COPY go.mod ./
COPY go.sum ./
COPY cmd/ ./
COPY internal/ ./
# RUN go get -d -v
RUN go mod download

# RUN go build -o weeve_agent cmd/agent/agent.go
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o weeve_agent cmd/agent/agent.go
ENTRYPOINT ["cmd/agent/agent.go"]
