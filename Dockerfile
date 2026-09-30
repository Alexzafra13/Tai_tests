# syntax=docker/dockerfile:1

# Multi-platform: the frontend and the Go binary are built on the build
# machine (BUILDPLATFORM) and Go cross-compiles for the target, so arm64
# images (e.g. a Raspberry Pi) build fast without emulation.

# 1. Frontend
FROM --platform=$BUILDPLATFORM node:22-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# 2. Go binary (pure Go, no CGO) with the frontend embedded
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/web/dist ./web/dist
ARG TARGETOS TARGETARCH VERSION=dev
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/tai ./cmd/tai

# 3. Runtime: poppler-utils provides pdftotext for the INAP importer
FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata poppler-utils \
    && adduser -D -H -u 10001 tai \
    && mkdir /data && chown tai:tai /data
COPY --from=build /out/tai /usr/local/bin/tai
USER tai
ENV TAI_ADDR=:8080 TAI_DB_PATH=/data/tai.db TZ=Europe/Madrid
VOLUME /data
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s CMD wget -qO- http://127.0.0.1:8080/api/health || exit 1
ENTRYPOINT ["tai"]
CMD ["serve"]
