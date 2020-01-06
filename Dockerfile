FROM alpine:3.7

ADD build/bin/linux/amd64/zipkin-es-templater /usr/local/bin/zipkin-es-templater

ENTRYPOINT [ "/usr/local/bin/zipkin-es-templater" ]
