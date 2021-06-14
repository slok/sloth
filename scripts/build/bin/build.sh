#!/usr/bin/env bash

set -o errexit
set -o nounset

build_script="./scripts/build/bin/build-raw.sh"
ostype=${ostype:-"native"}

echo "[+] Build OS type selected: ${ostype}"

if [ $ostype == 'Linux' ]; then
    EXTENSION="-linux-amd64" GOOS="linux" GOARCH="amd64" ${build_script}
elif [ $ostype == 'Darwin' ]; then
    EXTENSION="-darwin-amd64" GOOS="darwin" GOARCH="amd64" ${build_script}
    EXTENSION="-darwin-arm64" GOOS="darwin" GOARCH="arm64" ${build_script}
elif [ $ostype == 'Windows' ]; then
    EXTENSION="-windows-amd64.exe" GOOS="windows" GOARCH="amd64" ${build_script}
elif [ $ostype == 'ARM' ]; then
    EXTENSION="-linux-arm64" GOOS="linux" GOARCH="arm64" ${build_script}
    EXTENSION="-linux-arm-v7" GOOS="linux" GOARCH="arm" GOARM="7" ${build_script}
else
    # Native.
    ${build_script}
fi
