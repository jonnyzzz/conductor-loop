APP_NAME := conductor
BIN_DIR := bin
GO ?= go
DOCKER_IMAGE ?= conductor-loop:dev

GIT_HASH := $(shell git rev-parse --short HEAD)
BUILD_TIMESTAMP := $(shell date -u +%Y%m%d%H%M%S)
VERSION := v0.54-$(GIT_HASH)-$(BUILD_TIMESTAMP)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build test lint docker clean

build:
	@mkdir -p $(BIN_DIR)
	$(GO) build $(LDFLAGS) -o $(BIN_DIR)/ ./cmd/...
	@ln -sf $(BIN_DIR)/conductor conductor

test:
	$(GO) test ./...

lint:
	$(GO) vet ./...
	@test -z "$$(gofmt -l .)"

docker:
	docker build -t $(DOCKER_IMAGE) .

clean:
	rm -rf $(BIN_DIR) conductor
