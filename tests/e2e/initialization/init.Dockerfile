# syntax=docker/dockerfile:1

## Build Image
FROM golang:1.20 as go-builder

ARG BUILDPLATFORM

ARG E2E_SCRIPT_NAME

# Install minimum necessary dependencies, build Cosmos SDK, remove packages
RUN apt update
RUN apt install -y curl git build-essential
# debug: for live editting in the image
RUN apt install -y vim

WORKDIR /terra
COPY . /terra

RUN LINK_STATICALLY=true E2E_SCRIPT_NAME=${E2E_SCRIPT_NAME} make build-e2e-script

RUN if [ ${BUILDPLATFORM} = "linux/amd64" ]; then \
        WASMVM_URL="libwasmvm.x86_64.so"; \
    elif [ ${BUILDPLATFORM} = "linux/arm64" ]; then \
        WASMVM_URL="libwasmvm.aarch64.so"; \     
    else \
        echo "Unsupported Build Platfrom ${BUILDPLATFORM}"; \
        exit 1; \
    fi; \
    cp /go/pkg/mod/github.com/classic-terra/wasmvm@v*/internal/api/${WASMVM_URL} /lib/${WASMVM_URL}


## Deploy image
FROM ubuntu:23.04

# Args only last for a single build stage - renew
ARG E2E_SCRIPT_NAME

COPY --from=go-builder /terra/build/${E2E_SCRIPT_NAME} /bin/${E2E_SCRIPT_NAME}
COPY --from=go-builder /lib/${WASMVM_URL} /lib/${WASMVM_URL}

ENV HOME /terra
WORKDIR $HOME

# Docker ARGs are not expanded in ENTRYPOINT in the exec mode. At the same time,
# it is impossible to add CMD arguments when running a container in the shell mode.
# As a workaround, we create the entrypoint.sh script to bypass these issues.
RUN echo "#!/bin/bash\n${E2E_SCRIPT_NAME} \"\$@\"" >> entrypoint.sh && chmod +x entrypoint.sh

ENTRYPOINT ["./entrypoint.sh"]
