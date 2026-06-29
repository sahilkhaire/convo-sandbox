# Messaging Vendor Simulator

A **WhatsApp + RCS vendor simulator** for dev/testing. Point your messaging platform at this service instead of Meta or Jio to simulate message sending, delivery webhooks, and inbound conversations.

## Features

- **Jio RCS (JBM v2.2)** API parity: OAuth, send messages (basic/rich/carousel), capabilities, batch, upload, revoke, assistant events
- **Meta WhatsApp Cloud API** parity: send messages (all types), media upload, webhook verification
- **PostgreSQL** storage for all messages, conversations, accounts, webhook logs
- **Webhook dispatcher** — POSTs delivery/inbound events to your configured webhook URL
- **Conversation UI** — inbox, chat window, reply as end-user, manual delivery triggers
- **Clear all data** — reset simulator state from Settings (messages-only or full reset)

## Quick Start

### Docker (recommended)

```bash
docker compose up --build -d
```

- API: http://localhost:8080
- UI: http://localhost:3000

Run migrations and seed demo accounts:

```bash
make migrate
make seed
```

### Local development

```bash
# Start Postgres (uses port 5433 to avoid conflict with local PostgreSQL on 5432)
docker compose up -d postgres

# Migrate
make migrate

# Seed demo accounts
make seed

# Run API
make dev

# Run UI (separate terminal)
make web-dev
```

## Domain Swap (your platform)

Point your dev environment at the simulator using **production-identical paths**:

| Production host | Simulator base |
|---|---|
| `https://tgs.businessmessaging.jio.com` | `http://localhost:8080` |
| `https://api.businessmessaging.jio.com` | `http://localhost:8080` |
| `https://graph.facebook.com` | `http://localhost:8080` |

Legacy dev aliases `/rcs` and `/whatsapp` are also mounted for backward compatibility.

### RCS example (production paths)

```bash
# Get token — GET /v1/oauth/token?grant_type=client_credentials&client_id={assistantId}&client_secret={secret}&scope=read
curl "http://localhost:8080/v1/oauth/token?grant_type=client_credentials&client_id=6544c5b408febf98e5fc5ec4&client_secret=demo_secret&scope=read"

# Send message — POST /v1/messaging/users/{userPhoneNumber}/assistantMessages/async?messageId={uid}&assistantId={id}
curl -X POST "http://localhost:8080/v1/messaging/users/%2B919876543210/assistantMessages/async?assistantId=6544c5b408febf98e5fc5ec4&messageId=test123" \
  -H "Authorization: Bearer rcs_token_demo" \
  -H "Content-Type: application/json" \
  -d '{"content":{"plainText":"Hello from simulator"},"messageTrafficType":"TRANSACTION"}'
```

### WhatsApp example (production paths)

```bash
# POST /v19.0/{phone-number-id}/messages
curl -X POST "http://localhost:8080/v19.0/123456789012345/messages" \
  -H "Authorization: Bearer wa_token_demo" \
  -H "Content-Type: application/json" \
  -d '{"messaging_product":"whatsapp","recipient_type":"individual","to":"919876543210","type":"text","text":{"body":"Hello"}}'
```

Configure webhook URLs per account in the UI (**Accounts**) or via `POST /api/accounts`.

## API Routes

### Vendor APIs (production-identical paths)

**Jio RCS (JBM v2.2)**
- `GET /v1/oauth/token` — query: `grant_type`, `client_id`, `client_secret`, `scope`
- `POST /v1/messaging/users/{userPhoneNumber}/assistantMessages/async` — query: `messageId`, `assistantId`
- `DELETE /v1/messaging/users/{userPhoneNumber}/assistantMessages/{messageID}`
- `POST /v1/messaging/users/{userPhoneNumber}/assistantEvents` — body: `eventType`, `messageId`
- `GET /v1/messaging/users/{userPhoneNumber}/capabilities` — query: `assistantId`
- `POST /v1/messaging/usersBatchGet` — body: `phoneNumbers[]`
- `POST /v1/messaging/upload/files` — body: `fileName`, `contentType`, `fileContent`

**Meta WhatsApp Cloud API**
- `GET /v19.0/{id}` — webhook verify: `hub.mode`, `hub.verify_token`, `hub.challenge`
- `POST /v19.0/{phone-number-id}/messages` — send / mark-read
- `POST /v19.0/{phone-number-id}/media` — multipart: `file`, `type`, `messaging_product`
- `GET /v19.0/{media-id}` — media URL
- `DELETE /v19.0/{media-id}`
- Also supported: `v20.0`, `v21.0`

### Admin API (UI)

- `GET/POST /api/accounts`
- `GET /api/conversations`
- `GET/POST /api/conversations/{id}/messages`
- `POST /api/messages/{id}/status` — manual delivery trigger
- `GET /api/webhooks` — webhook delivery log
- `GET /api/events` — SSE stream
- `DELETE /api/data?scope=messages|all` — purge PostgreSQL data

## Environment

See [`.env.example`](.env.example).

| Variable | Default | Description |
|---|---|---|
| `DATABASE_URL` | local postgres DSN | PostgreSQL connection |
| `ENABLE_DATA_RESET` | `true` | Allow `DELETE /api/data` |
| `MEDIA_STORAGE_PATH` | `./data/media` | Uploaded file storage |
| `CORS_ORIGIN` | `http://localhost:3000` | UI origin |

## Project Structure

```
cmd/server/     Go API server
cmd/seed/       Demo account seeder
internal/rcs/   Jio RCS handlers
internal/whatsapp/  Meta WhatsApp handlers
internal/admin/ Admin REST + SSE
internal/core/  Messaging + webhook dispatcher
migrations/     PostgreSQL schema (goose)
web/            React conversation UI
```

## Testing

```bash
go test ./...
```

## Deployment

See [DEPLOY.md](DEPLOY.md) for GitHub Actions CI/CD to **Cloudflare Pages** (frontend) and **CapRover** (backend), including all required secrets and variables.

## License

Internal dev tool — Zixflow
