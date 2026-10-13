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
