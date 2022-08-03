HUB ?= docker.io/tetrate
NAME := zipkin_ensure_es_templates
TAG := 0.1.2
PLATFORMS ?= linux/amd64,linux/arm64

OCI_SOURCE=tetratelabs/zipkin-es-templater
OCI_REVISION=$$(git rev-parse HEAD | cut -c 1-10)

ensure_templates: deps
	CGO_ENABLED=0 go build -o ensure_templates ./cmd/ensure_templates/main.go

deps:
	go mod download

release.dryrun:
	goreleaser release --skip-publish --snapshot --rm-dist

clean:
	rm -f ensure_templates

docker:
	docker build \
	--build-arg OCI_SOURCE=$(OCI_SOURCE) \
	--build-arg OCI_REVISION=$(OCI_REVISION) \
	-t $(HUB)/$(NAME):$(TAG) .

push:
	docker buildx create --use --name $(NAME) --driver docker-container
	docker buildx build \
		--push \
		--build-arg OCI_SOURCE=$(OCI_SOURCE) \
		--build-arg OCI_REVISION=$(OCI_REVISION) \
		--platform $(PLATFORMS) \
		-t $(HUB)/$(NAME):$(TAG) .
	docker buildx rm $(NAME)
