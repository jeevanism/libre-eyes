.PHONY: api check db-down db-up demo-reset dev dev-seed fmt migrate-up spec-check test test-race web-build web-check web-dev web-e2e web-install web-test

spec-check:
	./scripts/validate-slices

fmt:
	gofmt -w $$(find cmd internal db -name '*.go' -type f)

test:
	go test ./cmd/... ./internal/... ./db/...

test-race:
	go test -race ./cmd/... ./internal/... ./db/...

db-up:
	docker compose up -d postgres

db-down:
	docker compose down

dev:
	./scripts/dev

demo-reset:
	./scripts/demo-reset

migrate-up:
	go run ./cmd/migrate up

dev-seed:
	go run ./cmd/devseed

api:
	go run ./cmd/api

web-install:
	npm --prefix web install

web-test:
	npm --prefix web test

web-build:
	npm --prefix web run build

web-dev:
	npm --prefix web run dev

web-e2e:
	npm --prefix web run test:e2e

web-check:
	npm --prefix web run lint
	npm --prefix web run typecheck
	npm --prefix web test
	npm --prefix web run build

check: spec-check test web-check
