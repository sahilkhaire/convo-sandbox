.PHONY: dev up down migrate seed test build

DATABASE_URL ?= postgres://simulator:simulator@localhost:5433/messaging_sim?sslmode=disable

dev:
	docker compose up -d postgres
	@echo "Waiting for postgres..."
	@sleep 3
	$(MAKE) migrate
	go run ./cmd/server

up:
	docker compose up --build -d

down:
	docker compose down

migrate:
	go run github.com/pressly/goose/v3/cmd/goose@latest -dir migrations postgres "$(DATABASE_URL)" up

migrate-down:
	go run github.com/pressly/goose/v3/cmd/goose@latest -dir migrations postgres "$(DATABASE_URL)" down

seed:
	go run ./cmd/seed

test:
	go test ./...

test-integration:
	go test -tags=integration ./internal/integration/...

build:
	go build -o bin/server ./cmd/server

web-dev:
	cd web && npm run dev

web-build:
	cd web && npm run build
