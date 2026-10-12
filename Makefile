ifneq (,$(wildcard .env))
include .env
export
endif

GOOSE := go run github.com/pressly/goose/v3/cmd/goose@v3.28.0

.PHONY: run build test test-integration vet lint fmt check db-up db-down db-reset migration sqlc docker-build

run:
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api

test:
	go test ./...

test-integration: export TEST_DATABASE_URL = postgres://healthcare:healthcare@localhost:5434/healthcare?sslmode=disable
test-integration:
	go test ./... -run Integration -count=1 -v

vet:
	go vet ./...

lint:
	docker run --rm -v "$(CURDIR):/app" -w /app golangci/golangci-lint:v2.14.0 golangci-lint run ./...

fmt:
	gofmt -w .

check: vet test

db-up:
	docker compose up -d --wait

db-down:
	docker compose down

db-reset:
	docker compose down -v
	docker compose up -d --wait

migration:
	$(GOOSE) -dir db/migrations -s create $(name) sql

sqlc:
	docker run --rm -v "$(CURDIR):/src" -w /src sqlc/sqlc:1.31.1 generate

docker-build:
	docker build -t healthcare-api .
