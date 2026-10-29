#!/bin/bash
# Kibana objects, applied on every start
set -euo pipefail

curl -sS --fail-with-body -o /dev/null -u "kibana_gateway:${KIBANA_GATEWAY_PASSWORD}" \
	-H 'Content-Type: application/json' -H 'kbn-xsrf: setup' \
	-X POST http://kibana:5601/kibana/api/data_views/data_view -d '{
	"override": true,
	"data_view": {
		"id": "logs-transcendance",
		"name": "Logs",
		"title": "logs-transcendance-*",
		"timeFieldName": "@timestamp"
	}
}'
curl -sS --fail-with-body -o /dev/null -u "kibana_gateway:${KIBANA_GATEWAY_PASSWORD}" \
	-H 'Content-Type: application/json' -H 'kbn-xsrf: setup' \
	-X POST http://kibana:5601/kibana/api/data_views/default -d '{"data_view_id": "logs-transcendance", "force": true}'

# Dashboard of the logs, replaces the one of a previous start
result=$(curl -sS --fail-with-body -u "kibana_gateway:${KIBANA_GATEWAY_PASSWORD}" -H 'kbn-xsrf: setup' \
	-X POST 'http://kibana:5601/kibana/api/saved_objects/_import?overwrite=true' \
	-F file=@/kibana-objects.ndjson)
# Failed objects still answer 200
if ! grep -q '"success":true' <<<"$result"; then
	echo "Importing the Kibana objects failed: $result" >&2
	exit 1
fi

echo "Kibana is set up"
