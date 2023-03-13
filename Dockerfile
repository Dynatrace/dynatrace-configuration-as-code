FROM amd64/alpine:3.17

ARG NAME=monaco
ARG SOURCE=/build/${NAME}-linux-amd64

RUN apk add --update --no-cache \
    curl \
    jq \
    ca-certificates

RUN addgroup monaco ; \
    adduser -s /bin/false -G monaco -D monaco

COPY --chown=monaco:monaco --chmod=755 ${SOURCE} /usr/local/bin/monaco

USER monaco

ENTRYPOINT "/usr/local/bin/monaco"
CMD ["--help"]
