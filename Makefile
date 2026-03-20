.PHONY: dev docker-up docker-down migrate seed bench lint unittest

SOURCE_PKGS := $(shell go list ./... | grep -v '/benchmark$$' | grep -v '/scripts$$' | tr '\n' ',')

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

unittest:
	go test -timeout 9000s -v -coverprofile=coverage.out -coverpkg=$(SOURCE_PKGS) ./... 2>&1 | tee report.out
