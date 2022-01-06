FROM golang:latest
# ADD / /app
WORKDIR /

COPY go.mod go.sum ./
# RUN go get -d -v
RUN go mod download

RUN go build -o weeve_agent cmd/agent/agent.go

ENTRYPOINT ["/cmd/agent/agent.go"]
