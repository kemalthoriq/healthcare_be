ifneq (,$(wildcard .env))
include .env
export
endif

GOOSE := go run github.com/pressly/goose/v3/cmd/goose@v3.28.0

.PHONY: run build test vet fmt check db-up db-down db-reset migration

run:
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api

test:
	go test ./...

vet:
	go vet ./...

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
