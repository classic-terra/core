#!/usr/bin/make -f

VERSION := $(shell echo $(shell git describe --tags) | sed 's/^v//')
COMMIT := $(shell git log -1 --format='%H')
PACKAGES_E2E=$(shell go list ./... | grep '/e2e')
BUILDDIR ?= $(CURDIR)/build

test-e2e: e2e-setup test-e2e-ci

# test-e2e-ci runs a full e2e test suite
# does not do any validation about the state of the Docker environment
# As a result, avoid using this locally.
test-e2e-ci:
	@VERSION=$(VERSION) TERRA_E2E=True TERRA_E2E_SKIP_UPGRADE=True TERRA_E2E_DEBUG_LOG=False  go test -mod=readonly -timeout=25m -v $(PACKAGES_E2E)

# test-e2e-debug runs a full e2e test suite but does
# not attempt to delete Docker resources at the end.
test-e2e-debug: e2e-setup
	@VERSION=$(VERSION) TERRA_E2E=True TERRA_E2E_SKIP_UPGRADE=True TERRA_E2E_DEBUG_LOG=True TERRA_E2E_SKIP_CLEANUP=True go test -mod=readonly -timeout=25m -v $(PACKAGES_E2E) -count=1

# test-e2e-short runs the e2e test with only short tests.
# Does not delete any of the containers after running.
# Deletes any existing containers before running.
# Does not use Go cache.
test-e2e-short: e2e-setup
	@VERSION=$(VERSION) TERRA_E2E=True TERRA_E2E_SKIP_UPGRADE=True TERRA_E2E_DEBUG_LOG=True TERRA_E2E_SKIP_CLEANUP=True go test -mod=readonly -timeout=25m -v $(PACKAGES_E2E) -count=1

build-e2e-script:
	mkdir -p $(BUILDDIR)
	go build -mod=readonly $(BUILD_FLAGS) -o $(BUILDDIR)/ ./tests/e2e/initialization/$(E2E_SCRIPT_NAME)

docker-build-debug:
	@DOCKER_BUILDKIT=1 docker build -t terra:${COMMIT} --build-arg BASE_IMG_TAG=debug -f ./tests/e2e/e2e.Dockerfile .
	@DOCKER_BUILDKIT=1 docker tag terra:${COMMIT} terra:debug

docker-build-e2e-init-chain:
	@DOCKER_BUILDKIT=1 docker build -t terra-e2e-init-chain:debug --build-arg E2E_SCRIPT_NAME=chain --platform=linux/x86_64 -f tests/e2e/initialization/init.Dockerfile .

docker-build-e2e-init-node:
	@DOCKER_BUILDKIT=1 docker build -t terra-e2e-init-node:debug --build-arg E2E_SCRIPT_NAME=node --platform=linux/x86_64 -f tests/e2e/initialization/init.Dockerfile .

e2e-setup: e2e-check-image-sha e2e-remove-resources
	@echo Finished e2e environment setup, ready to start the test

e2e-check-image-sha:
	tests/e2e/scripts/run/check_image_sha.sh

e2e-remove-resources:
	tests/e2e/scripts/run/remove_stale_resources.sh
