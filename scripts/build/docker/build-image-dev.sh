#!/usr/bin/env sh

set -e


[ -z "$VERSION" ] && echo "VERSION env var is required." && exit 1;
[ -z "$IMAGE" ] && echo "IMAGE env var is required." && exit 1;
[ -z "$DOCKER_FILE_PATH" ] && echo "DOCKER_FILE_PATH env var is required." && exit 1;

# Build image.
echo "Building dev image ${IMAGE}:${VERSION}..."
docker build \
    -t "${IMAGE}:${VERSION}" \
    -f "${DOCKER_FILE_PATH}" .