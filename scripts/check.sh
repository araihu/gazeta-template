#!/bin/sh
set -eu
export GOWORK=off
if [ -n "$(gofmt -l internal)" ]; then
  echo 'Run gofmt on internal/.' >&2
  exit 1
fi
go test -count=1 ./...
go vet ./...
