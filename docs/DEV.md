# Setup

## Go setup

Ensure `go` is installed. Version `1.27.1` is the one used by this project.
Get it here: <https://go.dev/dl/>

To install it, extract the archive in some 'standard' location, then add the `bin/` directory in your `$PATH`.

Also `go` should create a directory in your `$HOME`.
Make sure to add `$HOME/go/bin` to your path if you want to install/use go tools.

You should install `golangci-lint`, check out instruction [here](https://golangci-lint.run/docs/welcome/install/local/)
Then you can run lints like so: `golangci-lint run`

## Frontend

You must have a npm-capable package maneger.

NPM Dependencies:
```
npm i vite \
    svelte \
    @sveltejs/vite-plugin-svelte
```

To build frontend components, go to `frontend/` and execute `npm run build` (or `make frontend` from the root)

## Running the server

```
# Start the database container
docker compose up -d

# Run the server
go run .
```

To make the whole process of `Edit -> Stop Server -> Restart server` streamlined, you should install [fresh](https://github.com/gravityblast/fresh): `go install github.com/gravityblast/fresh@latest`.

Then at the project's root start fresh: `fresh`. It will automatically rebuild and restart the server whenever you edit the source code or localization files.

## Makefile

For convenience, a `Makefile` is provided. To build frontend components and run the server, simply execute `make dev`

# Database

To start the database use docker-compose: `docker compose up -d`

To connect to the database, do this:
```
docker exec -it <container-name> mariadb \
         -u root \
         -p
```
Then enter the password for `root` in `.env` and select the `main` database: `USE main;`

Once you have create a user account, you may want to make them an administrator by editing their 'rank' row:
```
update users set rank = 2 where username='admin';
```

# Server Dependencies

 * [gin](https://github.com/gin-gonic/gin) Web server framework
 * [gorm](https://gorm.io/gorm) Database ORM
 * [gorm-mysql](https://gorm.io/driver/mysql) MySQL driver for Gorm
 * [godotenv](https://github.com/joho/godotenv) Sources .env
 * [bcrypt](https://golang.org/x/crypto/bcrypt) bcrypt implementation
 * [gofluent](https://github.com/hakastein/gofluent) Fluent for Go
 * [testify](https://https://github.com/stretchr/testify) Assertion library

# Static routes

 * `/frontend`: Frontend components
 * `/favicon.ico`: Icon
 * `/static`: Static files

# Tests

If you wish to run tests, you must grant `admin` the rights to create and drop databases: 
```
GRANT CREATE, DROP ON main_test.* TO 'admin'@'%'
FLUSH PRIVILEGES
```

**Running tests:**
To run tests: `go test ./PACKAGE  -v` replace `PACKAGE` with the package to run tests for

# Tooling

The following tools are recommended for working with the project:
 * [golangci-lint](https://golangci-lint.run/docs/welcome/install/local/) go linter.
 * [eslint](https://www.npmjs.com/package/eslint) javascript linter. Installation: `npm install @eslint/js eslint-plugin-unicorn eslint-plugin-sonarjs eslint-plugin-import-x`.
 * [eslint-lsp](https://github.com/danielpza/eslint-lsp) language server that integrates ESLint lints as a language server.
 * [svelte-language-server](https://github.com/sveltejs/language-tools) language server for Svelte.
 * [typescript-language-server](https://github.com/typescript-language-server/typescript-language-server) language server for js/ts.
 * [gopls](https://go.dev/gopls/) go language server.
 * [LSP-html](https://github.com/sublimelsp/LSP-html) HTML language server.

# Docker

## Test/Production server

To setup a production or testing server you must first have `nginx` setup with a reverse proxy to `127.0.0.1:8080`.
`nginx` will manage `tls`, you can use [Let's Encrypt](https://letsencrypt.org/) to get a free certificate for your domain.

Then you must create a user, this user will be the one storing files for the project (backend, frontend, database), as well as managing the systemd units.
The user needs to be in the docker group (`usermod -aG docker user`) and have `linger` enabled: `loginctl enable-linger user`.

 1. Installing dependencies

The following dependencies are required:
 - `docker`
 - `systemd`
 - `bash`
 - `git`

 2. Setup

Install the following systemd unit and timer:

`$HOME/.config/systemd/user/live.service`:
```ini
[Unit]
Description=Check for GitHub updates and deploy transcendance
Wants=docker.service
After=docker.service network-online.target
Wants=network-online.target

[Service]
Type=oneshot
ExecStart=/home/transcendance/live.sh
```

`$HOME/.config/systemd/user/live.timer`:
```ini
[Unit]
Description=Check transcendance for updates every 5 minutes

[Timer]
OnBootSec=1min
OnUnitInactiveSec=5min
AccuracySec=15s
Unit=transcendance-deploy.service

[Install]
WantedBy=timers.target
```

Then install the following script in `$HOME/live.sh`:
```bash
#!/usr/bin/env bash
set -Eeuo pipefail

REPO_DIR="/home/transcendance/repo"
BRANCH="testing"
STATE_DIR="/home/transcendance/app-deploy"
STATE_FILE="$STATE_DIR/deployed-sha"
LOCK_FILE="/home/transcendance/app-deploy.lock"

# Prevent overlapping runs
exec 9>"$LOCK_FILE"
flock -n 9 || exit 0

mkdir -p "$STATE_DIR"
cd "$REPO_DIR"

# Fetch the latest branch state from GitHub
git fetch origin "$BRANCH"

REMOTE_SHA="$(git rev-parse "origin/$BRANCH")"
DEPLOYED_SHA=""

if [[ -f "$STATE_FILE" ]]; then
    DEPLOYED_SHA="$(cat "$STATE_FILE")"
fi

# Nothing new to deploy
if [[ "$REMOTE_SHA" == "$DEPLOYED_SHA" ]]; then
    echo "No changes detected."
    exit 0
fi

echo "Deploying $REMOTE_SHA..."

# IMPORTANT: use only in a dedicated deployment checkout
# This discards local tracked changes
git checkout -B "$BRANCH" "origin/$BRANCH"
git reset --hard "$REMOTE_SHA"

# Rebuild and start the updated application
docker compose up -d --build --remove-orphans

# Record success only after deployment succeeds
echo "$REMOTE_SHA" > "$STATE_FILE"

echo "Deployment successful: $REMOTE_SHA"
```

Every 5 minutes, the timer will run `live.sh`.
`live.sh` will check for changes the configured branch and pull the new repo and restart the `docker` images.

 3. Start

Now that you have setup the systemd services, you need to reload them: `systemd --user daemon-reload`.
