#!/bin/sh
# Read-only user of the metrics exporter, applied on every start
set -eu

mariadb -h mariadb -uroot -p"$MARIADB_ROOT_PASSWORD" <<SQL
CREATE USER IF NOT EXISTS 'exporter'@'%' IDENTIFIED BY '$MYSQLD_EXPORTER_PASSWORD' WITH MAX_USER_CONNECTIONS 3;
ALTER USER 'exporter'@'%' IDENTIFIED BY '$MYSQLD_EXPORTER_PASSWORD' WITH MAX_USER_CONNECTIONS 3;
GRANT PROCESS, REPLICATION CLIENT ON *.* TO 'exporter'@'%';
GRANT SELECT ON performance_schema.* TO 'exporter'@'%';
SQL
echo "MariaDB is set up"
