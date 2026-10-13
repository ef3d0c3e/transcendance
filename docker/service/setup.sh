#!/usr/bin/env bash
set -euo pipefail

mkdir -p /etc/nginx/ssl

if  [ ! -d /opt/go ]; then
	echo "Setting up go..."
	cd  /opt
	wget https://go.dev/dl/go1.27.1.linux-amd64.tar.gz
	tar -xvzf go1.27.1.linux-amd64.tar.gz
fi
export PATH="/opt/go/bin:$PATH"

# Install npm deps
npm install vite svelte @sveltejs/vite-plugin-svelte
cd /opt/transcendance/frontend
npm install || true

cd /opt/transcendance
make frontend
exec go run .
