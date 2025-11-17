# syntax=docker/dockerfile:1

ARG source=./
    ARG GO_VERSION="1.24.7"
    ARG BUILDPLATFORM=linux/amd64
    ARG BASE_IMAGE="golang:${GO_VERSION}-alpine3.21"
FROM --platform=${BUILDPLATFORM} ${BASE_IMAGE} AS base

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
    git

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

# download dependencies to cache as layer
WORKDIR ${GOPATH}/src/app
COPY ${source}go.mod ${source}go.sum ./
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/root/go/pkg/mod \
    go mod download -x

# Cosmwasm - Download correct libwasmvm version and verify checksum
RUN set -eux &&\
    WASMVM_VERSION=$(go list -m github.com/CosmWasm/wasmvm/v3 | cut -d ' ' -f 2) && \
    WASMVM_DOWNLOADS="https://github.com/CosmWasm/wasmvm/releases/download/${WASMVM_VERSION}"; \
    wget ${WASMVM_DOWNLOADS}/checksums.txt -O /tmp/checksums.txt; \
    if [ ${BUILDPLATFORM} = "linux/amd64" ]; then \
        WASMVM_URL="${WASMVM_DOWNLOADS}/libwasmvm_muslc.x86_64.a"; \
        LIB_NAME="libwasmvm_muslc.x86_64.a"; \
    elif [ ${BUILDPLATFORM} = "linux/arm64" ]; then \
        WASMVM_URL="${WASMVM_DOWNLOADS}/libwasmvm_muslc.aarch64.a"; \
        LIB_NAME="libwasmvm_muslc.aarch64.a"; \
    else \
        echo "Unsupported Build Platform ${BUILDPLATFORM}"; \
        exit 1; \
    fi; \
    wget ${WASMVM_URL} -O /tmp/${LIB_NAME}; \
    CHECKSUM=`sha256sum /tmp/${LIB_NAME} | cut -d" " -f1`; \
    grep ${CHECKSUM} /tmp/checksums.txt; \
    rm /tmp/checksums.txt

# Place libwasmvm_muslc.a in correct directory structure for wasmvm v2
RUN set -eux &&\
    WASMVM_VERSION=$(go list -m github.com/CosmWasm/wasmvm/v3 | cut -d ' ' -f 2) && \
    if [ ${BUILDPLATFORM} = "linux/amd64" ]; then \
        LIB_NAME="libwasmvm_muslc.x86_64.a"; \
    elif [ ${BUILDPLATFORM} = "linux/arm64" ]; then \
        LIB_NAME="libwasmvm_muslc.aarch64.a"; \
    fi; \
    mkdir -p /go/pkg/mod/github.com/!cosm!wasm/wasmvm/v3@${WASMVM_VERSION}/internal/api/; \
    cp /tmp/${LIB_NAME} /go/pkg/mod/github.com/!cosm!wasm/wasmvm/v3@${WASMVM_VERSION}/internal/api/

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

ENV HOME=/terra
WORKDIR $HOME

# rest server
EXPOSE 1317
# grpc
EXPOSE 9090
# tendermint p2p
EXPOSE 26656
# tendermint rpc
EXPOSE 26657

ENTRYPOINT ["terrad"]