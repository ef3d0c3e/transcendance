#!/bin/sh
# Adds the optional ALERT_WEBHOOK_URL and ALERT_DISCORD_WEBHOOK_URL receivers. The URLs are secrets: they go
# to files only Alertmanager reads, not to its configuration
set -eu

dir=$(mktemp -d)
config="$dir/alertmanager.yml"
cp /etc/alertmanager/alertmanager.yml "$config"

if [ -n "${ALERT_WEBHOOK_URL:-}" ]; then
	printf '%s' "$ALERT_WEBHOOK_URL" > "$dir/webhook-url"
	cat >> "$config" <<YAML
    webhook_configs:
      - url_file: $dir/webhook-url
        send_resolved: true
YAML
fi
if [ -n "${ALERT_DISCORD_WEBHOOK_URL:-}" ]; then
	printf '%s' "$ALERT_DISCORD_WEBHOOK_URL" > "$dir/discord-url"
	cat >> "$config" <<YAML
    discord_configs:
      - webhook_url_file: $dir/discord-url
        send_resolved: true
YAML
fi
chmod 600 "$dir"/*

exec /bin/alertmanager --config.file="$config" "$@"
