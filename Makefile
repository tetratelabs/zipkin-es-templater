ensure_templates: deps
	CGO_ENABLED=0 go build -o ensure_templates ./cmd/ensure_templates/main.go

deps:
	go mod download

release.dryrun:
	goreleaser release --skip-publish --snapshot --rm-dist

clean:
	rm -f ensure_templates
