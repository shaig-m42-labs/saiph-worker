.PHONY: run test build

run:
	go run ./cmd/worker

test:
	go test ./...

build:
	go build ./cmd/worker
