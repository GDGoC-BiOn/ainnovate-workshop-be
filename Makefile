# arahin-mini Makefile
# Simple commands. No Docker required.

# Load values from .env if it exists (ignore if missing).
-include .env
export

# Fallback default (used only when .env / shell do not define DATABASE_URL).
DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/arahin_mini?sslmode=disable

.PHONY: run migrate seed test fmt tidy

# Run the API server (compile + start).
run:
	go run ./cmd/api

# Apply database migrations.
migrate:
	psql "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -f migrations/001_init.sql

# Insert seed data (Alice + the hash-function lesson).
seed:
	psql "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -f seed/seed.sql

# Run the unit test suite.
test:
	go test ./...

# List files that are not gofmt-formatted (no output = all clean).
fmt:
	gofmt -l .

# Add/remove missing module dependencies.
tidy:
	go mod tidy