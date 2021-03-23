FROM gcr.io/tetratelabs/tetrate-base:v0.6

COPY ensure_templates /

ENTRYPOINT ["/ensure_templates"]
