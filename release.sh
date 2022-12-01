#!/bin/sh

if [ "$#" -ne 1 ]; then
  echo "specify only release version, 'sh release.sh <version>'" >&2
  exit 1
fi

if [ -z "${GITHUB_TOKEN}" ]; then
    echo "GITHUB_TOKEN env variable is missing" >&2
    exit 1
fi

if ! command -v goreleaser &> /dev/null
  then echo "goreleaser tool is not installed" >&2
  exit 1
fi

echo "releasing tag $1"
git tag -a "$1" -m "release $1"
git push origin "$1"
goreleaser release --rm-dist
