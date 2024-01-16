FROM golang:1.20 AS go-builder

ARG BUILDPLATFORM

# Install minimum necessary dependencies, build Cosmos SDK, remove packages
RUN apt update
RUN apt install -y curl git build-essential
# debug: for live editting in the image
RUN apt install -y vim

WORKDIR /code
COPY . /code/

RUN LEDGER_ENABLED=false make build

RUN if [ ${BUILDPLATFORM} = "linux/amd64" ]; then \
        WASMVM_URL="libwasmvm.x86_64.so"; \
    elif [ ${BUILDPLATFORM} = "linux/arm64" ]; then \
        WASMVM_URL="libwasmvm.aarch64.so"; \     
    else \
        echo "Unsupported Build Platfrom ${BUILDPLATFORM}"; \
        exit 1; \
    fi; \
    cp /go/pkg/mod/github.com/classic-terra/wasmvm@v*/internal/api/${WASMVM_URL} /lib/${WASMVM_URL}

FROM ubuntu:23.04

COPY --from=go-builder /code/build/terrad /usr/local/bin/terrad
COPY --from=go-builder /lib/${WASMVM_URL} /lib/${WASMVM_URL}

RUN  apt-get update \
  && apt-get install -y wget \
  && rm -rf /var/lib/apt/lists/*
  
ENV HOME /terra
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