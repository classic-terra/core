build-all: build-binary build-node

build-node:
	docker image build \
		-t nghuyenthevinh2000/node \
		--no-cache \
	 	--build-arg base_image=nghuyenthevinh2000/core \
		--label "nghuyenthevinh2000/node=Contains everything to run a full node" \
		contrib/terra-operator
	docker push nghuyenthevinh2000/node:latest

build-binary:
	docker image build \
		-t nghuyenthevinh2000/core \
		--no-cache \
		--label "nghuyenthevinh2000/core=terra classic binary" \
		.
	docker push nghuyenthevinh2000/core:latest

.PHONY: build-binary build-node