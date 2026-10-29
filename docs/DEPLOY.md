# Deployment

The production stack runs with Docker Compose from the `docker/` directory. For a development setup, see [DEV.md](DEV.md).

It needs about 3 GB of free memory once everything runs. Elasticsearch, Kibana and Logstash take most of it.

## Ports

Only these 3 are published on the host. Everything else, including the database, stays on the internal Docker network.

| Port (default) | Variable        | Protocol | Purpose                                     |
|----------------|-----------------|----------|---------------------------------------------|
| `8080`         | `HTTP_PORT`     | TCP      | Redirects to HTTPS                          |
| `8443`         | `HTTPS_PORT`    | TCP      | The site and the admin tools                |
| `2233`         | `AUTH_SSH_PORT` | TCP      | SSH, used only to approve admin sign-ins    |

Logstash also listens on `127.0.0.1:12201/udp`, only reachable from the host itself: Docker sends the container logs there.

## Configuration


Run `./init-env.sh` from `docker/`. It creates `.env` from `.env.example` and generates every empty password and key; run it again after an update to add new values. The values to review:
| Variable              | Description                                                                   |
|-----------------------|-------------------------------------------------------------------------------|
| `MYSQL_ROOT_PASSWORD` | MariaDB root password.                                                        |
| `MYSQL_USER`          | DB user of the site                                                           |
| `MYSQL_PASSWORD`      | Its password.                                                                 |
| `MYSQL_DATABASE`      | DB name                                                                       |
| `DOMAIN`              | Host name for the TLS certificate                                             |
| `HTTP_PORT`, `HTTPS_PORT`, `AUTH_SSH_PORT` | Published ports                                          |
| `TLS_EXTRA_SANS`      | Other names in the certificate, e.g `IP:192.168.1.10,DNS:transcendance.local` |
| `AUTH_SESSION_TTL`    | How long an admin stays signed in, e.g `12h`                                  |
| `LOG_RETENTION_DAYS`  | Days logs are kept, `730` (2 years) by default                                |
| `LOG_COMPRESS_AFTER_DAYS` | Days before logs are compressed, `14` by default                          |
| `METRICS_RETENTION`   | How long metrics are kept, e.g `30d`                                          |
| `DATA_DIR`            | Folder of the database, logs, metrics and admin keys, `./data` by default     |
| `DOCKER_SOCKET`       | Docker's socket, read for the container metrics. `init-env.sh` asks Docker for it |

The database passwords are only read when the database volume is first created. Changing them later requires changing them inside MariaDB too.

## First start

```sh
docker compose up -d --build
docker compose ps
```

Open `https://<host>:8443/`. The certificate is self-signed, so the browser shows a warning the first time; accept it to continue.

## Admin access

The admin tools (Grafana, Kibana, Prometheus, Alertmanager) are only reachable after signing in with an authorized SSH key. There are no passwords.
Authorize a key. Its comment names the admin, so it is required:

```sh
docker compose exec auth auth add-key "ssh-ed25519 AAAA... foo@bar"
docker compose exec auth auth list-keys
```

To sign in, open `https://<host>:8443/auth/` (or any admin tool). It will display a SSH command to authentify the device you used to access the page.

| Tool         | Path             | Purpose                                                    |
|--------------|------------------|------------------------------------------------------------|
| Grafana      | `/grafana/`      | Firing alerts on its home page, dashboards for the application, infrastructure and logs |
| Kibana       | `/kibana/`       | Search the logs, Logs dashboard (`/kibana/app/dashboards#/view/transcendance-logs`) |
| Prometheus   | `/prometheus/`   | Metrics and alert rules                                    |
| Alertmanager | `/alertmanager/` | Alert details and silences                                 |

To revoke an admin, which also ends their current sessions:
```sh
docker compose exec auth auth remove-key foo@bar
```

## Behind another reverse proxy

When an existing web server already holds the domain, forward it to the gateway's HTTPS port. With nginx:

```nginx
location / {
    proxy_pass https://127.0.0.1:8443;
    proxy_ssl_verify off;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
}
```

The SSH port cannot go through an HTTP proxy, it must be reachable directly.

## Data

Persistent data lives in `DATA_DIR` (`docker/data` by default), one folder per service: `mariadb`, `elasticsearch` (logs), `prometheus` (metrics), `grafana`, `alertmanager` (silences) and `auth` (admin keys and session secrets). The `data-setup` step creates them on start, owned by the user each service runs as, so they can't be read or deleted as your own user. To delete everything:

```sh
docker compose down
# Replace /path/to/data with the absolute path of DATA_DIR
docker run --rm -v /path/to/data:/data nginx:1.29-alpine sh -c 'rm -rf /data/*'
```

The self-signed certificate and the rendered card images stay in the Docker volumes `transcendance-stack_certificates` and `transcendance-stack_card_images`; both are recreated when deleted.

### Storing data on another disk

Set `DATA_DIR` to a folder on that disk. It must use a Linux filesystem such as ext4 or xfs: FAT, exFAT and NTFS can't keep the owners of the folders.

`DATA_DIR` does not hold Docker's images and containers, about 6 GB. They stay in Docker's own storage, `~/.local/share/docker` with rootless Docker. To move it too, with rootless Docker:

```sh
mkdir -p ~/.config/docker
echo '{"data-root": "/media/<disk>/docker"}' > ~/.config/docker/daemon.json
systemctl --user restart docker
```

## Operations

```sh
# Update to the latest master
git pull
docker compose up -d --build --wait

# Follow the logs of a service
docker compose logs -f service

# Stop everything, keeping the data
docker compose down
```

Container logs are rotated by Docker: at most 3 files of 10 MB per container. Docker also sends them to Logstash, which stores them in Elasticsearch; they are compressed after `LOG_COMPRESS_AFTER_DAYS` and deleted after `LOG_RETENTION_DAYS`.

Alerts are not sent anywhere, admins see them on Grafana's home page. The rules are in `docker/prometheus/rules/alerts.yml`. Test them after a change:

```sh
docker run --rm -v ./prometheus/rules:/rules -w /rules --entrypoint promtool prom/prometheus:v3.15.1 test rules alerts_test.yml
```
To create a new certificate, for example after changing `DOMAIN` or `TLS_EXTRA_SANS`:

```sh
docker compose down
docker volume rm transcendance-stack_certificates
docker compose up -d --wait
```
