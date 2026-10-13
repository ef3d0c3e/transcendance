#!/usr/bin/env bash
set -euo pipefail

export PATH="/opt/go/bin:${PATH}"
export CGO_ENABLED=1

cd /opt/transcendance/frontend

echo "Installing frontend dependencies..."
npm ci

cd /opt/transcendance

echo "Building frontend..."
make frontend

echo "Starting Go backend..."
exec go run .
