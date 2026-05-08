BIN ?= bin/avm

.PHONY: build build-ui build-all test fmt vet clean

build:
	go build -o $(BIN) ./cmd/avm

build-ui:
	cd ui && npm ci && npm run typecheck && npm run build

build-all: build build-ui

test:
	go test ./...

fmt:
	gofmt -w ./cmd ./internal

vet:
	go vet ./...

clean:
	rm -rf bin dist ui/dist coverage.out
