APP_NAME    := larasense-limbo
MODULE      := github.com/Mattel-Limbo/larasense-limbo
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT      := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE  := $(shell date -u +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || echo "unknown")

# Detect Windows and append .exe suffix
ifeq ($(OS),Windows_NT)
  BIN_EXT := .exe
else
  BIN_EXT :=
endif
BINARY := $(APP_NAME)$(BIN_EXT)

LDFLAGS := -ldflags "\
	-X $(MODULE)/cmd.Version=$(VERSION) \
	-X $(MODULE)/cmd.Commit=$(COMMIT) \
	-X $(MODULE)/cmd.BuildDate=$(BUILD_DATE)"

.PHONY: build test vet lint clean install help

## build: Build the binary
build:
	go build $(LDFLAGS) -o $(BINARY) .

## test: Run all tests
test:
	go test ./... -v -count=1

## vet: Run go vet
vet:
	go vet ./...

## lint: Run vet + test
lint: vet test

## clean: Remove build artifacts
clean:
	rm -f $(APP_NAME) $(APP_NAME).exe

## install: Install to $GOPATH/bin
install:
	go install $(LDFLAGS) .

## run: Build and run analyze (usage: make run ARGS="--base main")
run: build
	./$(BINARY) analyze $(ARGS)

## version: Build and show version
version: build
	./$(BINARY) version

## help: Show this help
help:
	@echo "Usage: make [target]"
	@echo ""
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'
