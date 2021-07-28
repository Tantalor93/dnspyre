export GO111MODULE := on

EXECUTABLE = dnstrace
VERSION = development

all: check test build

MAKEFLAGS += --no-print-directory

prepare:
	@echo "Downloading tools"
ifeq (, $(shell which go-junit-report))
	go get github.com/jstemmer/go-junit-report
endif
ifeq (, $(shell which gocov))
	go get github.com/axw/gocov/gocov
endif
ifeq (, $(shell which gocov-xml))
	go get github.com/AlekSi/gocov-xml
endif

check:
ifeq (, $(shell which golangci-lint))
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin v1.30.0
endif
	golangci-lint run
	go mod tidy

test:
	@echo "Running tests"
	mkdir -p report
	go test -race -v ./... -coverprofile=report/coverage.txt | tee report/report.txt
	gocov convert report/coverage.txt | gocov-xml > report/coverage.xml
	go mod tidy

generate:
	@echo "Running generate"
	go generate

build: generate
	@echo "Running build"
	env GOOS=darwin go build -ldflags="-X 'github.com/tantalor93/dnstrace/cmd/dnstrace.Version=$(VERSION)-darwin'" -o bin/$(EXECUTABLE)-darwin
	env GOOS=linux GARCH=amd64 go build -ldflags="-X 'github.com/tantalor93/dnstrace/cmd/dnstrace.Version=$(VERSION)-linux-amd64'" -o bin/$(EXECUTABLE)-linux-amd64
	env GOOS=windows GARCH=amd64 go build -tags -ldflags="-X 'github.com/tantalor93/dnstrace/cmd/dnstrace.Version=$(VERSION)-windows-amd64'" -o bin/$(EXECUTABLE)-windows-amd64

clean:
	rm -rf "bin/"

.PHONY: all check test generate build clean
