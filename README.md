# Quick start
(Compiled binary)
main --port 8050


# Dev
Using the [reflex](https://github.com/cespare/reflex) file watcher;
`reflex -r '\.go$' -s -- sh -c 'go run ./cmd/node-service.go -v -p 8030'`

Running the server;
`go run main.go --port 8030`

make build

## Docker notes
`docker container rm $(docker container ls -aq)   `

