SHELL := /bin/sh
.PHONY: fmt vet test race lint vuln generate api-types check-api-types migrate-up migrate-down docker-up docker-down run-api run-worker run-outbox

fmt:
	gofmt -w .

vet:
	go vet ./...

test:
	go test ./...

race:
	go test -race ./...

lint:
	golangci-lint run

vuln:
	govulncheck ./...

generate:
	CGO_ENABLED=0 go tool oapi-codegen -config api/oapi-codegen.yaml api/openapi/v1/openapi.yaml
	CGO_ENABLED=0 go tool sqlc generate

api-types:
	npm --prefix frontend-api run generate

check-api-types:
	npm --prefix frontend-api run check

migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down

run-api:
	go run ./cmd/api

run-worker:
	go run ./cmd/worker

run-outbox:
	go run ./cmd/outbox list pending 50
