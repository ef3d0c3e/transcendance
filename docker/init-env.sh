#!/bin/sh
# Create or complete .env from .env.example, generating the secrets left empty.
# Values already in .env are kept, so it can run again after an update.
# The Docker paths are asked to Docker on every run, they differ with rootless Docker.
set -eu
cd "$(dirname "$0")"

random() {
	od -An -tx1 -N24 /dev/urandom | tr -d ' \n'
}

# Replace or add KEY=VALUE in .env
set_value() {
	grep -v "^$1=" .env >.env.tmp || true
	printf '%s=%s\n' "$1" "$2" >>.env.tmp
	cat .env.tmp >.env
	rm .env.tmp
}

touch .env
chmod 600 .env
added=0
while IFS= read -r line; do
	case "$line" in
	'' | '#'*) continue ;;
	esac
	key=${line%%=*}
	value=${line#*=}
	if grep -q "^$key=" .env; then
		continue
	fi
	# Secrets shared between the services, nobody types them
	case "$key" in
	*_PASSWORD | *_KEY)
		[ -z "$value" ] && value=$(random)
		;;
	esac
	printf '%s=%s\n' "$key" "$value" >>.env
	added=$((added + 1))
done <.env.example
echo "Added $added value(s) to $(pwd)/.env"

# Where the daemon keeps container logs and how to reach it
root=$(docker info --format '{{.DockerRootDir}}' 2>/dev/null || true)
host=$(docker context inspect --format '{{.Endpoints.docker.Host}}' 2>/dev/null || true)
if [ -n "$root" ] && [ "${host#unix://}" != "$host" ]; then
	set_value DOCKER_ROOT "$root"
	set_value DOCKER_SOCKET "${host#unix://}"
	echo "Docker keeps its data in $root, its socket is ${host#unix://}"
else
	echo "Cannot ask Docker for its paths, check DOCKER_ROOT and DOCKER_SOCKET in .env" >&2
fi
