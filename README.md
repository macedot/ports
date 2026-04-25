<p align="center">
  <h1 align="center">Listen Ports</h1>
  <p align="center">
    <strong>Real-time network socket viewer for Linux</strong>
  </p>
  <p align="center">
    <img src="https://img.shields.io/badge/license-AGPL--3.0-blue" alt="License: AGPL-3.0" />
    <img src="https://img.shields.io/badge/Go-1.25-00ADD8?logo=go" alt="Go" />
    <img src="https://img.shields.io/badge/Vue.js-3-4FC08D?logo=vue.js" alt="Vue.js" />
    <img src="https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker" alt="Docker" />
  </p>
</p>

---

**Listen Ports** reads `/proc/net/tcp`, `/proc/net/tcp6`, `/proc/net/udp`, `/proc/net/udp6` directly — no dependency on `ss`, `netstat`, or `lsof` — and presents every TCP/UDP socket (IPv4 + IPv6) in a dark-themed, auto-refreshing web dashboard with process name resolution.

![Listen Ports dashboard showing socket table](docs/screenshot.png)

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

Open [http://localhost:80](http://localhost:80).

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

### Docker Compose

| Variable | Default | Description |
|----------|---------|-------------|
| `GHCR_OWNER` | `macedot` | GHCR image namespace |
| `IMAGE_TAG` | `latest` | Image tag for both services |

> The backend runs with `pid: host` and `network_mode: host` to access the host's `/proc` filesystem. This is required for socket enumeration and process resolution.

## API

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
│  Browser :80                                 │
│  ┌─────────────────────────────────────────┐ │
│  │  Vue.js 3 + Pinia + vue-virtual-scroller│ │
│  │  Polls /api/sockets every 5s            │ │
│  └─────────────┬───────────────────────────┘ │
│                │                              │
│  ┌─────────────▼───────────────────────────┐ │
│  │  nginx (reverse proxy)                  │ │
│  │  /api/* → backend:8080                  │ │
│  │  /*     → static SPA                    │ │
│  └─────────────┬───────────────────────────┘ │
│                │                              │
│  ┌─────────────▼───────────────────────────┐ │
│  │  Go Backend :8080                       │ │
│  │  /proc/net/{tcp,tcp6,udp,udp6} parser  │ │
│  │  /proc/[pid]/fd → inode → process map  │ │
│  │  singleflight cache (configurable TTL)  │ │
│  └─────────────────────────────────────────┘ │
└──────────────────────────────────────────────┘
```

**How it works:**

1. **Parser** — reads `/proc/net/{tcp,tcp6,udp,udp6}` and decodes hex addresses, ports, and connection states
2. **Mapper** — walks `/proc/[pid]/fd/*` symlinks, matches `socket:[inode]` patterns to resolve the owning process
3. **Cache** — `singleflight.Group` coalesces concurrent requests; serves stale data if a fresh fetch fails
4. **API** — single `GET /api/sockets` endpoint with protocol and IP version filters
5. **Frontend** — Pinia store with `shallowRef` for efficient updates, composable for polling with visibility API awareness

## Deployment

The project ships three Docker containers orchestrated by Compose:

| Service | Base Image | Notes |
|---------|-----------|-------|
| `backend` | `distroless/static-debian12` | Requires `pid: host` + `network_mode: host` |
| `frontend` | `nginx:alpine` | Serves static SPA + proxies `/api` |
| — | — | Published to `ghcr.io/macedot/ports/{backend,frontend}` |

## CI/CD

GitHub Actions builds and pushes Docker images to GHCR on every published release (prereleases excluded). Images are tagged with both the release version and `latest`.

## License

This project is licensed under the [GNU Affero General Public License v3.0](LICENSE).
