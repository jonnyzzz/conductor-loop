APP_NAME := conductor
BIN_DIR := bin
GO ?= go
DOCKER_IMAGE ?= conductor-loop:dev

.PHONY: build test lint docker clean

build:
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/ ./cmd/...

test:
	$(GO) test ./...

lint:
	$(GO) vet ./...
	@test -z "$$(gofmt -l .)"

docker:
	docker build -t $(DOCKER_IMAGE) .

clean:
	rm -rf $(BIN_DIR)
