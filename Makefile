HUB ?= docker.io/tetrate
NAME := zipkin_ensure_es_templates
TAG := 0.1.2
PLATFORMS := linux/amd64,linux/arm64

OCI_SOURCE=tetratelabs/zipkin-es-templater
OCI_REVISION=$$(git rev-parse HEAD | cut -c 1-10)

build: deps
	CGO_ENABLED=0 go build -o build/ensure_templates ./cmd/ensure_templates/main.go

deps:
	go mod download

release.dryrun:
	goreleaser release --skip-publish --snapshot --rm-dist

clean:
	rm -f ensure_templates

docker:
	docker build \
	--no-cache \
	--build-arg OCI_SOURCE=$(OCI_SOURCE) \
	--build-arg OCI_REVISION=$(OCI_REVISION) \
	-t $(HUB)/$(NAME):$(TAG) .

push: build
	docker buildx create --use --driver docker-container --name $(NAME) > /dev/null 2>&1 || true
	docker buildx build \
		--push --no-cache \
		--build-arg OCI_SOURCE=$(OCI_SOURCE) \
		--build-arg OCI_REVISION=$(OCI_REVISION) \
		--platform $(PLATFORMS) \
		-t $(HUB)/$(NAME):$(TAG) .
	docker buildx rm $(NAME) || true