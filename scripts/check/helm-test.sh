#!/usr/bin/env sh

set -o errexit
set -o nounset

cd ./deploy/kubernetes/helm/sloth/tests
go test -race -coverprofile=.test_coverage.txt $(go list ./... | grep -v /test/integration )
go tool cover -func=.test_coverage.txt | tail -n1 | awk '{print "Total test coverage: " $3}'