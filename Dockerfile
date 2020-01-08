FROM alpine:3.7

COPY ensure_templates /

ENTRYPOINT ["/ensure_templates"]
