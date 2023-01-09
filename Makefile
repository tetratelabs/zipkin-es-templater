NAME := ensure_templates

# The container image name.
IMAGE_NAME := zipkin_ensure_es_templates

# Current version.
VERSION ?= dev

# Destination image registry.
REGISTRY ?= docker.io/tetrate

# Supported container platforms
PLATFORMS ?= linux/amd64 linux/arm64

go     := $(shell which go) # Currently we resolve go using which. A more sophisticated approach is to use infer GOROOT.
goarch := $(shell $(go) env GOARCH)
goexe  := $(shell $(go) env GOEXE)
goos   := $(shell $(go) env GOOS)
ko     := $(go) run github.com/google/ko@v0.12.0

current_binary_path := build/$(NAME)_$(goos)_$(goarch)
current_binary      := $(current_binary_path)/$(NAME)$(goexe)
main_go_sources     := cmd/$(NAME)/main.go

binary_platforms := linux_amd64 linux_arm64 darwin_amd64 darwin_arm64 # currently we don't support Windows.
archives         := $(binary_platforms:%=dist/$(NAME)_$(VERSION)_%.tar.gz)
checksums        := dist/$(NAME)_$(VERSION)_checksums.txt

export PATH := $(go_tools_dir):$(PATH)

build: $(current_binary) ## Build the current binary

dist: $(archives) $(checksums) ## Generate release assets

images: ## Build and push images
	$(call ko-build,$(REGISTRY)/$(IMAGE_NAME))

# Currently, we only do sanity check. This requires a running elasticseach service on port 9200 with disabled security.
sanity: $(current_binary) ## Run sanity check
	@./$^

clean: ## Clean all build and dist artifacts
	@rm -fr build dist

build/$(NAME)_%/$(NAME)$(goexe): $(main_go_sources) $(current_dist)
	$(call go-build,$@,$<)

dist/$(NAME)_$(VERSION)_%.tar.gz: build/$(NAME)_%/$(NAME)
	@mkdir -p $(@D)
	@tar -C $(<D) -cpzf $@ $(<F)

# Darwin doesn't have sha256sum. See https://github.com/actions/virtual-environments/issues/90
sha256sum := $(if $(findstring darwin,$(goos)),shasum -a 256,sha256sum)
$(checksums): $(archives)
	@$(sha256sum) $^ > $@

go_link = -X main.version=$(VERSION)
go-arch = $(if $(findstring amd64,$1),amd64,arm64)
go-os   = $(if $(findstring .exe,$1),windows,$(if $(findstring linux,$1),linux,darwin))
define go-build
	@CGO_ENABLED=0 GOOS=$(call go-os,$1) GOARCH=$(call go-arch,$1) $(go) build \
		-ldflags "-s -w $(go_link)" \
		-o $1 $2
endef

# Run ko build with the specified tag, labels, and platforms.
define ko-build
	@KO_DOCKER_REPO=$1 \
		$(ko) build github.com/tetratelabs/zipkin-es-templater/cmd/$(NAME) \
			--tags $(VERSION) \
			--image-label org.opencontainers.image.title=zipkin-es-templater \
			--image-label org.opencontainers.image.source=github.com/tetratelabs/zipkin-es-templater \
			--image-label org.opencontainers.image.revision=$(shell git rev-parse HEAD | cut -c 1-10) \
			$(addprefix --platform ,$(PLATFORMS)) --bare $2
endef
