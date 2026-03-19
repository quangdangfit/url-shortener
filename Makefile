.PHONY: dev docker-up docker-down migrate seed bench lint

dev:
	go run ./cmd/api

docker-up:
	docker compose up -d

docker-down:
	docker compose down

migrate:
	go run ./cmd/migrate

seed:
	go run ./scripts/seed.go

bench:
	go test ./benchmark/... -bench=. -benchtime=10s

lint:
	golangci-lint run
