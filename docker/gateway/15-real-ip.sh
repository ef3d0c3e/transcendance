#!/bin/sh
# Proxies in front of the gateway may name the client, in X-Forwarded-For
set -eu

conf=/etc/nginx/conf.d/real-ip.conf
: > "$conf"
for proxy in $(echo "${TRUSTED_PROXIES:-}" | tr ',' ' '); do
	echo "set_real_ip_from $proxy;" >> "$conf"
done
if [ -s "$conf" ]; then
	echo "real_ip_header X-Forwarded-For;" >> "$conf"
	echo "real_ip_recursive on;" >> "$conf"
fi
