FROM amd64/alpine:3.17

ARG NAME=monaco
ARG SOURCE=/build/${NAME}-linux-amd64

RUN apk add --update --no-cache \
    curl \
    jq \
    ca-certificates

RUN addgroup ${NAME} ; \
    adduser -s /bin/false -G ${NAME} -D ${NAME}

COPY --chown=${NAME}:${NAME} --chmod=755 ${SOURCE} /usr/local/bin/${NAME}

USER ${NAME}

ENTRYPOINT ["/usr/local/bin/${NAME}"]
CMD ["--help"]
