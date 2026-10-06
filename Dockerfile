# syntax=docker/dockerfile:1

ARG source=./
ARG GO_VERSION="1.24.7"
ARG BUILDPLATFORM=linux/amd64

# Get Go installation from the official image
FROM --platform=${BUILDPLATFORM} golang:${GO_VERSION} AS go-source

# Use Alpine 3.18 as base for muslc compatibility
FROM --platform=${BUILDPLATFORM} alpine:3.18 AS base

# Copy Go from the official image
COPY --from=go-source /usr/local/go /usr/local/go
# Setup Go environment properly
ENV GOPATH="/go" \
    PATH="/usr/local/go/bin:/go/bin:${PATH}"
# Create necessary directories
RUN mkdir -p "$GOPATH/bin" "$GOPATH/src" && \
    # Verify Go installation
    go version

###############################################################################
# Builder
###############################################################################

FROM base AS builder-stage-1

ARG source
ARG GIT_COMMIT
ARG GIT_VERSION
ARG BUILDPLATFORM
ARG GOOS=linux \
    GOARCH=amd64

ENV GOOS=$GOOS \ 
    GOARCH=$GOARCH

# NOTE: add libusb-dev to run with LEDGER_ENABLED=true
RUN set -eux &&\
    apk update &&\
    apk add --no-cache \
    ca-certificates \
    linux-headers \
    build-base \
    cmake \
    git \
    openssh-client \
    xz

# install mimalloc for musl
WORKDIR ${GOPATH}/src/mimalloc
RUN set -eux &&\
    git clone --depth 1 --branch v2.1.2 \
        https://github.com/microsoft/mimalloc . &&\
    mkdir -p build &&\
    cd build &&\
    cmake .. &&\
    make -j$(nproc) &&\
    make install

# Private modules (e.g. unreleased security forks): pass GOPRIVATE and build with
# `--ssh default`; they are then fetched directly via git+SSH using the host's agent.
# Without GOPRIVATE everything goes through the public module proxy as usual.
ARG GOPRIVATE=""
ENV GOPRIVATE=${GOPRIVATE}
RUN set -eux; \
    if [ -n "${GOPRIVATE}" ]; then \
        mkdir -p -m 0700 /root/.ssh && \
        echo "github.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl" >> /root/.ssh/known_hosts && \
        git config --global url."git@github.com:".insteadOf "https://github.com/"; \
    fi

# download dependencies to cache as layer
WORKDIR ${GOPATH}/src/app
COPY ${source}go.mod ${source}go.sum ./
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=ssh \
    go mod download -x

# Cosmwasm - provide the static libwasmvm for muslc.
# The library goes into the directory of the module actually used (honours `replace`).
# If the module ships it as .xz (covered by go.sum), unpack that; otherwise download
# the matching upstream release asset and verify its checksum.
RUN set -eux &&\
    if [ ${BUILDPLATFORM} = "linux/amd64" ]; then \
        LIB_NAME="libwasmvm_muslc.x86_64.a"; \
    elif [ ${BUILDPLATFORM} = "linux/arm64" ]; then \
        LIB_NAME="libwasmvm_muslc.aarch64.a"; \
    else \
        echo "Unsupported Build Platfrom ${BUILDPLATFORM}"; \
        exit 1; \
    fi; \
    WASMVM_DIR=$(go list -m -f '{{if .Replace}}{{.Replace.Dir}}{{else}}{{.Dir}}{{end}}' github.com/CosmWasm/wasmvm/v3); \
    if [ -f ${WASMVM_DIR}/internal/api/${LIB_NAME}.xz ]; then \
        xz -dc ${WASMVM_DIR}/internal/api/${LIB_NAME}.xz > ${WASMVM_DIR}/internal/api/${LIB_NAME}; \
    else \
        WASMVM_VERSION=$(go list -m -f '{{.Version}}' github.com/CosmWasm/wasmvm/v3); \
        WASMVM_DOWNLOADS="https://github.com/CosmWasm/wasmvm/releases/download/${WASMVM_VERSION}"; \
        wget ${WASMVM_DOWNLOADS}/checksums.txt -O /tmp/checksums.txt; \
        wget ${WASMVM_DOWNLOADS}/${LIB_NAME} -O /tmp/${LIB_NAME}; \
        CHECKSUM=`sha256sum /tmp/${LIB_NAME} | cut -d" " -f1`; \
        grep ${CHECKSUM} /tmp/checksums.txt; \
        rm /tmp/checksums.txt; \
        cp /tmp/${LIB_NAME} ${WASMVM_DIR}/internal/api/; \
        rm /tmp/${LIB_NAME}; \
    fi; \
    ls -la ${WASMVM_DIR}/internal/api/${LIB_NAME}

###############################################################################

FROM builder-stage-1 AS builder-stage-2

ARG source
ARG GOOS=linux \
    GOARCH=amd64

ENV GOOS=$GOOS \ 
    GOARCH=$GOARCH

# Copy the remaining files
COPY ${source} .

# Build app binary
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/root/go/pkg/mod \
    go install \
        -mod=readonly \
        -tags "netgo,muslc" \
        -ldflags " \
            -w -s -linkmode=external -extldflags \
            '-L/go/src/mimalloc/build -lmimalloc -Wl,-z,muldefs -static' \
            -X github.com/cosmos/cosmos-sdk/version.Name='terra' \
            -X github.com/cosmos/cosmos-sdk/version.AppName='terrad' \
            -X github.com/cosmos/cosmos-sdk/version.Version=${GIT_VERSION} \
            -X github.com/cosmos/cosmos-sdk/version.Commit=${GIT_COMMIT} \
            -X github.com/cosmos/cosmos-sdk/version.BuildTags='netgo,muslc' \
        " \
        -trimpath \
        ./...

################################################################################

FROM alpine AS terra-core

RUN apk update && apk add wget lz4 aria2 curl jq gawk coreutils "zlib>1.2.12-r2" libssl3

COPY --from=builder-stage-2 /go/bin/terrad /usr/local/bin/terrad

RUN addgroup -g 1000 terra && \
    adduser -u 1000 -G terra -D -h /app terra

# rest server
EXPOSE 1317
# grpc server
EXPOSE 9090
# tendermint p2p
EXPOSE 26656
# tendermint rpc
EXPOSE 26657

WORKDIR /app

CMD ["terrad", "version"]