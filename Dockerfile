FROM amd64/alpine:3.18

ARG NAME=monaco
ARG SOURCE=/build/${NAME}-linux-amd64

RUN apk add --update --no-cache \
    ca-certificates

RUN addgroup monaco ; \
    adduser --shell /bin/false --ingroup monaco --disabled-password --home /monaco monaco

COPY --chown=monaco:monaco --chmod=755 ${SOURCE} /usr/local/bin/monaco

USER monaco
WORKDIR /monaco
ENTRYPOINT ["/usr/local/bin/monaco"]
CMD ["--help"]
