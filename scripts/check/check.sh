#!/usr/bin/env sh

set -o errexit
set -o nounset

golangci-lint run -E goimports --timeout 3m