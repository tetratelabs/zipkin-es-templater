HUB ?= docker.io/tetrate
TAG ?= dev

deps:
	go mod download

build: deps
	CGO_ENABLED=0 go build -o ensure_templates ./cmd/ensure_templates/main.go

release.dryrun:
	goreleaser release --skip-publish --snapshot --rm-dist

clean:
	rm -f ensure_templates
