# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

ROOT := cola.io/koffee
OUTPUT_DIR := ./bin

# Module name.
MODULE_NAME := koffee

BUILD_DIR := ./build
# Track code version with Docker Label.
DOCKER_LABELS ?= git-describe="$(shell date +v%Y%m%d)-$(shell git describe --tags --always --dirty)"
GITCOMMIT    ?= $(shell git rev-parse HEAD)
BUILDDATE    ?= $(shell date +"%Y-%m-%dT%H:%M:%SZ")
VERSION      ?= $(shell git describe --tags --always --dirty)
CMD_DIR := ./cmd

export GOFLAGS ?= -count=1

# Golang standard bin directory.
GOPATH ?= $(shell go env GOPATH)
BIN_DIR := $(GOPATH)/bin
GOLANGCI_LINT := $(BIN_DIR)/golangci-lint

.PHONY: all
all: build-local

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: ## Run tests.
	@go test -v -race -gcflags="all=-N -l" -coverpkg=./... -coverprofile=coverage.out ./...
	@go tool cover -func coverage.out | tail -n 1 | awk '{ print "Total coverage: " $$3 }'

.PHONY: lint
lint: $(GOLANGCI_LINT)  ## Lint code
	@$(GOLANGCI_LINT) run

build-local:
	/bin/bash -c 'GOFLAGS="$(GOFLAGS)"                                             \
	  go build -v -o $(OUTPUT_DIR)/$(MODULE_NAME)                                  \
	    -ldflags "-s -w -X $(ROOT)/pkg/version.module=$(MODULE_NAME)               \
		-X $(ROOT)/pkg/version.version=$(VERSION)                                  \
		-X $(ROOT)/pkg/version.gitCommit=$(GITCOMMIT)                              \
		-X $(ROOT)/pkg/version.buildDate=$(BUILDDATE)"                             \
		$(CMD_DIR)'

.PHONY: docker-build
docker-build: ## Build docker image with the manager.
	@docker build -t $(MODULE_NAME):$(VERSION) --label $(DOCKER_LABELS) -f $(BUILD_DIR)/Dockerfile .;

.PHONY: clean
clean:
	rm -rf ./bin coverage.out cover-files

$(GOLANGCI_LINT):
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(BIN_DIR) v2.1.2
