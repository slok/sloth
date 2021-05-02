#!/usr/bin/env bash

set -o errexit
set -o nounset

version_path="github.com/slok/sloth/internal/info.Version"
src=./cmd/sloth
out=./bin/sloth

ostype=${ostype:-"native"}

function build() {
    ext="${1:-}"
    goos="${2:-}"
    goarch="${3:-}"
    goarm="${4:-}"

    [[ ! -z "${goos}" ]] && export GOOS="${goos}"
    [[ ! -z "${goarch}" ]] && export GOARCH="${goarch}"
    [[ ! -z "${goarm}" ]] && export GOARM="${goarm}"

    final_out=${out}${ext}
    ldf_cmp="-s -w -extldflags '-static'"
    f_ver="-X ${version_path}=${VERSION:-dev}"

    echo "Building binary at ${final_out} (GOOS=${GOOS:-}, GOARCH=${GOARCH:-}, GOARM=${GOARM:-}, VERSION=${VERSION:-})"
    CGO_ENABLED=0 go build -o ${final_out} --ldflags "${ldf_cmp} ${f_ver}"  ${src}
}


if [ $ostype == 'Linux' ]; then
    build "-linux-amd64" "linux" "amd64"
elif [ $ostype == 'Darwin' ]; then
    build "-darwin-amd64" "darwin" "amd64"
    build "-darwin-arm64" "darwin" "arm64"
elif [ $ostype == 'Windows' ]; then
    build "-windows-amd64.exe" "windows" "amd64"
elif [ $ostype == 'ARM' ]; then
    build "-linux-arm64" "linux" "arm64"
    build "-linux-arm-v7" "linux" "arm" "7"
else
    # Native.
    build
fi
