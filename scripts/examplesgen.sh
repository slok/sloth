#!/bin/bash
# vim: ai:ts=8:sw=8:noet
set -efCo pipefail
export SHELLOPTS
IFS=$'\t\n'

command -v go >/dev/null 2>&1 || {
    echo 'please install go'
    exit 1
}

SLOS_PATH="${SLOS_PATH:-./examples}"
[ -z "$SLOS_PATH" ] && echo "SLOS_PATH env is needed" && exit 1

GEN_PATH="${GEN_PATH:-./examples/_gen}"
[ -z "$GEN_PATH" ] && echo "GEN_PATH env is needed" && exit 1

mkdir -p "${GEN_PATH}"

# We already know that we are building sloth for each SLO, good enough, this way we can check
# the current development version.
go run ./cmd/sloth/ generate -i "${SLOS_PATH}" -o "${GEN_PATH}" -p "${SLOS_PATH}" --extra-labels "cmd=examplesgen.sh" -e "_gen|windows"
