FROM amd64/alpine:3.16

RUN apk add --update --no-cache \
    curl \
    jq \
    ca-certificates

COPY /build/monaco-linux-amd64 /usr/local/bin/monaco
RUN chmod +x /usr/local/bin/monaco

ENTRYPOINT ["/usr/local/bin/monaco"]
CMD ["--help"]
