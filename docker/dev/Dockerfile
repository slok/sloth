FROM golang:1.24

LABEL org.opencontainers.image.source=https://github.com/slok/sloth

ARG GOLANGCI_LINT_VERSION="1.64.8"
ARG MOCKERY_VERSION="3.1.0"
ARG GOMARKDOC_VERSION="1.1.0"
ARG HELM_VERSION="3.17.0"
ARG YAEGI_VERSION="0.16.1"
ARG ostype=Linux

RUN apt-get update && apt-get install -y \
    git \
    bash \
    zip


RUN wget https://github.com/golangci/golangci-lint/releases/download/v${GOLANGCI_LINT_VERSION}/golangci-lint-${GOLANGCI_LINT_VERSION}-linux-amd64.tar.gz && \
    tar zxvf golangci-lint-${GOLANGCI_LINT_VERSION}-linux-amd64.tar.gz --strip 1 -C /usr/local/bin/ && \
    rm golangci-lint-${GOLANGCI_LINT_VERSION}-linux-amd64.tar.gz && \
    \
    wget https://github.com/vektra/mockery/releases/download/v${MOCKERY_VERSION}/mockery_${MOCKERY_VERSION}_Linux_x86_64.tar.gz && \
    tar zxvf mockery_${MOCKERY_VERSION}_Linux_x86_64.tar.gz -C /tmp && \
    mv /tmp/mockery /usr/local/bin/ && \
    rm mockery_${MOCKERY_VERSION}_Linux_x86_64.tar.gz && \
    \
    wget https://github.com/princjef/gomarkdoc/releases/download/v${GOMARKDOC_VERSION}/gomarkdoc_${GOMARKDOC_VERSION}_linux_amd64.tar.gz && \
    tar zxvf gomarkdoc_${GOMARKDOC_VERSION}_linux_amd64.tar.gz -C /tmp && \
    mv /tmp/gomarkdoc_${GOMARKDOC_VERSION}_linux_amd64/gomarkdoc /usr/local/bin/ && \
    rm -rf gomarkdoc_${GOMARKDOC_VERSION}_linux_amd64.tar.gz /tmp/gomarkdoc_${GOMARKDOC_VERSION}_linux_amd64 && \
    \
    wget https://get.helm.sh/helm-v${HELM_VERSION}-linux-amd64.tar.gz && \
    tar zxvf helm-v${HELM_VERSION}-linux-amd64.tar.gz -C /tmp && \
    mv /tmp/linux-amd64/helm /usr/local/bin/ && \
    rm -rf helm-v${HELM_VERSION}-linux-amd64.tar.gz /tmp/linux-amd64 && \
    \
    wget https://github.com/traefik/yaegi/releases/download/v${YAEGI_VERSION}/yaegi_v${YAEGI_VERSION}_linux_amd64.tar.gz && \
    tar zxvf yaegi_v${YAEGI_VERSION}_linux_amd64.tar.gz -C /tmp && \
    mv /tmp/yaegi /usr/local/bin/ && \
    rm yaegi_v${YAEGI_VERSION}_linux_amd64.tar.gz


# Create user.
ARG uid=1000
ARG gid=1000

RUN bash -c 'if [ ${ostype} == Linux ]; then addgroup -gid $gid app; else addgroup app; fi && \
    adduser --disabled-password -uid $uid --ingroup app --gecos "" app && \
    chown app:app -R /go'

# Fill Go apps cache:
# Main app.
RUN mkdir -p /tmp/cache
COPY go.mod /tmp/cache
COPY go.sum /tmp/cache
RUN chown app:app -R /tmp/cache
USER app
RUN cd /tmp/cache && \
    go mod download

# Helm testing app.
USER root
RUN mkdir -p /tmp/cache
COPY deploy/kubernetes/helm/sloth/tests/go.mod /tmp/cache
COPY deploy/kubernetes/helm/sloth/tests/go.sum /tmp/cache
RUN chown app:app -R /tmp/cache
USER app
RUN cd /tmp/cache && \
    go mod download

USER app
WORKDIR /src
