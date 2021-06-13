#!/usr/bin/env sh

set -e


[ -z "$VERSION" ] && echo "VERSION env var is required." && exit 1;
[ -z "$IMAGE" ] && echo "IMAGE env var is required." && exit 1;

DEF_ARCH=amd64
ARCH=${ARCH:-$DEF_ARCH}

IMAGE_TAG_ARCH="${IMAGE}:${VERSION}-${ARCH}"

echo "Pushing image ${IMAGE_TAG_ARCH}..."
docker push ${IMAGE_TAG_ARCH}
