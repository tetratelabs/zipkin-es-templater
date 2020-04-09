FROM scratch

COPY ensure_templates /

ENTRYPOINT ["/ensure_templates"]
