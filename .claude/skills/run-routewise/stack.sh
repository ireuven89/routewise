#!/usr/bin/env bash
# Isolated local RouteWise stack for agents: throwaway Postgres (docker) + Go API + CRA dev server.
# Never touches backend/.env, the `routewise-db` compose container, or real Twilio/AWS/Google keys.
#
#   stack.sh up [api]   start db + api (+ web unless "api")
#   stack.sh seed       register a fresh org user, print {token,email,password,...} JSON
#   stack.sh status     show what is up
#   stack.sh logs [api|web]
#   stack.sh down       stop everything and drop the throwaway db
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
STATE="${RW_STATE:-${TMPDIR:-/tmp}/routewise-run}"
DB_CONTAINER=routewise-run-db
DB_PORT="${RW_DB_PORT:-55432}"
API_PORT="${RW_API_PORT:-18080}"
WEB_PORT="${RW_WEB_PORT:-3100}"
mkdir -p "$STATE"

wait_for() { # url timeout_s
  for _ in $(seq 1 "$2"); do curl -sf -o /dev/null "$1" && return 0; sleep 1; done
  echo "timed out waiting for $1" >&2; return 1
}
alive() { [ -f "$STATE/$1.pid" ] && kill -0 "$(cat "$STATE/$1.pid")" 2>/dev/null; }

up_db() {
  docker info >/dev/null 2>&1 || { echo "docker daemon not running (macOS: open -a Docker)" >&2; exit 1; }
  if ! docker ps --format '{{.Names}}' | grep -qx "$DB_CONTAINER"; then
    docker rm -f "$DB_CONTAINER" >/dev/null 2>&1 || true
    # Deliberately NOT mounting backend/migrations into docker-entrypoint-initdb.d (compose.yaml does):
    # the API applies them itself and tracks them in schema_migrations.
    docker run -d --name "$DB_CONTAINER" -p "$DB_PORT:5432" \
      -e POSTGRES_USER=routewise -e POSTGRES_PASSWORD=routewise -e POSTGRES_DB=routewise \
      postgres:15 >/dev/null
  fi
  for _ in $(seq 1 60); do
    docker exec "$DB_CONTAINER" pg_isready -U routewise -d routewise >/dev/null 2>&1 && return 0; sleep 1
  done
  echo "postgres did not become ready" >&2; exit 1
}

up_api() {
  alive api && return 0
  (cd "$ROOT/backend" && go build -o "$STATE/routewise-api" ./cmd/server)
  # main.go calls godotenv.Overload() on ./.env (overrides the shell env) and globs ./migrations,
  # so run from a scratch dir that has its own .env and a symlink to the real migrations.
  mkdir -p "$STATE/api"
  ln -sfn "$ROOT/backend/migrations" "$STATE/api/migrations"
  cat > "$STATE/api/.env" <<EOF
PORT=$API_PORT
DB_HOST=localhost
DB_PORT=$DB_PORT
DB_USER=routewise
DB_PASSWORD=routewise
DB_NAME=routewise
JWT_SECRET=local-run-secret
ALLOWED_ORIGINS=http://localhost:$WEB_PORT
FRONTEND_BASE_URL=http://localhost:$WEB_PORT
APP_ENV=local
AWS_REGION=us-east-1
EOF
  (cd "$STATE/api" && nohup "$STATE/routewise-api" >"$STATE/api.log" 2>&1 & echo $! >"$STATE/api.pid")
  wait_for "http://localhost:$API_PORT/live" 60 || { tail -30 "$STATE/api.log" >&2; exit 1; }
}

up_web() {
  alive web && return 0
  [ -d "$ROOT/frontend/node_modules" ] || (cd "$ROOT/frontend" && npm ci)
  # Shell env beats frontend/.env in CRA, so this points the dev server at our API.
  (cd "$ROOT/frontend" && BROWSER=none PORT="$WEB_PORT" REACT_APP_API_URL="http://localhost:$API_PORT" \
     nohup npm start >"$STATE/web.log" 2>&1 & echo $! >"$STATE/web.pid")
  wait_for "http://localhost:$WEB_PORT" 180 || { tail -30 "$STATE/web.log" >&2; exit 1; }
}

kill_tree() { # pid — npm start spawns children, kill the whole group of descendants
  local p=$1 c
  for c in $(pgrep -P "$p" 2>/dev/null); do kill_tree "$c"; done
  kill "$p" 2>/dev/null || true
}

case "${1:-}" in
  up)
    up_db; up_api
    [ "${2:-}" = api ] || up_web
    echo "db   postgres://routewise:routewise@localhost:$DB_PORT/routewise"
    echo "api  http://localhost:$API_PORT   (log: $STATE/api.log)"
    alive web && echo "web  http://localhost:$WEB_PORT   (log: $STATE/web.log)" || true
    ;;
  seed)
    n=$(date +%s)$RANDOM
    email="agent$n@example.com"; pw="password123"
    resp=$(curl -sf -X POST "http://localhost:$API_PORT/api/v1/register" -H 'Content-Type: application/json' \
      -d "{\"email\":\"$email\",\"password\":\"$pw\",\"name\":\"Agent Tester\",\"phone\":\"+972500000000\",\"company_name\":\"Agent Co $n\",\"industry\":\"plumbing\"}")
    node -e 'const r=JSON.parse(process.argv[1]);console.log(JSON.stringify({email:process.argv[2],password:process.argv[3],token:r.token,user:r.user,organization:r.organization}))' "$resp" "$email" "$pw"
    ;;
  status)
    docker ps --filter "name=$DB_CONTAINER" --format 'db   {{.Status}}'
    alive api && echo "api  up (pid $(cat "$STATE/api.pid"))" || echo "api  down"
    alive web && echo "web  up (pid $(cat "$STATE/web.pid"))" || echo "web  down"
    ;;
  logs) tail -n 50 "$STATE/${2:-api}.log" ;;
  down)
    for s in web api; do alive "$s" && kill_tree "$(cat "$STATE/$s.pid")"; rm -f "$STATE/$s.pid"; done
    docker rm -f "$DB_CONTAINER" >/dev/null 2>&1 || true
    echo down
    ;;
  *) sed -n '2,10p' "$0"; exit 1 ;;
esac
