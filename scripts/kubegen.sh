#!/usr/bin/env sh

set -o errexit
set -o nounset

IMAGE_GEN=ghcr.io/slok/kube-code-generator:v0.6.0
GEN_DIRECTORY="pkg/kubernetes/gen"

echo "Cleaning gen directory"
rm -rf ./${GEN_DIRECTORY}

docker run --rm -it -v ${PWD}:/app "${IMAGE_GEN}" \
	--apis-in ./pkg/kubernetes/api \
	--go-gen-out ./${GEN_DIRECTORY} \
	--crd-gen-out ./${GEN_DIRECTORY}/crd \
	--apply-configurations

echo "Copying crd to helm chart..."
rm ./deploy/kubernetes/helm/sloth/crds/*
cp "${GEN_DIRECTORY}/crd"/* deploy/kubernetes/helm/sloth/crds/
