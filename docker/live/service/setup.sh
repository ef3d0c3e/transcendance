#!/usr/bin/env bash
set -euo pipefail

mkdir -p /etc/nginx/ssl

export PATH="/opt/go/bin:${PATH}"
export CGO_ENABLED=1

cd /opt/transcendance

# Build frontend and start go backend
make frontend
exec go run .
