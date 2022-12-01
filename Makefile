export GO111MODULE := on

EXECUTABLE = dnspyre

all: check test build

MAKEFLAGS += --no-print-directory

check:
ifeq (, $(shell which golangci-lint))
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin v1.49.0
endif
ifeq (, $(shell which gosec))
	go install github.com/securego/gosec/v2/cmd/gosec@latest
endif
	golangci-lint run
	gosec -conf=.gosec-conf.json ./...
	go mod tidy

test:
	@echo "Running tests"
	go test -race -v ./...
	go mod tidy

generate:
	@echo "Running generate"
	go generate

build: generate
	@echo "Running build"
	go build -o bin/$(EXECUTABLE)

clean:
	rm -rf "bin/"
	rm -rf "dist/"

.PHONY: all check test generate build clean
