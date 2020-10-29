# Quick start
(Compiled binary)
main --port 8030

# Dev
## reflex
Using the [reflex](https://github.com/cespare/reflex) file watcher;
(Install with `go get github.com/cespare/reflex`)
`reflex -r '\.go$' -s -- sh -c 'go run ./cmd/node-service.go -v -p 8030'`

Running the server;
`go run ./cmd/node-service.go --port 8030`

make build

## Docker notes
`docker container rm $(docker container ls -aq)   `

