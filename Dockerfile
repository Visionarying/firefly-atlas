# syntax=docker/dockerfile:1

# ---------- frontend ----------
FROM node:22-alpine AS web
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY web/ ./
RUN npm run build

# ---------- backend ----------
FROM golang:1.23-alpine AS service
WORKDIR /app/service
ARG GOPROXY=https://goproxy.cn,direct
ARG TARGETOS=linux
ARG TARGETARCH=amd64
# Single source of truth for the reported version. `docker build --build-arg
# VERSION=x.y.z` (or the compose build args) overrides it, so cutting a release
# no longer requires editing the ldflags line by hand.
ARG VERSION=1.1.0
ENV GOPROXY=${GOPROXY}
COPY service/go.mod service/go.sum ./
RUN go mod download
COPY service/ ./
# The Go binary embeds the SPA, so the Vite output must be in place before compiling.
COPY --from=web /app/web/dist ./router/webdist
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -buildvcs=false \
    -ldflags="-s -w -X firefly-panel/global.Version=${VERSION}" \
    -o /firefly-panel .

# ---------- runtime ----------
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=service /firefly-panel /app/firefly-panel

ENV TZ=Asia/Shanghai \
    DATA_DIR=/app/data \
    PORT=3002 \
    GIN_MODE=release

VOLUME ["/app/data"]
EXPOSE 3002
CMD ["/app/firefly-panel"]
