FROM golang:1.23-alpine3.22 AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/tkstar-gateway ./cmd/tkstar-gateway

FROM ghcr.io/home-assistant/base:3.24
ARG BUILD_VERSION=0.1.0
ARG BUILD_ARCH
LABEL io.hass.version="${BUILD_VERSION}" \
      io.hass.type="app" \
      io.hass.arch="${BUILD_ARCH}" \
      org.opencontainers.image.title="TKSTAR Gateway" \
      org.opencontainers.image.description="Lightweight multi-tracker TKSTAR/WINNES tracking server for Home Assistant" \
      org.opencontainers.image.source="https://github.com/ArrowSK/ha-tkstar-gateway" \
      org.opencontainers.image.licenses="MIT"
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /out/tkstar-gateway /usr/local/bin/tkstar-gateway
COPY run.sh /run.sh
RUN chmod 0755 /run.sh /usr/local/bin/tkstar-gateway && /usr/local/bin/tkstar-gateway --self-test
CMD ["/run.sh"]
