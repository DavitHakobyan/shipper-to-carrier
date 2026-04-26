# shipper-to-carrier

Milestone 1 provides the initial Go API, PostgreSQL migration bootstrap, session auth, and a lightweight dashboard shell for shipper and carrier actors.

## Foundation stack

- **Backend:** Go API
- **Database:** PostgreSQL
- **Frontend:** embedded static dashboard shell served by the API
- **Auth:** email/password login with cookie-backed sessions

## Local development

1. Start PostgreSQL:
   `docker compose up -d`
2. Run the API:
   `go run ./cmd/api`
3. Open:
   `http://localhost:8080`

The server automatically applies SQL migrations on startup.

## Environment

Copy `.env.example` to your preferred environment file or export the variables directly.

## Included milestone 1 endpoints

- `GET /healthz`
- `GET /api/v1/config`
- `POST /api/v1/accounts/register`
- `POST /api/v1/sessions`
- `GET /api/v1/me`
- `POST /api/v1/sessions/logout`
