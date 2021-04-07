#!/usr/bin/env sh

set -o errexit
set -o nounset

go test -race -coverprofile=.test_coverage.txt ./...
go tool cover -func=.test_coverage.txt | tail -n1 | awk '{print "Total test coverage: " $3}'