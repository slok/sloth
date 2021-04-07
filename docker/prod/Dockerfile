FROM golang:1.16-alpine as build-stage

RUN apk --no-cache add \
    g++ \
    git \
    make \
    curl \
    bash

ARG VERSION
ENV VERSION=${VERSION}

# Compile.
WORKDIR /src
COPY . .
RUN ./scripts/build/build.sh


FROM alpine:latest

RUN apk --no-cache add \
    ca-certificates \
    bash

COPY --from=build-stage /src/bin/sloth /usr/local/bin/sloth

ENTRYPOINT ["/usr/local/bin/sloth"]