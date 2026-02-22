APP_NAME := run-agent
BIN_DIR := bin
GO ?= go
DOCKER_IMAGE ?= conductor-loop:dev

GIT_HASH := $(shell git rev-parse --short HEAD)
BUILD_TIMESTAMP := $(shell date -u +%Y%m%d%H%M%S)
VERSION := v0.54-$(GIT_HASH)-$(BUILD_TIMESTAMP)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

COVERAGE_THRESHOLD ?= 60

.PHONY: build test test-coverage lint docker docs-serve docs-build docs-verify clean

build:
	@mkdir -p $(BIN_DIR)
	$(GO) build $(LDFLAGS) -o $(BIN_DIR)/ ./cmd/...
	@ln -sf $(BIN_DIR)/run-agent run-agent

test:
	$(GO) test ./...

test-coverage:
	$(GO) test -coverprofile=cover.out ./...
	$(GO) tool cover -func=cover.out
	$(GO) tool cover -html=cover.out -o cover.html
	@total=$$($(GO) tool cover -func=cover.out | grep total | awk '{print $$3}' | tr -d '%'); \
	echo "Total coverage: $$total%"; \
	if [ $$(echo "$$total < $(COVERAGE_THRESHOLD)" | bc) -eq 1 ]; then \
	  echo "FAIL: coverage $$total% is below threshold $(COVERAGE_THRESHOLD)%"; \
	  exit 1; \
	fi; \
	echo "Coverage OK ($$total% >= $(COVERAGE_THRESHOLD)%)"

lint:
	$(GO) vet ./...
	@test -z "$$(gofmt -l .)"

docker:
	docker build -t $(DOCKER_IMAGE) .

docs-serve:
	./scripts/docs.sh serve

docs-build:
	./scripts/docs.sh build

docs-verify:
	./scripts/docs.sh verify

clean:
	rm -rf $(BIN_DIR) run-agent
