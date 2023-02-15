all: terrad-env terrad-upgrade-env

terrad-env:
	docker build --build-arg UID=$(shell id -u) --build-arg GID=$(shell id -g) --tag terramoney/terrad-env terrad-env

terrad-upgrade-env:
	docker build --build-arg UID=$(shell id -u) --build-arg GID=$(shell id -g) --build-arg entrypoint=cosmovisor-entrypoint.sh --tag classic-terra/terrad-upgrade-env terrad-env

.PHONY: all terrad-env terrad-upgrade-env
