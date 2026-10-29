# Frontend bundle
FROM node:24-trixie-slim AS frontend
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Server binary, cgo is needed by the webp encoder
FROM golang:1.27.1-trixie AS server
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /out/transcendance .

FROM debian:trixie-slim
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates curl tini \
 && rm -rf /var/lib/apt/lists/* \
 && useradd --system --home-dir /opt/transcendance --shell /usr/sbin/nologin transcendance

# The server reads its files relative to the working directory
WORKDIR /opt/transcendance
COPY --from=server /out/transcendance ./
COPY templates ./templates
COPY static ./static
COPY data/assets ./data/assets
COPY data/cards ./data/cards
COPY data/collections ./data/collections
COPY data/tags ./data/tags
COPY --from=frontend /src/frontend/dist ./frontend/dist
# Rendered card images are written at startup
RUN mkdir -p data/dist && chown transcendance: data/dist

USER transcendance
ENV GIN_MODE=release
EXPOSE 8080
HEALTHCHECK --interval=15s --timeout=3s --start-period=30s --retries=3 \
    CMD curl -fsS http://127.0.0.1:8080/healthz || exit 1
ENTRYPOINT ["/usr/bin/tini", "--", "./transcendance"]
