# Deployment Guide

Deploy the **dashboard** (React UI + `/api` proxy) and **vendor API** (Go server) on **separate domains**. The dashboard and its admin API share one hostname so nginx can route `/api/*` to the backend and everything else to the SPA.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  simulator.yourdomain.com  (dashboard domain)               │
│  nginx: static SPA + proxy /api/* → Go backend              │
├─────────────────────────────────────────────────────────────┤
│  GET  /              → React app                            │
│  GET  /login         → React app                            │
│  GET  /api/health    → Go admin API                         │
│  POST /api/auth/login→ Go admin API                         │
│  GET  /api/events    → Go SSE (proxied, buffering off)      │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│  vendor-sim.yourdomain.com  (vendor domain — separate)      │
│  Go server directly (production-identical paths, no /api)   │
├─────────────────────────────────────────────────────────────┤
│  GET  /v1/oauth/token                                       │
│  POST /v1/messaging/users/{phone}/assistantMessages/async   │
│  POST /v19.0/{phone-number-id}/messages                     │
└─────────────────────────────────────────────────────────────┘
```

| Route prefix | Domain | Purpose |
|---|---|---|
| `/api/*` | Dashboard | Admin UI API (auth, accounts, inbox, webhooks) |
| `/`, `/login`, … | Dashboard | React SPA |
| `/v1/*`, `/v19.0/*`, … | Vendor | Jio RCS + Meta WhatsApp simulator (unchanged) |

The frontend is built with `VITE_API_URL=/api` so all dashboard requests are same-origin relative paths — nginx identifies and proxies them.

---

## Prerequisites

1. **CapRover** (or any Docker host) with two apps:
   - `messaging-api` — Go backend (`Dockerfile`)
   - `messaging-dashboard` — nginx + SPA (`web/Dockerfile`)
2. **PostgreSQL** reachable from the API app
3. GitHub repo with Actions enabled

---

## GitHub Secrets

| Secret | Description |
|---|---|
| `CAPROVER_SERVER` | CapRover dashboard URL |
| `CAPROVER_APP_TOKEN` | API app deploy token |
| `CAPROVER_DASHBOARD_TOKEN` | Dashboard app deploy token (can reuse same token if same CapRover) |

---

## GitHub Variables

| Variable | Description | Example |
|---|---|---|
| `CAPROVER_APP_NAME` | Go API app name | `messaging-api` |
| `CAPROVER_DASHBOARD_APP_NAME` | nginx dashboard app name | `messaging-dashboard` |
| `VITE_API_URL` | Frontend API base (same-domain) | `/api` |

> **Do not** set `VITE_API_URL` to a full cross-origin URL when frontend and API share a domain. Use `/api`.

---

## CapRover: API app (`messaging-api`)

**Dockerfile:** `./Dockerfile` (root)

**Environment variables:**

| Variable | Required | Example |
|---|---|---|
| `DATABASE_URL` | Yes | `postgres://user:pass@srv-captain--postgres:5432/messaging_sim?sslmode=disable` |
| `JWT_SECRET` | Yes (prod) | `openssl rand -base64 48` |
| `JWT_EXPIRY_HOURS` | No | `24` |
| `CORS_ORIGIN` | Yes | `https://simulator.yourdomain.com` |
| `MEDIA_STORAGE_PATH` | No | `/data/media` |
| `ENABLE_DATA_RESET` | No | `true` |
| `DEFAULT_WEBHOOK_URL` | No | `https://your-platform/webhooks` |

**Custom domain:** `vendor-sim.yourdomain.com` (vendor testing only)

**Persistent volume:** `/data/media`

Container HTTP port: **8080**

---

## CapRover: Dashboard app (`messaging-dashboard`)

**Dockerfile:** `./web/Dockerfile`

**Environment variables:**

| Variable | Required | Description |
|---|---|---|
| `API_UPSTREAM` | Yes | Internal URL of the Go API app |

Example `API_UPSTREAM` values on CapRover:

```
http://srv-captain--messaging-api:80
```

Use the internal service hostname CapRover assigns to your API app (check **App Configs → Service Discovery**).

**Custom domain:** `simulator.yourdomain.com`

The nginx config in `web/nginx.conf.template` proxies `/api/` to `${API_UPSTREAM}/api/` and serves the SPA for all other paths.

**Build arg** (set in CapRover or CI): `VITE_API_URL=/api`

---

## Auth

On first visit (no users in DB), the UI shows a setup form for the initial admin.

All `/api/*` routes require JWT except:

- `GET /api/auth/status`
- `POST /api/auth/setup`
- `POST /api/auth/login`

Vendor APIs on the **vendor domain** remain open (no `/api` prefix).

### Permission model

| Key | Grants |
|---|---|
| `view_inbox` | Conversations inbox + chat |
| `view_accounts` | RCS / WhatsApp accounts |
| `view_webhooks` | Webhook delivery log |
| `view_settings` | Settings page |
| `view_users` | User management |
| `action_reply` | Send inbound reply |
| `action_delivery` | Trigger delivery status |
| `action_accounts_write` | Manage vendor accounts |
| `action_data_purge` | Clear all data |
| `action_users_manage` | Manage users |

Admins (`is_admin=true`) bypass all permission checks.

---

## Local development

```bash
docker compose up --build -d   # UI on :3000, API on :8080 (internal)
```

Open **http://localhost:3000** — nginx proxies `/api` to the Go server.

For UI hot-reload without Docker:

```bash
make dev          # API :8080
make web-dev      # Vite :3000, proxies /api → :8080 (vite.config.ts)
```

---

## Manual deploy

```bash
# API
caprover deploy --caproverUrl https://captain.apps.yourdomain.com \
  --appToken YOUR_API_TOKEN --appName messaging-api

# Dashboard (from repo root; CapRover uses web/Dockerfile via captain-definition override)
cd web && VITE_API_URL=/api npm run build
# Or deploy the web Docker image with API_UPSTREAM set
```

---

## Troubleshooting

| Issue | Fix |
|---|---|
| Dashboard 404 on `/api/*` | Check `API_UPSTREAM` points to the API app; verify Go listens on `/api` |
| SPA routes 404 on refresh | nginx `try_files` must serve `index.html` (included in template) |
| SSE disconnects | nginx `proxy_buffering off` on `/api/` (included) |
| CORS errors | Set `CORS_ORIGIN` to the dashboard domain only |
| Vendor paths wrong | Vendor domain must hit Go directly — paths are `/v1/...`, `/v19.0/...`, not `/api/...` |
