.PHONY: build install test clean docker-build docker-push lint lint-fix docs help

# Binary name
BINARY := packnplay

# Container image
IMAGE := ghcr.io/obra/packnplay/devcontainer:latest

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOINSTALL := $(GOCMD) install
GOTEST := $(GOCMD) test
GOCLEAN := $(GOCMD) clean

# Version information
VERSION := $(shell git describe --tags --always)
COMMIT := $(shell git rev-parse HEAD)
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# Linker flags for version injection
LDFLAGS := -ldflags "-X 'github.com/obra/packnplay/cmd.version=$(VERSION)' -X 'github.com/obra/packnplay/cmd.commit=$(COMMIT)' -X 'github.com/obra/packnplay/cmd.date=$(BUILD_DATE)'"

help: ## Show this help
	@echo "Available targets:"
	@echo "  Build targets:"
	@echo "    build       Build binary with version info"
	@echo "    install     Install to GOPATH/bin with version info"
	@echo "  Quality targets:"
	@echo "    test        Run tests"
	@echo "    lint        Run golangci-lint"
	@echo "  Utility targets:"
	@echo "    clean       Clean build artifacts"
	@echo "    docs        Show documentation links"
	@echo "    help        Show this help"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

build: ## Build the binary
	$(GOBUILD) $(LDFLAGS) -o $(BINARY) .

install: ## Install the binary to GOPATH/bin
	$(GOINSTALL) $(LDFLAGS)

test: ## Run tests
	$(GOTEST) -v ./...

test-coverage: ## Run tests with coverage
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

lint: ## Run golangci-lint
	golangci-lint run

lint-fix: ## Run golangci-lint with auto-fix
	golangci-lint run --fix

docs: ## Show documentation links
	@echo "Build documentation: docs/building.md"
	@echo "Release process: docs/release-process.md"
	@echo "Release engineering: docs/release-engineering.md"

clean: ## Clean build artifacts
	$(GOCLEAN)
	rm -f $(BINARY)
	rm -f coverage.out coverage.html

docker-build: ## Build the default container image
	docker build -t $(IMAGE) .devcontainer/

docker-test: docker-build ## Test the container image
	@echo "Testing container image..."
	docker run --rm $(IMAGE) which node npm claude codex gemini gh
	docker run --rm $(IMAGE) node --version
	docker run --rm $(IMAGE) npm --version
	docker run --rm $(IMAGE) gh --version

docker-push: docker-build ## Push the container image to GHCR
	docker push $(IMAGE)

all: clean build test ## Clean, build, and test

.DEFAULT_GOAL := help
