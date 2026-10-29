#!/bin/sh
set -eu
export GOWORK=off
if [ -n "$(gofmt -l internal)" ]; then
  echo 'Execute gofmt em internal/.' >&2
  exit 1
fi
go test ./...
go vet ./...
