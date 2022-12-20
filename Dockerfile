FROM amd64/alpine:3.16

RUN apk add --update --no-cache \
    curl \
    jq \
    ca-certificates

RUN addgroup monaco && adduser -s /bin/false -G monaco -D monaco

COPY /build/monaco-linux-amd64 /usr/local/bin/monaco
RUN chown -R monaco:monaco /usr/local/bin/monaco
RUN chmod +x /usr/local/bin/monaco

USER monaco

ENTRYPOINT ["/usr/local/bin/monaco"]
CMD ["--help"]
