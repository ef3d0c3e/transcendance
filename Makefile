COMPOSE := docker compose -f docker/docker-compose.yml

# Deployment: `make` creates docker/.env when needed, then builds and starts the whole stack
.PHONY: all
all: up

.PHONY: up
up:
	./docker/init-env.sh
	$(COMPOSE) up -d --build --wait

.PHONY: down
down:
	$(COMPOSE) down

.PHONY: logs
logs:
	$(COMPOSE) logs -f

# Development: the database in Docker, the server on the host
.PHONY: db
db:
	docker compose -f docker-compose.dev.yml up -d

.PHONY: run
run: frontend
	go run .

.PHONY: frontend
frontend:
	cd frontend && npm run build

.PHONY: dev
dev:
	@set -m; \
	fresh & \
	fresh_pid=$$!; \
	trap 'kill -TERM -$$fresh_pid 2>/dev/null; sleep 0.3; kill -KILL -$$fresh_pid 2>/dev/null; exit 0' INT TERM; \
	cd frontend && npx vite build --watch
