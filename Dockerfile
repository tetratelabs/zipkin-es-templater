FROM gcr.io/tetratelabs/tetrate-base:v0.1

COPY ensure_templates /

ENTRYPOINT ["/ensure_templates"]
