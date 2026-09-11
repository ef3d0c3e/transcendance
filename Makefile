# Run dev server
.PHONY: dev
dev: frontend
	go run .

.PHONY: frontend
frontend:
	cd frontend && npm run build
