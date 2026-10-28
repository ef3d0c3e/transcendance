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

# Logstash writes the logs, nothing else
es /_security/role/logstash_writer -X PUT -d '{
	"cluster": ["monitor"],
	"indices": [{"names": ["logs-transcendance-*"], "privileges": ["create_doc", "auto_configure", "view_index_metadata"]}]
}'
es /_security/user/logstash_internal -X PUT -d "{
	\"password\": \"${LOGSTASH_PASSWORD}\",
	\"roles\": [\"logstash_writer\"]
}"

# Logs are kept LOG_RETENTION_DAYS, compressed after LOG_COMPRESS_AFTER_DAYS
es /_ilm/policy/transcendance-logs -X PUT -d "{
	\"policy\": {
		\"phases\": {
			\"hot\": {
				\"actions\": {
					\"rollover\": {\"max_age\": \"1d\", \"max_primary_shard_size\": \"5gb\"}
				}
			},
			\"warm\": {
				\"min_age\": \"${LOG_COMPRESS_AFTER_DAYS}d\",
				\"actions\": {
					\"readonly\": {},
					\"forcemerge\": {\"max_num_segments\": 1, \"index_codec\": \"best_compression\"}
				}
			},
			\"delete\": {
				\"min_age\": \"${LOG_RETENTION_DAYS}d\",
				\"actions\": {\"delete\": {}}
			}
		}
	}
}"
# Wins over the built-in logs template, a single node has no replicas
es /_index_template/logs-transcendance -X PUT -d '{
	"index_patterns": ["logs-transcendance-*"],
	"data_stream": {},
	"priority": 200,
	"composed_of": ["logs@mappings", "logs@settings", "ecs@mappings"],
	"template": {
		"settings": {
			"index.lifecycle.name": "transcendance-logs",
			"index.number_of_replicas": 0
		}
	}
}'
# Indices made before the template, if any
es '/logs-transcendance-*/_settings?allow_no_indices=true' -X PUT -d '{"index.number_of_replicas": 0}'

echo "Elasticsearch is set up"
