FROM golang:1.20 AS go-builder

# Install minimum necessary dependencies, build Cosmos SDK, remove packages
RUN apt update
RUN apt install -y curl git build-essential
# debug: for live editting in the image
RUN apt install -y vim

WORKDIR /code
COPY . /code/

RUN LEDGER_ENABLED=false make build

RUN cp /go/pkg/mod/github.com/classic-terra/wasmvm@v*/internal/api/libwasmvm.aarch64.so /lib/libwasmvm.aarch64.so

FROM ubuntu:22.04

WORKDIR /root

COPY --from=go-builder /code/build/terrad /usr/local/bin/terrad
COPY --from=go-builder /lib/libwasmvm.aarch64.so /lib/libwasmvm.aarch64.so

# rest server
EXPOSE 1317
# grpc
EXPOSE 9090
# tendermint p2p
EXPOSE 26656
# tendermint rpc
EXPOSE 26657