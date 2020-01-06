# Copyright (c) Tetrate, Inc 2020 All Rights Reserved.

NAME := zipkin-es-templater
BUILD_DIR := build/bin
GOOSES := linux darwin windows
GOARCHS := amd64 386

# override to push to a different registry or tag the image differently
HUB ?= gcr.io/tetrate-internal-containers
TAG ?= $(or ${CIRCLE_SHA1},$(shell git rev-parse HEAD))

# Retrieve git versioning details so we can add to our binary assets
VERSION_PATH    := github.com/tetrateio/tetrate/pkg/version
VERSION_STRING  := $(shell git describe --tags --long)
GIT_BRANCH_NAME := $(shell git rev-parse --abbrev-ref HEAD)
GO_LINK_VERSION := -X ${VERSION_PATH}.build=${VERSION_STRING}-${GIT_BRANCH_NAME}

all: build

build:
	@echo "Building binary"
	go build -v -o $(BUILD_DIR)/$(NAME) github.com/tetratelabs/zipkin-es-templater/cmd/ensure_templates/$*
	chmod +x $(BUILD_DIR)/$(NAME)
	@echo "Done building $(NAME)"

vet:
	go vet ./...

test:
	go test ./...

clean:
	rm -rf $(BUILD_DIR)

docker-build: $(BUILD_DIR)/linux/amd64/$(NAME)
	docker build -t $(HUB)/$(NAME):$(TAG) -f Dockerfile .

docker-push:
	docker push $(HUB)/$(NAME):$(TAG)

docker-run:
	docker run $(HUB)/$(NAME):$(TAG)

release:
	@mkdir -p build/bin ; \
	for GOOS in ${GOOSES}; do \
		for GOARCH in ${GOARCHS}; do \
			BINARY_PATH="${BUILD_DIR}/$${GOOS}/$${GOARCH}/$(NAME)"; \
			if [ ! -f "$${BINARY_PATH}" ]; then \
				echo "--- Building binary under $${BINARY_PATH} ---"; \
				CGO_ENABLED=0 GOOS=$${GOOS} GOARCH=$${GOARCH} go build \
					-a --ldflags '${GO_LINK_VERSION} -extldflags "-static"' -tags netgo -installsuffix netgo \
					-o $${BINARY_PATH} github.com/tetratelabs/zipkin-es-templater/cmd/ensure_templates; \
				chmod +x $${BINARY_PATH}; \
				echo "Done building binary under $${BINARY_PATH}"; \
			fi; \
		done; \
	done;

.PHONY: all build vet test clean docker-build docker-push release
