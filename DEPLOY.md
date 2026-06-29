# Deployment Guide

Deploy **backend** to CapRover and **frontend** to Cloudflare Pages via GitHub Actions (`.github/workflows/deploy.yml`).

## Prerequisites

1. **CapRover** server with a deployed app (e.g. `messaging-api`)
2. **PostgreSQL** reachable from CapRover (CapRover one-click Postgres or external managed DB)
3. **Cloudflare** account with Pages project created (e.g. `messaging-simulator`)
4. GitHub repo with Actions enabled

---

## GitHub Secrets

Add under **Settings → Secrets and variables → Actions → Secrets**:

| Secret | Description | Example |
|---|---|---|
| `CAPROVER_SERVER` | CapRover dashboard URL (HTTPS) | `https://captain.apps.yourdomain.com` |
| `CAPROVER_APP_TOKEN` | App deploy token from CapRover app → Deployment → Method 3 | `eyJhbGciOiJIUzI1NiIs...` |
| `CLOUDFLARE_API_TOKEN` | Cloudflare API token with **Cloudflare Pages → Edit** permission | `abc123...` |
| `CLOUDFLARE_ACCOUNT_ID` | Cloudflare account ID (dashboard URL or Overview page) | `a1b2c3d4e5f6...` |

### How to get CapRover app token

1. CapRover dashboard → your app → **Deployment** tab
2. Select **Method 3: Deploy from GitHub/Bitbucket/GitLab**
3. Copy the **app token** (not the root password)

### How to get Cloudflare credentials

1. **Account ID:** Cloudflare dashboard → any zone → right sidebar, or Workers & Pages overview
2. **API Token:** [dash.cloudflare.com/profile/api-tokens](https://dash.cloudflare.com/profile/api-tokens) → Create Token → Edit Cloudflare Pages template

---

## GitHub Variables

Add under **Settings → Secrets and variables → Actions → Variables**:

| Variable | Description | Example |
|---|---|---|
| `CAPROVER_APP_NAME` | CapRover app name (no spaces) | `messaging-api` |
| `CLOUDFLARE_PAGES_PROJECT` | Cloudflare Pages project name | `messaging-simulator` |
| `VITE_API_URL` | Public backend API URL used by the frontend build | `https://messaging-api.apps.yourdomain.com/api` |

> **Important:** `VITE_API_URL` must be the full URL to your CapRover backend **including `/api`** suffix, because the React app calls `/accounts`, `/conversations`, etc. under that base.

---

## CapRover App Environment Variables

Set in CapRover → app → **App Configs** → **Environment Variables**:

| Variable | Required | Description | Example |
|---|---|---|---|
| `DATABASE_URL` | Yes | PostgreSQL connection string | `postgres://user:pass@srv-captain--postgres:5432/messaging_sim?sslmode=disable` |
| `PORT` | No | HTTP port (CapRover sets automatically) | `80` |
| `CORS_ORIGIN` | Yes | Allowed frontend origin(s), comma-separated | `https://messaging-simulator.pages.dev,https://simulator.yourdomain.com` |
| `MEDIA_STORAGE_PATH` | No | Uploaded media directory | `/data/media` |
| `ENABLE_DATA_RESET` | No | Allow data purge API | `true` |
| `DEFAULT_WEBHOOK_URL` | No | Default webhook for new accounts | `https://your-platform/webhooks` |
| `JWT_SECRET` | **Yes (prod)** | Secret for signing admin UI JWTs (min 32 chars) | `openssl rand -base64 48` |
| `JWT_EXPIRY_HOURS` | No | Admin session token lifetime in hours | `24` |

> **Auth:** On first visit (no users in DB), the web UI shows a setup form to create the initial admin. Vendor APIs (`/v1/...`, `/v19.0/...`) remain open for platform integration testing. All `/api/*` admin routes require a valid JWT except `/api/auth/status`, `/api/auth/setup`, and `/api/auth/login`.

### Permission model

Each user has **view** permissions (nav visibility) and **action** permissions (API writes):

| Key | Grants |
|---|---|
| `view_inbox` | Conversations inbox + chat |
| `view_accounts` | RCS / WhatsApp accounts |
| `view_webhooks` | Webhook delivery log |
| `view_settings` | Settings page |
| `view_users` | User management (admin only in practice) |
| `action_reply` | Send inbound reply from conversation |
| `action_delivery` | Manually trigger delivery status |
| `action_accounts_write` | Create / edit / delete accounts |
| `action_data_purge` | Clear all data |
| `action_users_manage` | Create users, reset passwords, activate/deactivate |

Admins (`is_admin=true`) bypass all permission checks.

### CapRover persistent volume

Mount a persistent volume so uploaded media survives redeploys:

- **Path in container:** `/data/media`
- CapRover → app → **App Configs** → **Persistent Directories** → `/data/media`

### CapRover HTTP settings

- Container HTTP Port: **8080** (or leave default if CapRover maps correctly)
- Enable HTTPS via CapRover/nginx

---

## Cloudflare Pages

1. Create a Pages project named `messaging-simulator` (or match `CLOUDFLARE_PAGES_PROJECT`)
2. First deploy happens via GitHub Actions — no need to connect Git in Cloudflare UI if using wrangler deploy
3. Optional: add custom domain in Cloudflare Pages → Custom domains
4. Add that custom domain to CapRover `CORS_ORIGIN`

---

## Domain layout (recommended)

| Service | URL |
|---|---|
| Frontend (Cloudflare Pages) | `https://messaging-simulator.pages.dev` |
| Backend (CapRover) | `https://messaging-api.apps.yourdomain.com` |
| Vendor APIs (same backend) | `https://messaging-api.apps.yourdomain.com/v1/...` and `/v19.0/...` |

Set `VITE_API_URL=https://messaging-api.apps.yourdomain.com/api`

---

## Manual deploy (local)

```bash
# Backend to CapRover (install caprover CLI first)
caprover deploy --caproverUrl https://captain.apps.yourdomain.com \
  --appToken YOUR_APP_TOKEN \
  --appName messaging-api

# Frontend to Cloudflare Pages
cd web && VITE_API_URL=https://your-api.com/api npm run build
npx wrangler pages deploy dist --project-name=messaging-simulator
```

---

## Troubleshooting

| Issue | Fix |
|---|---|
| CORS errors in browser | Set `CORS_ORIGIN` on CapRover to exact Cloudflare Pages URL |
| Frontend can't reach API | Verify `VITE_API_URL` includes `https://` and `/api` |
| DB connection failed | Check `DATABASE_URL` host is reachable from CapRover container |
| Migrations fail on boot | Ensure Postgres is running before app starts; check logs |
