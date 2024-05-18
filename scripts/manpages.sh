#!/bin/sh
# scripts/manpages.sh
set -e
rm -rf manpages
mkdir manpages
go run main.go --help-man >"manpages/dnspyre.1"
gzip --best "manpages/dnspyre.1"
