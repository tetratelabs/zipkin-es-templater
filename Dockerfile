FROM golang:1.16-alpine as builder
ENV BUILD_PATH="github.com/tetratelabs/zipkin-es-templater"
RUN mkdir -p $GOPATH/src/$BUILD_PATH
COPY . $GOPATH/src/$BUILD_PATH
WORKDIR $GOPATH/src/$BUILD_PATH
RUN CGO_ENABLED=0 \
    go build -o ./build/ensure_templates \
    ./cmd/ensure_templates/main.go


FROM gcr.io/tetratelabs/tetrate-base:v0.4
COPY --from=builder go/src/github.com/tetratelabs/zipkin-es-templater/build/ensure_templates /
ARG OCI_SOURCE
ARG OCI_REVISION
LABEL org.opencontainers.image.title zipkin-es-templater
LABEL org.opencontainers.image.source $OCI_SOURCE
LABEL org.opencontainers.image.revision $OCI_REVISION
ENTRYPOINT ["/ensure_templates"]
