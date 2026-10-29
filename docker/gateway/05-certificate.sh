#!/bin/sh
# Self-signed certificate, generated once and kept in the certificate volume
set -eu

dir=/etc/nginx/certs
if [ -s "$dir/gateway.crt" ] && [ -s "$dir/gateway.key" ]; then
	exit 0
fi

names="DNS:${DOMAIN},DNS:localhost,IP:127.0.0.1"
if [ -n "${TLS_EXTRA_SANS:-}" ]; then
	names="$names,$TLS_EXTRA_SANS"
fi

echo "Generating a self-signed certificate for $names"
mkdir -p "$dir"
openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:P-256 -nodes \
	-days 3650 -subj "/CN=${DOMAIN}" -addext "subjectAltName=$names" \
	-keyout "$dir/gateway.key" -out "$dir/gateway.crt"
chmod 600 "$dir/gateway.key"
