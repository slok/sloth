#!/usr/bin/env bash

set -o errexit
set -o nounset

# Build all.
ostypes=("Linux" "Darwin" "Windows" "ARM")
for ostype in "${ostypes[@]}"
do
	ostype="${ostype}" ./scripts/build/bin/build.sh
done

# Create checksums.
checksums_dir="./bin"
cd ${checksums_dir} && sha256sum * > ./checksums.txt
