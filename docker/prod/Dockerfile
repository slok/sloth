# Set also `ARCH` ARG here so we can use it on all the `FROM`s. 
ARG ARCH

FROM golang:1.18.3-alpine as build-stage

LABEL org.opencontainers.image.source https://github.com/slok/sloth

RUN apk --no-cache add \
    g++ \
    git \
    make \
    curl \
    bash

# Required by the built script for setting verion and cross-compiling.
ARG VERSION
ENV VERSION=${VERSION}
ARG ARCH
ENV GOARCH=${ARCH}

# Compile.
WORKDIR /src
COPY . .
RUN ./scripts/build/bin/build-raw.sh


# Although we are on an specific architecture (normally linux/amd64) our go binary has been built for
# ${ARCH} specific architecture.
# To make portable our building process we base our final image on that same architecture as the binary 
# to obtain a resulting ${ARCH} image independently where we are building this image.
FROM gcr.io/distroless/static:nonroot-${ARCH}

COPY --from=build-stage /src/bin/sloth /usr/local/bin/sloth

ENTRYPOINT ["/usr/local/bin/sloth"]