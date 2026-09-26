---
name: run-routewise
description: Run, start, launch, screenshot, or drive the RouteWise stack (Go API + React web app) locally with an isolated throwaway Postgres; log in, click through pages, run UI flows, curl the API, and run backend/frontend/integration tests. Use when asked to run routewise, start the backend or frontend, take a screenshot of a page, or verify a change in the real app.
---

# Run RouteWise

RouteWise = Go/Gin API (`backend/`) + CRA React web app (`frontend/`) + Postgres. Agents drive it with two files in this skill:

- `stack.sh` – starts a **throwaway** Postgres container (port 55432), the API (18080) and the web dev server (3100). Isolated from `backend/.env`, the `routewise-db` compose container, and every real Twilio/AWS/Google key.
- `drive.mjs` – headless Chrome via `playwright-core`: screenshots, real UI login, or scripted flows. Prints JSON with headings, console errors and failed API calls.

All paths below are relative to the repo root. Verified on macOS (arm64), Go 1.24, Node 24, Docker Desktop.

## Prerequisites

Docker daemon running (`open -a Docker` on macOS), `go`, `node`/`npm`, Google Chrome installed. One-time driver install:

```bash
npm install --prefix .claude/skills/run-routewise
```

## Run (agent path)

```bash
S=.claude/skills/run-routewise/stack.sh
D=.claude/skills/run-routewise/drive.mjs

$S up            # db + api + web (~10s warm; `$S up api` skips the web)
$S status

node $D login                                   # registers a fresh org user, logs in via the real form, shoots /dashboard
node $D shot /customers --auth                  # any route; --auth = fresh user token injected into localStorage
node $D shot /find-service find-service.png --lang he   # public page, Hebrew/RTL (driver defaults to --lang en)
node $D flow .claude/skills/run-routewise/flows/add-customer.mjs --auth

$S down          # kills api + web, deletes the db container
```

Screenshots go to `$TMPDIR/routewise-run/shots/` (full paths are in the JSON output). **Open them with Read and look.** Check `consoleErrors` / `failedApi` in the output too.

**Custom flows:** copy `flows/add-customer.mjs`. A flow is `export default async ({ page, shot, api, user, web }) => {}`. `page` is a Playwright page, `shot('name.png')` saves a screenshot, and `api('GET', '/api/v1/customers')` calls the API as the `--auth` user. Throw an error to fail the flow.

**API only (curl):**

```bash
$S seed > $TMPDIR/routewise-run/seed.json     # {email,password,token,user,organization}
T=$(node -p 'require(process.env.TMPDIR+"/routewise-run/seed.json").token')
curl -s localhost:18080/health
curl -s localhost:18080/api/v1/me -H "Authorization: Bearer $T"
curl -s -X POST localhost:18080/api/v1/customers -H "Authorization: Bearer $T" -H 'Content-Type: application/json' \
  -d '{"name":"Dana Cohen","phone":"+972501111111","address":"Herzl 1, Tel Aviv"}'
curl -s -X POST localhost:18080/api/v1/public/service-requests -H 'Content-Type: application/json' \
  -d '{"service_type":"plumbing","customer_name":"Noa","customer_phone":"+972502222222","latitude":32.08,"longitude":34.78,"address":"Tel Aviv"}'
```

Routes: `backend/internal/api/routes.go`. Logs: `$S logs api`, `$S logs web`.

## Tests

```bash
(cd backend && go test ./...)
# integration tests (//go:build integration) against the throwaway db, while it is up:
docker exec routewise-run-db psql -U routewise -c 'CREATE DATABASE routewise_test'
(cd backend && TEST_DATABASE_URL='postgres://routewise:routewise@localhost:55432/routewise_test?sslmode=disable' JWT_SECRET=t \
  go test -tags integration -count=1 -p 1 ./internal/integration/...)
(cd frontend && CI=true npm test -- --watchAll=false)
```

## Run (human path)

`docker compose up -d` (the `routewise-db` container on 5432), then `cd backend && go run ./cmd/server` (it reads `backend/.env`, which has **real** Twilio/AWS keys), and `cd frontend && npm start` (port 3000, opens a browser). Agents should not use this path.

## Gotchas

- **`godotenv.Overload()` in `cmd/server/main.go`**: `backend/.env` *overrides* shell env vars, so `DB_PORT=… go run` does nothing. Migrations are also loaded from the relative path `./migrations`. That is why `stack.sh` runs the built binary from `$TMPDIR/routewise-run/api/`, which has its own `.env` and a `migrations` symlink.
- **Migration order**: fresh DBs only migrate cleanly with the numeric sort in `internal/config/database.go` (`sortMigrations`). On a branch without it, `0010_…` sorts before `001_…` and the API dies at startup. `$S logs api` shows the `▶️ Running:` order.
- **compose.yaml mounts `backend/migrations` into `docker-entrypoint-initdb.d`**. Postgres then runs them lexically *and* untracked, before the API's own tracked run. `stack.sh` deliberately does not mount them.
- **CORS is an exact-match allowlist** (`ALLOWED_ORIGINS`). If you change `RW_WEB_PORT`, restart the API too (`$S down && $S up`) so its `.env` is regenerated.
- **CRA env precedence**: the shell's `REACT_APP_API_URL` beats `frontend/.env`, which is how the web is pointed at port 18080. CRA bakes it in at dev-server start, so restart the web after changing it.
- **Default language is Hebrew (RTL)** (`LanguageContext`, stored in `localStorage.language`). `drive.mjs` forces `en` unless you pass `--lang he`, so selectors like `getByRole('button', {name: /Add Customer/})` work.
- **No Google Maps key locally**: address fields in `CustomerModal`/`WorkerModal` fall back to a plain `input[name="address"]` (scriptable). Geocoding silently leaves `latitude/longitude` null. On `/find-service` the address box is a Places autocomplete that stays inert, so the "post job" button stays disabled in the UI. Create service requests via the API (curl above) and screenshot `/find-service/requests/<access_token>`.
- **Twilio creds are blank**: sends fail and are logged, not fatal. A new service request only notifies orgs whose service area covers the point, and freshly registered orgs have none, so usually nothing is sent at all.
- `$S up` can return before webpack's first compile finishes. `drive.mjs` waits for `networkidle`, so it doesn't matter there; with plain curl, give it a few seconds.
- The dashboard greeting shows `Welcome back,` with no name, and the industry chip shows HVAC for a `plumbing` registration. That is current app behavior, not a harness bug.

## Troubleshooting

- `docker daemon not running` → `open -a Docker`, wait for `docker info` to succeed.
- Playwright can't find Chrome → the driver launches the installed Google Chrome (`channel: 'chrome'`). Install Chrome, or set `RW_BROWSER_CHANNEL` to another installed channel. The Linux/Chromium path is untested.
- API exits at startup → `$S logs api`. It is almost always a migration error or a port already in use (`lsof -iTCP:18080 -sTCP:LISTEN`).
