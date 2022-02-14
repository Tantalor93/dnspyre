#!/bin/sh

if [ "$#" -ne 1 ]; then
  echo "tag argument missing" >&2
  exit 1
fi
echo "releasing tag $1"
make VERSION="$1" build
ghr "$1" bin/
