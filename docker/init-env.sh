#!/bin/sh
# Create or complete .env from .env.example, generating the secrets left empty.
# Values already in .env are kept, so it can run again after an update.
set -eu
cd "$(dirname "$0")"

random() {
	od -An -tx1 -N24 /dev/urandom | tr -d ' \n'
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
