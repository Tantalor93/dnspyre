#!/bin/sh

if [ "$#" -ne 2 ]; then
  echo "release version and message is required, 'sh release.sh <version> <message>'" >&2
  exit 1
fi

echo "releasing tag $1"
git tag -a $1
git push origin $1
goreleaser release --rm-dist
