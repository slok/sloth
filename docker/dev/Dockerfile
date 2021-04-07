FROM golang:1.16

ARG GOLANGCI_LINT_VERSION="1.37.1"
ARG MOCKERY_VERSION="2.5.1"
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
    rm mockery_${MOCKERY_VERSION}_Linux_x86_64.tar.gz

# Create user.
ARG uid=1000
ARG gid=1000

RUN bash -c 'if [ ${ostype} == Linux ]; then addgroup -gid $gid app; else addgroup app; fi && \
    adduser --disabled-password -uid $uid --ingroup app --gecos "" app && \
    chown app:app -R /go'

# Fill go mod cache.
RUN mkdir /tmp/cache
COPY go.mod /tmp/cache
COPY go.sum /tmp/cache
RUN chown app:app -R /tmp/cache
USER app
RUN cd /tmp/cache && \
    go mod download

WORKDIR /src
