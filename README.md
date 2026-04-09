# Notorious Search Platform

Notorious is an internal, production-grade search platform built for teams that need fast discovery over large datasets with strong access control and auditability.

The system is designed to handle high-volume search traffic on top of a 500GB+ OpenSearch data layer, while PostgreSQL manages application state, user controls, sessions, and operational metadata.

## What Problem This Solves

Teams often face the same challenges when running internal search tools:

- Data is large and distributed, so lookups become slow or inconsistent.
- Access is hard to govern across different user roles.
- Search usage needs limits, monitoring, and accountability.
- Admins need tooling to manage users and requests without direct database intervention.

Notorious solves this with a single platform that combines search performance, role-aware workflows, usage governance, and complete audit visibility.

## Core Capabilities

### Search at Scale

- OpenSearch-backed querying over 500GB+ indexed data.
- Supports logical operators and query refinement flows.
- Smart search counting (usage increments only on successful result returns).
- Suggest and refine endpoints for faster query iteration.
- Export support for end-of-day reporting workflows.

### Access Control and Identity

- JWT authentication with secured API routes.
- Role-based authorization for user and admin scopes.
- Password hashing with bcrypt.
- Session tracking and session revocation.
- Active/inactive account lifecycle controls.

### Operational Guardrails

- Per-user daily search limits with automatic reset.
- IST-aware limit reset behavior for predictable daily quotas.
- Rate-limited auth, search, and admin endpoints.
- Bot/automation filtering middleware for public entrypoints.

### Admin and Internal Tooling

- User lifecycle management (create, update, deactivate, delete).
- Access request review (approve/reject).
- Password change request workflow.
- Search history and usage visibility by user and system-wide.
- Admin session management and online-user visibility.

### Collaboration and Communication

- Built-in chat and WebSocket-based real-time messaging.
- Unread tracking, broadcast messages, and presence monitoring.

### Upload and Ingestion Support

- S3 multipart upload flow (`init`, `presign`, `complete`, `abort`).
- Ingestion utilities and worker tuning options for large data feeds.

## Architecture Overview

### Backend

- Go 1.24 service using Gin.
- PostgreSQL via pgx for transactional and metadata storage.
- OpenSearch integration for search and indexing workflows.
- Layered structure: handlers -> services -> repository -> database.
- Scheduled reset services and middleware-based security controls.

### Frontend

- Next.js (App Router) with TypeScript.
- Auth-aware routing and protected views.
- User search interface and admin dashboard modules.
- Centralized API endpoint configuration.

## High-Level Flow

1. User authenticates and receives a JWT.
2. Search request is authorized and rate-checked.
3. Query executes against OpenSearch indices.
4. Result metadata is persisted in PostgreSQL.
5. Limits, history, and dashboard metrics update in real time.

## Repository Layout

```text
notorious/
├── backend/
│   ├── main.go
│   ├── migrations/
│   ├── cmd/
│   ├── internal/
│   │   ├── auth/
│   │   ├── config/
│   │   ├── database/
│   │   ├── handlers/
│   │   ├── middleware/
│   │   ├── models/
│   │   ├── repository/
│   │   ├── scheduler/
│   │   ├── services/
│   │   ├── utils/
│   │   └── websocket/
│   └── scripts/
└── frontend/
    ├── src/app/
    ├── src/components/
    ├── src/contexts/
    ├── src/services/
    ├── src/config/
    └── src/types/
```

## Local Setup

### Prerequisites

- Go 1.24+
- PostgreSQL 14+
- Node.js 18+
- pnpm
- Reachable OpenSearch cluster/index

### 1) Clone and install dependencies

```bash
git clone <your-repo-url>
cd notorious

cd backend && go mod download
cd ../frontend && pnpm install
```

### Using env samples

Linux/macOS:

```bash
cp backend/.env.sample backend/.env
cp frontend/.env.sample frontend/.env.local
```

Windows PowerShell:

```powershell
Copy-Item backend/.env.sample backend/.env
Copy-Item frontend/.env.sample frontend/.env.local
```

Windows Command Prompt:

```bat
copy backend\.env.sample backend\.env
copy frontend\.env.sample frontend\.env.local
```

### 2) Configure backend environment

Create `backend/.env` with your own values:

```env
PORT=8080
DATABASE_URL=postgresql://<user>:<password>@localhost:5432/notorious
JWT_SECRET=<long-random-secret>

OPENSEARCH_ENDPOINT=https://<your-opensearch-endpoint>
OPENSEARCH_INDEX=<primary-index>
# Optional: comma-separated list of search indices
OPENSEARCH_INDICES=<index-a>,<index-b>
OPENSEARCH_MASTER_USER=<opensearch-username>
OPENSEARCH_MASTER_PASSWORD=<opensearch-password>

# Optional upload/ingest settings
AWS_REGION=us-east-1
S3_UPLOAD_BUCKET=<bucket-name>
S3_UPLOAD_PREFIX=ingest/raw/
AWS_ACCESS_KEY_ID=<aws-access-key-id>
AWS_SECRET_ACCESS_KEY=<aws-secret-access-key>
```

### 3) Initialize database

```bash
cd backend
createdb notorious
go run main.go
```

On startup, backend migrations are applied automatically from `backend/migrations`.

### 4) Configure frontend

Create `frontend/.env.local`:

```env
NEXT_PUBLIC_API_URL=http://localhost:8080
```

### 5) Run both services

```bash
# Terminal 1
cd backend
go run main.go

# Terminal 2
cd frontend
pnpm dev
```

Frontend: `http://localhost:3000`
Backend health: `http://localhost:8080/health`

## API Surface (Summary)

### Public

- `POST /auth/login`
- `POST /auth/request-access`
- `POST /auth/revoke-session`

### Authenticated User

- `GET|POST /search`
- `POST /search/refine`
- `GET /search/suggest`
- `GET /search/export-eod`
- `GET /api/user/search-history`
- `GET /api/user/metadata`
- `POST /api/user/heartbeat`
- `POST /api/user/password-change/request`
- `GET /api/user/password-change/requests`

### Admin

- User management endpoints under `/api/admin/users`
- Request handling under `/api/admin/user-requests`
- Password-change review under `/api/admin/password-change-requests`
- Search visibility under `/api/admin/search-history`
- Session and online metrics under `/api/admin/sessions`, `/api/admin/users/online`

### Realtime and Upload

- WebSocket endpoint: `GET /ws`
- Chat endpoints under `/api/chat/*`
- Multipart upload endpoints under `/upload/*`

## Security Notes

- Use strong, rotated secrets for JWT and service credentials.
- Keep all environment files out of version control.
- Restrict network access to Postgres and OpenSearch to trusted networks.
- Prefer HTTPS/TLS termination in production.
- Monitor rate-limit events and failed auth patterns.

## Production Readiness Checklist

- Configure managed PostgreSQL backups and retention.
- Configure OpenSearch snapshots and index lifecycle policies.
- Add centralized logging and request tracing.
- Run behind a reverse proxy / load balancer with TLS.
- Set up alerting for auth spikes, 5xx errors, and latency.
- Periodically verify migration, restore, and rollback workflows.

## Troubleshooting

### Backend fails to boot

- Verify `DATABASE_URL` and `JWT_SECRET` are set.
- Confirm Postgres is reachable from the backend host.
- Check backend logs for migration or connection errors.

### Search returns empty unexpectedly

- Verify OpenSearch endpoint and credentials.
- Confirm target index names in `OPENSEARCH_INDEX` or `OPENSEARCH_INDICES`.
- Validate the query payload format and selected fields.

### Frontend cannot reach API

- Verify `NEXT_PUBLIC_API_URL` points to the running backend.
- Check CORS allowlist for your frontend origin.
- Confirm auth token presence for protected routes.

## Tech Stack

### Backend

- Go
- Gin
- PostgreSQL + pgx
- OpenSearch client
- JWT + bcrypt
- WebSockets

### Frontend

- Next.js
- React
- TypeScript
- Tailwind CSS

## Status

This project is actively positioned as production-grade internal software for secure, high-volume search operations and admin-managed access workflows.

## License

Private project.
