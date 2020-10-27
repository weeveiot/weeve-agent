.DEFAULT_GOAL := build

.PHONY: build
build:
	cp -R cmd docs internal doc.go go.mod go.sum docker/opt
	(cd docker && ./build.sh)

.PHONY: test
test: build
	(cd docker && docker-compose down || true)
	(cd docker && docker-compose up)
