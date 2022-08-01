#! /usr/bin/make -f


# Go related variables.
GOBASE := $(shell pwd)
GOBIN := $(GOBASE)/bin


# Go files.
GOFMT_FILES?=$$(find . -name '*.go' | grep -v vendor)


# Common commands.
all: fmt lint test

test:
	@echo "  >  Running unit tests"
	GOBIN=$(GOBIN) go test -cover -race -coverprofile=coverage.txt -covermode=atomic -v ./...

fmt:
	@echo "  >  Format all go files"
	GOBIN=$(GOBIN) gofmt -w ${GOFMT_FILES}

lint-install:
ifeq (,$(wildcard test -f bin/golangci-lint))
	@echo "  >  Installing golint"
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s
endif

lint: lint-install
	@echo "  >  Running golint"
	bin/golangci-lint run --timeout=2m


# Assets commands.
check:
	go run cmd/main.go check

fix:
	go run cmd/main.go fix

update-auto:
	go run cmd/main.go update-auto

update-manual:
	go run cmd/main.go update-manual


# Helper commands.
add-token:
	go run cmd/main.go add-token $(asset_id)

add-tokenlist:
	go run cmd/main.go add-tokenlist $(asset_id)

add-tokenlist-extended:
	go run cmd/main.go add-tokenlist-extended $(asset_id)
