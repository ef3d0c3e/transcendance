#!/bin/sh
# Create or complete .env from .env.example, generating the secrets left empty.
# Values already in .env are kept, so it can run again after an update.
# The Docker socket is asked to Docker, it differs with rootless Docker.
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

# cAdvisor reads the containers from the Docker socket, a value set by hand is kept
current=$(sed -n 's/^DOCKER_SOCKET=//p' .env)
host=$(docker context inspect --format '{{.Endpoints.docker.Host}}' 2>/dev/null || true)
if [ "${host#unix://}" != "$host" ]; then
	if [ "$current" = "/var/run/docker.sock" ] && [ "${host#unix://}" != "$current" ]; then
		set_value DOCKER_SOCKET "${host#unix://}"
	fi
	echo "Docker socket: $(sed -n 's/^DOCKER_SOCKET=//p' .env)"
else
	echo "Cannot ask Docker for its socket, check DOCKER_SOCKET in .env" >&2
fi
