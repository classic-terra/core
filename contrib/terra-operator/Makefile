build-all: build-binary build-node

build-node:
	docker image build \
		-t nghuyenthevinh2000/node \
	 	--build-arg base_image=nghuyenthevinh2000/core \
		--label "nghuyenthevinh2000/node=Contains everything to run a full node" \
		contrib/terra-operator

build-binary:
	docker image build \
		-t nghuyenthevinh2000/core \
		--label "nghuyenthevinh2000/node=Contains everything to run a full node" \
		.

publish-image:
	docker push nghuyenthevinh2000/node:latest
	docker push nghuyenthevinh2000/core:latest

.PHONY: build-binary build-node publish-image