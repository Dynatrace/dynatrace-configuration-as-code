# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM golang:1.21.4-alpine AS build
WORKDIR /src
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=2.x
COPY . .
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=0  go build -a -tags netgo -ldflags "-X github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/version.MonitoringAsCode=${VERSION} -w -extldflags '-static'" ./cmd/monaco


FROM alpine:3.18
RUN apk add --update --no-cache ca-certificates
RUN addgroup monaco ; \
    adduser --shell /bin/false --ingroup monaco --disabled-password --home /monaco monaco
COPY --chown=monaco:monaco --chmod=755 --from=build /src/monaco /usr/local/bin/monaco
USER monaco
WORKDIR /monaco
ENTRYPOINT ["/usr/local/bin/monaco"]
CMD ["--help"]
