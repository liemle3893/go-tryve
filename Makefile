VERSION ?= $(shell git describe --tags --always 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build test lint clean

build:
	go build -ldflags "$(LDFLAGS)" -o bin/tryve ./cmd/tryve

test:
	go test ./...

test-v:
	go test -v ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/

run:
	go run ./cmd/tryve $(ARGS)
