#!/bin/sh
set -eu
private_dir=$(mktemp -d "${TMPDIR:-/tmp}/want-keep-access.XXXXXX")
chmod 700 "$private_dir"
project="want-keep-access-$(date +%s)-$$"
api_pid=""
cleanup() {
  if [ -n "$api_pid" ]; then kill "$api_pid" 2>/dev/null || true; wait "$api_pid" 2>/dev/null || true; fi
  docker rm -fv "$project" >/dev/null 2>&1 || true
  rm -rf "$private_dir"
}
trap cleanup EXIT HUP INT TERM
image='postgres:17.11@sha256:67f41722b7a8cbdb868a44a4995c846eddfdc2973bccb291ce937dce88ad5675'
docker run -d --name "$project" --label want-keep-test=access -e POSTGRES_DB=want_keep_test -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=synthetic-admin -p 127.0.0.1::5432 "$image" > "$private_dir/container-id"
count=0
until docker exec "$project" pg_isready -h 127.0.0.1 -U postgres -d want_keep_test >/dev/null 2>&1; do
  count=$((count + 1)); if [ "$count" -ge 30 ]; then echo 'Access test PostgreSQL unavailable' >&2; exit 1; fi
  sleep 1
done
port=$(docker inspect --format '{{(index (index .NetworkSettings.Ports "5432/tcp") 0).HostPort}}' "$project")
docker exec "$project" psql -U postgres -d want_keep_test -v ON_ERROR_STOP=1 -c "CREATE ROLE want_keep_app LOGIN PASSWORD 'synthetic-app'; CREATE ROLE want_keep_maintenance LOGIN PASSWORD 'synthetic-maintenance';" >/dev/null
export WANT_KEEP_ENV=test
unset WANT_KEEP_ATTACHMENT_KEYRING WANT_KEEP_CONNECTION_KEYRING WANT_KEEP_ATTACHMENT_DIRECTORY WANT_KEEP_DOCUMENT_PROCESSOR_SOCKET WANT_KEEP_TRUSTED_PROXIES
export WANT_KEEP_MAX_MEMBERS=2
export WANT_KEEP_MIGRATION_DATABASE_URL="postgres://postgres:synthetic-admin@127.0.0.1:$port/want_keep_test?sslmode=disable"
export WANT_KEEP_DATABASE_URL="postgres://want_keep_app:synthetic-app@127.0.0.1:$port/want_keep_test?sslmode=disable"
access_port=4183
api_port=8183
if [ "${WANT_KEEP_ACCESS_MANUAL:-0}" = 1 ]; then access_port=4182; api_port=8182; fi
export WANT_KEEP_ORIGIN="http://localhost:$access_port"
export WANT_KEEP_ACCESS_ORIGIN="$WANT_KEEP_ORIGIN"
export WANT_KEEP_LISTEN_ADDR="127.0.0.1:$api_port"
export WANT_KEEP_DEV_API="http://127.0.0.1:$api_port"
export WANT_KEEP_ACCESS_BOOTSTRAP_FILE="$private_dir/bootstrap-token"
export WANT_KEEP_ACCESS_DATABASE_CONTAINER="$project"
export WANT_KEEP_ACCESS_E2E=1
# This launcher owns disposable resources only. Production configuration is never used.
(cd backend && go run ./cmd/migrate && go run ./cmd/identity-bootstrap --output "$WANT_KEEP_ACCESS_BOOTSTRAP_FILE" && go build -o "$private_dir/api" ./cmd/api)
"$private_dir/api" > "$private_dir/api.log" 2>&1 &
api_pid=$!
count=0
until curl --silent --output /dev/null --header "Host: localhost:$access_port" "$WANT_KEEP_DEV_API/api/v1/me"; do
  count=$((count + 1)); if [ "$count" -ge 30 ]; then echo 'Access test API unavailable' >&2; exit 1; fi
  sleep 1
done
if [ "${WANT_KEEP_ACCESS_MANUAL:-0}" = 1 ]; then
  printf 'Private test bootstrap file: %s\n' "$WANT_KEEP_ACCESS_BOOTSTRAP_FILE"
  printf 'Manual Chrome URL: http://localhost:4182/setup\n'
  (cd web && npm run dev -- --host localhost --port 4182 --strictPort)
else
  (cd web && npm exec playwright test -- e2e/access.spec.ts)
fi
