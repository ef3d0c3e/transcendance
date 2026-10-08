.PHONY: run
run: frontend
	go run .

.PHONY: frontend
frontend:
	cd frontend && npm run build

.PHONY: dev
dev:
	(fresh&) && cd frontend && npx vite build --watch
