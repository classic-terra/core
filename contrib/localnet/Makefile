all: terrad-env

terrad-env:
	docker build --platform linux/amd64 --build-arg UID=$(shell id -u) --build-arg GID=$(shell id -g) --tag classic-terra/terrad-env terrad-env

.PHONY: all terrad-env
