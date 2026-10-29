#!/bin/sh
# Data folders of the stack, owned by the user each service runs as
set -eu

folder() {
	mkdir -p "/data/$1"
	chown "$2" "/data/$1"
	chmod 700 "/data/$1"
}

folder mariadb 999:999
folder elasticsearch 1000:0
folder prometheus 65534:65534
folder grafana 472:0
folder alertmanager 65534:65534
folder auth 65532:65532
echo "Data folders are ready"
