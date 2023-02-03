all: terrad-env terrad-upgrade-env

terrad-env:
	docker build --build-arg UID=$(shell id -u) --build-arg GID=$(shell id -g) --tag classic-terra/terrad-env terrad-env

terrad-upgrade-env:
	docker build --build-arg UID=$(shell id -u) --build-arg GID=$(shell id -g) --build-arg wrapper_script=cosmovisor-wrapper.sh --tag classic-terra/terrad-upgrade-env terrad-env

.PHONY: all terrad-env terrad-upgrade-env
