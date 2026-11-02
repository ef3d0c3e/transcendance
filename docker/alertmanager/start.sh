#!/bin/sh
# Start Alertmanager with the receivers the environment names, both optional:
#   ALERT_WEBHOOK_URL          gets every alert as JSON (Alertmanager's webhook format)
#   ALERT_DISCORD_WEBHOOK_URL  posts them in a Discord channel
# Without them, alerts are only shown in Grafana and Alertmanager. The URLs are secrets, they are written to
# files readable by Alertmanager alone instead of its configuration.
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
