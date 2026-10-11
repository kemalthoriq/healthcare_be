# Lumière Aesthetic Clinic — Backend

REST API for a beauty clinic website: public treatment catalog, online booking with real-time slot availability, and an admin panel for bookings, therapists, schedules, patients, invoices, and promos.

Built with the Go standard library (`net/http`), PostgreSQL, and no web framework.

## Features

**Public**
- Treatment, therapist, and promo catalog; clinic profile
- Available dates and 30-minute slots per treatment and therapist
- Booking without an account (name, WhatsApp number, optional email) with a `LMR-XXXX-XXXX` booking code
- Booking status lookup and self-cancellation before a configurable deadline
- Cloudflare Turnstile verification and per-IP rate limiting

**Admin and therapist**
- Session login with role-based access (`admin`, `therapist`)
- Booking workflow: confirm, reschedule, cancel, complete, no-show, with treatment notes
- Treatments, therapists, weekly schedules, time off, therapist accounts
- Patient search and booking history
- Invoices with race-safe monthly numbering (`INV-YYYYMM-NNNN`) and payment recording
- Promos, clinic settings, and a dashboard summary
- Signed image uploads to Cloudinary
- Email notifications through a background SMTP queue

## Design highlights

- **No double booking.** A PostgreSQL exclusion constraint on `tstzrange(start_at, blocked_until)` rejects overlapping active bookings per therapist, so concurrent requests cannot both succeed.
- **Data integrity.** Deactivating a treatment or therapist, changing schedules, or adding time off is rejected with `409 HAS_ACTIVE_BOOKINGS` when it would break an upcoming booking.
- **Optimistic concurrency.** Status transitions update `WHERE status = <previous>` and return `409 STATUS_CHANGED` on conflict.
- **Modular monolith.** Each domain in `internal/` owns its handler, service, and repository; cross-domain access goes through small interfaces wired in `cmd/api`.
- **Secure by default.** argon2id password hashing, hashed session tokens in `HttpOnly` cookies, Origin checks on state-changing requests, and security headers.
- **Clinic time zone.** All scheduling runs in `Asia/Jakarta` with embedded tzdata.

## Tech stack

| Area | Choice |
|---|---|
| Language | Go 1.27 (`net/http` routing) |
| Database | PostgreSQL 17, `pgx/v5` |
| Migrations | goose (embedded, applied on startup with an advisory lock) |
| Queries | sqlc |
| Logging | `log/slog` (JSON in production) |
| CI | GitHub Actions: vet, golangci-lint, `go test -race` with PostgreSQL, `sqlc diff`, Docker build |
| Deployment | Docker (distroless) on Render, Neon PostgreSQL |

## Project structure

```
cmd/api/              entry point, routing, dependency wiring
db/migrations/        SQL migrations (embedded)
db/queries/           SQL queries for sqlc
db/sqlc/              generated code
internal/<domain>/    auth, booking, dashboard, integrity, invoice, notify,
                      promo, setting, therapist, treatment
internal/platform/    config, database, httpx, middleware, logger, email,
                      storage, captcha, ratelimit, clientip, ...
```

## Getting started

Requirements: Go 1.27+, Docker, and `make`.

```bash
cp .env.example .env
make db-up
make run
```

`make db-up` starts PostgreSQL on port `5434` and Mailpit (SMTP `1025`, inbox UI at http://localhost:8025). The server listens on http://localhost:8080 and applies migrations on startup. Set `SEED_ADMIN_EMAIL` and `SEED_ADMIN_PASSWORD` to create the first admin.

| Command | Description |
|---|---|
| `make run` | Run the API |
| `make test` | Unit tests |
| `make test-integration` | Integration tests against the Docker database |
| `make vet` / `make lint` | `go vet` / golangci-lint (Docker) |
| `make sqlc` | Regenerate query code (Docker) |
| `make migration name=...` | Create a migration |
| `make docker-build` | Build the production image |

## Configuration

| Variable | Default | Description |
|---|---|---|
| `APP_ENV` | `development` | `production` enables `Secure` cookies, JSON logs, and required Turnstile |
| `PORT` | `8080` | HTTP port |
| `SHUTDOWN_TIMEOUT` | `10s` | Graceful shutdown limit |
| `DATABASE_URL` | — | PostgreSQL connection string (required) |
| `DB_MAX_CONNS` | `5` | Connection pool size |
| `SEED_ADMIN_NAME` / `SEED_ADMIN_EMAIL` / `SEED_ADMIN_PASSWORD` | `Admin` / — / — | First admin account, created once (password ≥ 12 characters) |
| `FRONTEND_ORIGIN` | `http://localhost:3000` | Allowed `Origin` for state-changing requests; base URL in emails |
| `SESSION_TTL` | `168h` | Session lifetime |
| `TRUSTED_PROXY_HOPS` | `0` | Reverse proxies in front of the API, used to resolve the client IP |
| `TURNSTILE_SECRET_KEY` | — | Cloudflare Turnstile secret (required in production) |
| `EMAIL_ENABLED` | `false` | Send emails; when `false` they are only logged |
| `SMTP_HOST` / `SMTP_PORT` / `SMTP_USER` / `SMTP_PASSWORD` / `SMTP_FROM` | — / `587` / — / — / — | SMTP settings |
| `CLOUDINARY_CLOUD_NAME` / `CLOUDINARY_API_KEY` / `CLOUDINARY_API_SECRET` | — | Image uploads; disabled when unset |

## API overview

Base path `/api/v1`. Responses use `{"data": ...}`, paginated lists add `"meta": {"page", "per_page", "total"}`, and errors use `{"error": {"code", "message", "fields"}}`.

| Area | Endpoints |
|---|---|
| Public catalog | `GET /treatments`, `/treatments/{slug}`, `/treatments/{slug}/therapists`, `/therapists`, `/therapists/{slug}`, `/promos`, `/promos/{slug}`, `/clinic` |
| Availability | `GET /availability/dates`, `GET /availability/slots` |
| Booking | `POST /bookings`, `GET /bookings/{code}`, `POST /bookings/{code}/cancel` |
| Auth | `POST /admin/auth/login`, `POST /admin/auth/logout`, `GET /admin/auth/me` |
| Admin | `/admin/bookings`, `/admin/treatments`, `/admin/therapists` (schedules, time off, accounts), `/admin/patients`, `/admin/invoices`, `/admin/promos`, `/admin/settings`, `/admin/clinic`, `/admin/uploads/sign`, `/admin/dashboard` |

Health checks: `GET /healthz` (liveness, no database) and `GET /readyz` (database ping).
