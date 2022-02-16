#!/bin/sh

if [ "$#" -ne 1 ]; then
  echo "tag argument missing" >&2
  exit 1
fi

if ! command -v ghr &> /dev/null
  then echo "ghr tool is not installed" >&2
  exit 1
fi

echo "releasing tag $1"

if [ -z "${GITHUB_TOKEN}" ]; then
  echo "GITHUB_TOKEN variable is not set" >&2
  exit 1
fi

make VERSION="$1" release && ghr "$1" bin/
