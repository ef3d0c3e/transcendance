# Deployment

The production stack runs with Docker Compose from the `docker/` directory. For a development setup, see [DEV.md](DEV.md).

## Ports

Only these 3 are published on the host. Everything else, including the database, stays on the internal Docker network.

| Port (default) | Variable        | Protocol | Purpose                                     |
|----------------|-----------------|----------|---------------------------------------------|
| `8080`         | `HTTP_PORT`     | TCP      | Redirects to HTTPS                          |
| `8443`         | `HTTPS_PORT`    | TCP      | The site and the admin tools                |
| `2233`         | `AUTH_SSH_PORT` | TCP      | SSH, used only to approve admin sign-ins    |

## Configuration


Create `.env` from `.env.example`. DB values are required, otherwise the stack won't start:
| Variable              | Description                                                                   |
|-----------------------|-------------------------------------------------------------------------------|
| `MYSQL_ROOT_PASSWORD` | MariaDB root password.                                                        |
| `MYSQL_USER`          | DB user of the site                                                           |
| `MYSQL_PASSWORD`      | Its password.                                                                 |
| `MYSQL_DATABASE`      | DB name                                                                       |
| `DOMAIN`              | Host name for the TLS certificate                                             |
| `HTTP_PORT`, `HTTPS_PORT`, `AUTH_SSH_PORT` | Published ports                                          |
| `TLS_EXTRA_SANS`      | Other names in the certificate, e.g `IP:192.168.1.10,DNS:transcendance.local` |
| `AUTH_SESSION_TTL`    | How long an admin stays signed in, e.g `12h`                                 |

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

Data lives in Docker volumes named `transcendance-stack_<name>`. Running `docker compose down` keeps them; `docker compose down -v` deletes them.

| Volume         | Content                                                      |
|----------------|--------------------------------------------------------------|
| `db_data`      | The DB                                                       |
| `card_images`  | Card images rendered at startup, can be deleted              |
| `certificates` | Self-signed certificate                                      |
| `auth_data`    | Authorized admin keys and session secrets                    |

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

Container logs are rotated by Docker: at most 3 files of 10 MB per container.
To create a new certificate, for example after changing `DOMAIN` or `TLS_EXTRA_SANS`:

```sh
docker compose down
docker volume rm transcendance-stack_certificates
docker compose up -d --wait
```
