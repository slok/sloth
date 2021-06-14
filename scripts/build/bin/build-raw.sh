#!/usr/bin/env bash

set -o errexit
set -o nounset

# Env vars that can be set.
# - EXTENSION: The binary out extension.
# - VERSION: Version for the binary.
# - GOOS: OS compiling target
# - GOARCH: Arch compiling target.
# - GOARM: ARM version.

version_path="github.com/slok/sloth/internal/info.Version"
src=./cmd/sloth
out=./bin/sloth

# Prepare flags.
final_out=${out}${EXTENSION:-}
ldf_cmp="-s -w -extldflags '-static'"
f_ver="-X ${version_path}=${VERSION:-dev}"

# Build binary.
echo "[*] Building binary at ${final_out} (GOOS=${GOOS:-}, GOARCH=${GOARCH:-}, GOARM=${GOARM:-}, VERSION=${VERSION:-}, EXTENSION=${EXTENSION:-})"
CGO_ENABLED=0 go build -o ${final_out} --ldflags "${ldf_cmp} ${f_ver}"  ${src}
