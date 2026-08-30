#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
cd "$ROOT_DIR"

for command_name in docker go bun; do
  if ! command -v "$command_name" >/dev/null 2>&1; then
    printf 'LibreEyes demo requires %s on PATH.\n' "$command_name" >&2
    exit 1
  fi
done

export LIBREEYES_ENV="development"
export LIBREEYES_HTTP_ADDR="${LIBREEYES_HTTP_ADDR:-:8080}"
export LIBREEYES_DATABASE_URL="${LIBREEYES_DATABASE_URL:-postgres://libreeyes:libreeyes_dev@localhost:5432/libreeyes?sslmode=disable}"
export LIBREEYES_CSRF_KEY="${LIBREEYES_CSRF_KEY:-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA}"
export LIBREEYES_DEV_USERNAME="${LIBREEYES_DEV_USERNAME:-clinician}"
export LIBREEYES_DEV_DISPLAY_NAME="${LIBREEYES_DEV_DISPLAY_NAME:-Synthetic Clinician}"
export LIBREEYES_DEV_PASSWORD="${LIBREEYES_DEV_PASSWORD:-123456}"

cleanup() {
  trap - EXIT INT TERM
  if [[ -n "${API_PID:-}" ]]; then kill "$API_PID" 2>/dev/null || true; fi
  if [[ -n "${WEB_PID:-}" ]]; then kill "$WEB_PID" 2>/dev/null || true; fi
}
trap cleanup EXIT INT TERM

printf 'Starting synthetic PostgreSQL...\n'
docker compose up -d postgres
printf 'Waiting for PostgreSQL...\n'
until docker compose exec -T postgres pg_isready -U libreeyes -d libreeyes >/dev/null 2>&1; do
  sleep 1
done

printf 'Applying migrations and seeding the synthetic clinician...\n'
go run ./cmd/migrate up
go run ./cmd/devseed

printf 'Installing frontend dependencies from the frozen lockfile...\n'
bun install --cwd web --frozen-lockfile

printf 'Starting LibreEyes API on http://localhost:8080...\n'
go run ./cmd/api &
API_PID=$!

printf 'Starting LibreEyes web app on http://localhost:5173/login...\n'
bun run --cwd web dev -- --host 127.0.0.1 &
WEB_PID=$!

printf '\nLibreEyes demo is ready.\nUsername: %s\nPassword: %s\nPress Ctrl-C to stop the API and frontend.\n' "$LIBREEYES_DEV_USERNAME" "$LIBREEYES_DEV_PASSWORD"
wait "$WEB_PID"
