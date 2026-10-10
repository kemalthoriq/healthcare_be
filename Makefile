ifneq (,$(wildcard .env))
include .env
export
endif

.PHONY: run build test vet fmt check

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
