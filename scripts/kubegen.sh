#!/usr/bin/env sh

set -o errexit
set -o nounset

IMAGE_CLI_GEN=quay.io/slok/kube-code-generator:v1.20.1
IMAGE_CRD_GEN=quay.io/slok/kube-code-generator:v1.20.1
ROOT_DIRECTORY=$(dirname "$(readlink -f "$0")")/../
PROJECT_PACKAGE="github.com/slok/sloth"
GEN_DIRECTORY="pkg/kubernetes/gen"

echo "Cleaning gen directory"
rm -rf ./${GEN_DIRECTORY}

echo "Generating Kubernetes CRD clients..."
docker run -it --rm \
	-v ${ROOT_DIRECTORY}:/go/src/${PROJECT_PACKAGE} \
	-e PROJECT_PACKAGE=${PROJECT_PACKAGE} \
	-e CLIENT_GENERATOR_OUT=${PROJECT_PACKAGE}/pkg/kubernetes/gen \
	-e APIS_ROOT=${PROJECT_PACKAGE}/pkg/kubernetes/api \
	-e GROUPS_VERSION="sloth:v1" \
	-e GENERATION_TARGETS="deepcopy,client" \
	${IMAGE_CLI_GEN}

echo "Generating Kubernetes CRD manifests..."
docker run -it --rm \
	-v ${ROOT_DIRECTORY}:/src \
	-e GO_PROJECT_ROOT=/src \
	-e CRD_TYPES_PATH=/src/pkg/kubernetes/api \
	-e CRD_OUT_PATH=/src/pkg/kubernetes/gen/crd \
	${IMAGE_CRD_GEN} update-crd.sh