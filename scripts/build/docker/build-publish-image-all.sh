#!/usr/bin/env bash

set -o errexit
set -o nounset

[ -z "$VERSION" ] && echo "VERSION env var is required." && exit 1;
[ -z "$IMAGE" ] && echo "IMAGE env var is required." && exit 1;

# Build and publish images for all architectures.
archs=("amd64" "arm64" "arm" "ppc64le" "s390x")
for arch in "${archs[@]}"; do
  ARCH="${arch}" ./scripts/build/docker/build-image.sh
  ARCH="${arch}" ./scripts/build/docker/publish-image.sh
done

IMAGE_TAG="${IMAGE}:${VERSION}"

# Create manifest to join all arch images under one virtual tag.
MANIFEST="docker manifest create -a ${IMAGE_TAG}"
for arch in "${archs[@]}"; do
  MANIFEST="${MANIFEST} ${IMAGE_TAG}-${arch}"
done
eval "${MANIFEST}"

# Annotate each arch manifest to set which image is build for which CPU architecture.
for arch in "${archs[@]}"; do
  docker manifest annotate --arch "${arch}" "${IMAGE_TAG}" "${IMAGE_TAG}-${arch}"
done

# Push virual tag metadata.
docker manifest push "${IMAGE_TAG}"

# Same as the regular virtual tag but for `:latest`.
if [ ! -z "${TAG_IMAGE_LATEST:-}" ]; then
    IMAGE_TAG_LATEST="${IMAGE}:latest"

    # Clean latest manifest in case there is one.
    docker manifest rm ${IMAGE_TAG_LATEST} || true

    # Create manifest to join all arch images under one virtual tag.
    MANIFEST_LATEST="docker manifest create -a ${IMAGE_TAG_LATEST}"
    for arch in "${archs[@]}"; do
      MANIFEST_LATEST="${MANIFEST_LATEST} ${IMAGE_TAG}-${arch}"
    done
    eval "${MANIFEST_LATEST}"

    # Annotate each arch manifest to set which image is build for which CPU architecture.
    for arch in "${archs[@]}"; do
      docker manifest annotate --arch "${arch}" "${IMAGE_TAG_LATEST}" "${IMAGE_TAG}-${arch}"
    done

    # Push virual tag metadata.
    docker manifest push "${IMAGE_TAG_LATEST}"
fi