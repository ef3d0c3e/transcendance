#!/bin/bash
# Users, roles and policies of Elasticsearch, applied on every start
set -euo pipefail

es() {
	local path=$1
	shift
	curl -sS --fail-with-body -o /dev/null -u "elastic:${ELASTIC_PASSWORD}" \
		-H 'Content-Type: application/json' "http://elasticsearch:9200${path}" "$@"
}

echo "Waiting for Elasticsearch"
until es '/_cluster/health?wait_for_status=yellow&timeout=5s' 2>/dev/null; do
	sleep 2
done

# Kibana's own connection
es /_security/user/kibana_system/_password -X POST -d "{\"password\": \"${KIBANA_SYSTEM_PASSWORD}\"}"

# Admins reach Kibana through the gateway as this user
es /_security/role/log_admin -X PUT -d '{
	"cluster": ["monitor", "manage_ilm", "manage_index_templates"],
	"indices": [{"names": ["logs-transcendance-*"], "privileges": ["all"]}]
}'
es /_security/user/kibana_gateway -X PUT -d "{
	\"password\": \"${KIBANA_GATEWAY_PASSWORD}\",
	\"roles\": [\"kibana_admin\", \"log_admin\", \"monitoring_user\"]
}"

echo "Elasticsearch is set up"
