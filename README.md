<h1 align="center">Listen Ports</h1>

<p align="center"><strong>Real-time network socket viewer for Linux</strong></p>

<p align="center">
  <img src="https://img.shields.io/github/license/macedot/ports?color=blue" alt="License" />
  <img src="https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white" alt="Go" />
  <img src="https://img.shields.io/badge/Vue.js-3-4FC08D?logo=vue.js&logoColor=white" alt="Vue.js" />
  <img src="https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker&logoColor=white" alt="Docker" />
</p>

---

**Listen Ports** reads `/proc/net/tcp`, `/proc/net/tcp6`, `/proc/net/udp`, `/proc/net/udp6` directly — no dependency on `ss`, `netstat`, or `lsof` — and presents every TCP/UDP socket (IPv4 + IPv6) in a dark-themed, auto-refreshing web dashboard with process name resolution.

## Features

- **All sockets at a glance** — TCP, UDP, IPv4, IPv6 in a single view
- **Process resolution** — maps socket inodes to owning PID and process name
- **Client-side filtering** — by protocol (TCP / UDP / Both) and IP version (4 / 6 / Both)
- **Column sorting** — click any column header to sort ascending or descending
- **Virtual scrolling** — handles thousands of sockets without lag
- **Auto-refresh** — polls every 5 seconds, pauses when tab is hidden
- **Manual refresh** — one-click with spinner indicator
- **Color-coded badges** — protocol (TCP blue, UDP green) and connection state (LISTEN green, ESTABLISHED blue, TIME_WAIT orange)
- **Server-side caching** — configurable TTL with singleflight deduplication and stale-serve on failure

## Quick Start

```bash
docker compose up --build
```

Open [http://localhost:8080](http://localhost:8080).

### Pre-built images (from GitHub release)

```bash
GHCR_OWNER=macedot IMAGE_TAG=latest docker compose up
```

## Configuration

### Backend

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP listen port |
| `CACHE_TTL` | `2s` | Response cache TTL (Go duration: `5s`, `1m`, etc.) |
| `ADMIN_TOKEN` | _(unset)_ | Shared secret for authentication. When set, all API requests require `Authorization: Bearer <token>` header. When unset, authentication is disabled. |

### Docker Compose

| Variable | Default | Description |
|----------|---------|-------------|
| `GHCR_OWNER` | `macedot` | GHCR image namespace |
| `IMAGE_TAG` | `latest` | Image tag for both services |
| `ADMIN_TOKEN` | _(unset)_ | Uncomment in `docker-compose.yml` to enable authentication |

> The backend runs with `pid: host` and `network_mode: host` to access the host's `/proc` filesystem. This is required for socket enumeration and process resolution.

## API

### `POST /api/auth`

Validates the admin token. Always responds — when `ADMIN_TOKEN` is unset, returns `{"valid": true}` without checking.

**Request:**
```
Authorization: Bearer <token>
```

**Response (200):**
```json
{"valid": true}
```

**Response (401):**
```json
{"error": "invalid token"}
```

### `GET /api/sockets`

Returns all socket entries with optional filtering.

**Query parameters:**

| Param | Values | Default |
|-------|--------|---------|
| `proto` | `tcp`, `udp`, `both` | `both` |
| `ipver` | `4`, `6`, `both` | `both` |

**Response:**

```json
{
  "sockets": [
    {
      "protocol": "TCP",
      "local_addr": "0.0.0.0",
      "local_port": 8080,
      "remote_addr": "0.0.0.0",
      "remote_port": 0,
      "state": "LISTEN",
      "process": "nginx"
    }
  ],
  "updated_at": "2026-04-25T12:00:00Z",
  "count": 42
}
```

## Development

### Prerequisites

- Go 1.25+
- Node.js 20+
- npm

### Local development

```bash
# Terminal 1 — Backend (starts on :8080)
cd backend
go run .

# Terminal 2 — Frontend (starts on :5173, proxies /api → localhost:8080)
cd frontend
npm install
npm run dev
```

### Testing

```bash
# Backend
cd backend && go test ./...

# Frontend
cd frontend && npm test
```

## Architecture

```
┌──────────────────────────────────────────────┐
│  Single Container (ports)                   │
│  ┌─────────────────────────────────────────┐ │
│  │  Go Binary :8080                        │ │
│  │  ┌───────────────────────────────────┐  │ │
│  │  │  Embedded Vue.js SPA (go:embed)   │  │ │
│  │  │  Polls /api/sockets every 5s      │  │ │
│  │  └───────────────────────────────────┘  │ │
│  │  ┌───────────────────────────────────┐  │ │
│  │  │  Go Backend                       │  │ │
│  │  │  /proc/net/{tcp,tcp6,udp,udp6}   │  │ │
│  │  │  /proc/[pid]/fd → process map    │  │ │
│  │  │  singleflight cache (TTL)         │  │ │
│  │  │  Optional ADMIN_TOKEN auth        │  │ │
│  │  └───────────────────────────────────┘  │ │
│  └─────────────────────────────────────────┘ │
│  Requires: pid: host, network_mode: host     │
└──────────────────────────────────────────────┘
```

**How it works:**

1. **Parser** — reads `/proc/net/{tcp,tcp6,udp,udp6}` and decodes hex addresses, ports, and connection states
2. **Mapper** — walks `/proc/[pid]/fd/*` symlinks, matches `socket:[inode]` patterns to resolve the owning process
3. **Cache** — `singleflight.Group` coalesces concurrent requests; serves stale data if a fresh fetch fails
4. **API** — `POST /api/auth` for token validation and `GET /api/sockets` for socket data with protocol and IP version filters
5. **Frontend** — Pinia store with `shallowRef` for efficient updates, composable for polling with visibility API awareness, optional token-based authentication

## Deployment

The project ships as a single Docker container with the Vue.js frontend embedded in the Go binary:

| Service | Base Image | Notes |
|---------|-----------|-------|
| `ports` | `distroless/static-debian12` | Runs as non-root (UID 65534). Requires `pid: host` + `network_mode: host`. Published to `ghcr.io/macedot/ports` |

### Security

- **Authentication**: Set the `ADMIN_TOKEN` environment variable to enable token-based authentication. When set, a login form is shown and all API requests require the token via `Authorization: Bearer` header. Token is stored in `sessionStorage` (cleared on tab close).
- **No auth mode**: When `ADMIN_TOKEN` is unset, the application is freely accessible. Only use this on trusted networks.
- **Security headers**: `X-Content-Type-Options: nosniff` and `X-Frame-Options: DENY` are set on all responses.
- **Non-root container**: The process runs as UID 65534 (nobody) inside the container.
- **TLS**: The server listens on plain HTTP. For production, run behind a TLS-terminating reverse proxy (nginx, Caddy, Traefik).

## CI/CD

GitHub Actions builds and pushes Docker images to GHCR on every published release (prereleases excluded). Images are tagged with both the release version and `latest`.

## License

This project is licensed under the [GNU Affero General Public License v3.0](LICENSE).
