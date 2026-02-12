# RouteWise

Field service management platform supporting HVAC and construction industries. Dispatcher-facing web app + mobile app for workers/technicians.

## Project Structure

```
backend/          Go (Gin) REST API
  cmd/server/     Entrypoint (main.go)
  internal/
    api/          Routes + handlers + middleware
    models/       DB models + request/response types
    repository/   Database layer (raw SQL, PostgreSQL)
    service/      Business logic
    config/       DB init + migrations runner
  migrations/     SQL migration files (run in order on startup)
  config/         YAML config files (local.yaml, prod.yaml)
  pkg/utils/      Shared utilities (viper config loader)
  services/       External services (S3)

frontend/         React SPA (Create React App)
  src/
    api/          Axios API client (client.js)
    components/   Shared components (Layout, Navbar, Skeleton, ServiceCallModal, etc.)
    pages/        Page components (Dashboard, Jobs, Customers, Workers, Login, Register)
    context/      React contexts (AuthContext, LanguageContext)
    translations/ i18n files (en.js, he.js)

routewisemobile/  React Native mobile app
```

## Tech Stack

- **Backend:** Go 1.24, Gin framework, PostgreSQL 15, `lib/pq` driver
- **Frontend:** React 19, React Router 7, Axios, Tailwind CSS, react-icons
- **Database:** PostgreSQL 15 (Docker container `routewise-db`)
- **Auth:** JWT (golang-jwt/v5), OTP for workers (Twilio)
- **Storage:** AWS S3 for project files
- **Monitoring:** Sentry
- **i18n:** English + Hebrew (RTL support)

## Development Setup

### Prerequisites
- Docker (for PostgreSQL)
- Go 1.24+
- Node.js + npm

### Start Database
```bash
docker compose up -d
```
Container `routewise-db` runs on port 5432. Credentials in `backend/.env`.

### Start Backend
```bash
cd backend
go run cmd/server/main.go
```
Runs on port 8080. Loads `.env` via godotenv (shell env vars take precedence over `.env` file). Runs SQL migrations automatically on startup.

### Start Frontend
```bash
cd frontend
npm start
```
Runs on port 3000. API URL configured in `frontend/.env` (`REACT_APP_API_URL=http://localhost:8080`).

## Key Patterns

### Backend
- **Layered architecture:** Handler → Service → Repository
- Handlers in `internal/api/handlers/` parse requests, call services, return JSON
- Services in `internal/service/` contain business logic
- Repositories in `internal/repository/` execute raw SQL queries (no ORM)
- Routes registered in `internal/api/routes.go`, all protected routes use `middleware.AuthMiddleware()`
- Auth middleware sets `organization_id` and `organization_user_id` on gin context
- Multi-tenant: all queries scoped by `organization_id`

### Frontend
- Modal pattern: fixed overlay with `bg-gray-500 bg-opacity-75`, centered white card `max-w-2xl`, form with `px-6 py-4 space-y-4`
- Translations via `useLanguage()` hook — `t('section.key')` with `{{var}}` interpolation
- Industry-aware labels: `t('industry.hvac.workers')` vs `t('industry.construction.workers')`
- API client in `src/api/client.js` — interceptors handle auth token and 401 redirects
- Styling: Tailwind utility classes, orange accent `#ff6b35`, consistent rounded-xl cards with shadow-sm

### Database
- Migrations in `backend/migrations/` — numbered SQL files, tracked in `schema_migrations` table
- Auto-run on startup (skips already-applied)

## API Routes

All under `/api/v1`. Public: `/register`, `/login`, `/workers/request-otp`, `/worker/verify-otp`.
Protected (JWT required): `/me`, `/service_calls`, `/jobs`, `/customers`, `/workers`, `/files`.

## Workflow

**IMPORTANT:** After implementing ANY new feature (handlers, services, repositories, React components), automatically invoke the `unit-test-writer` agent to create comprehensive unit tests. Do NOT wait for the user to request tests — this is a mandatory part of the feature development workflow.

## Common Tasks

- **Add a new API endpoint:** Create handler method → add service interface method → implement in service → add repository method → register route in `routes.go` → **invoke unit-test-writer agent for tests**
- **Add translations:** Update both `en.js` and `he.js` — keep keys in sync
- **Add a migration:** Create numbered SQL file in `backend/migrations/` (e.g., `011_description.sql`)