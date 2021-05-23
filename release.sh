#!/bin/bash

env GOOS=darwin go build -ldflags="-X 'main.Version=$1-darwin'" -o dnstrace-darwin
env GOOS=linux GARCH=amd64 go build -ldflags="-X 'main.Version=$1-linux-amd64'" -o dnstrace-linux-amd64