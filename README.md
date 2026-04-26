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
- **Docker container detection** — optionally shows container name, image, and network mode for sockets when Docker socket is mounted (opt-in, requires docker.sock read-only mount)

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
| `PROC_PATH` | `/proc` | Base path for procfs. Set to `/host-proc` when using the hardened docker-compose. |
| `ADMIN_TOKEN` | _(unset)_ | Shared secret for authentication. When set, all API requests require `Authorization: Bearer <token>` header. When unset, authentication is disabled. |
| `DOCKER_HOST` | _(unset)_ | Path to Docker unix socket. Set to `/var/run/docker.sock` when mounting the Docker socket for container monitoring. |

### Docker Compose

| Variable | Default | Description |
|----------|---------|-------------|
| `GHCR_OWNER` | `macedot` | GHCR image namespace |
| `IMAGE_TAG` | `latest` | Image tag for both services |
| `ADMIN_TOKEN` | _(unset)_ | Uncomment in `docker-compose.yml` to enable authentication |

> The hardened docker-compose mounts the host's `/proc` read-only at `/host-proc` via `PROC_PATH`. No `pid: host` or `network_mode: host` required. Socket data (addresses, ports, states) is fully functional. Process name resolution requires `SYS_PTRACE` capability (included in docker-compose).

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

### `GET /api/containers`

Returns all Docker containers with port mappings and metadata. Requires authentication when `ADMIN_TOKEN` is set.

**Response:**

```json
{
  "containers": [
    {
      "id": "a1b2c3d4e5f6",
      "name": "nginx",
      "image": "nginx:alpine",
      "status": "running",
      "state": "running",
      "network_mode": "bridge",
      "pid": 1234,
      "ports": [
        {"host_port": 8080, "container_port": 80, "protocol": "tcp"}
      ]
    }
  ]
}
```

## Docker Container Monitoring (Optional)

Listen Ports can optionally detect Docker containers associated with network sockets. When enabled, the socket table shows:
- **Container name** — the Docker container name (e.g. `nginx`, `postgres`)
- **Image** — the container image (shown as tooltip)
- **Network mode** — `host`, `bridge`, or other

### Enabling

1. Mount the Docker socket (read-only) in `docker-compose.yml`:
   ```yaml
   volumes:
     - /proc:/host-proc:ro
     - /var/run/docker.sock:/var/run/docker.sock:ro
   environment:
     - DOCKER_HOST=/var/run/docker.sock
   ```

2. Restart: `docker compose up -d`

### How it works

- **Host-network containers** — matched by PID (process ID inside the container equals host PID)
- **Bridge-network containers** — matched by port binding (container's published port matches the socket's local port)
- Container data is cached for 10 seconds to avoid hammering the Docker API

### Security considerations

- The Docker socket is a **privileged interface** — mounting it grants the container full Docker API access (read-only mount mitigates mutation but not information disclosure)
- Only enable on trusted hosts
- The socket is mounted `:ro` (read-only) as a defense-in-depth measure
- Container detection is **purely informational** — no container management operations are performed

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
│  Volume: /proc → /host-proc (read-only)      │
│  Volume: /var/run/docker.sock (optional, :ro) │
└──────────────────────────────────────────────┘
```

**How it works:**

1. **Parser** — reads `/proc/net/{tcp,tcp6,udp,udp6}` and decodes hex addresses, ports, and connection states
2. **Mapper** — walks `/proc/[pid]/fd/*` symlinks, matches `socket:[inode]` patterns to resolve the owning process
3. **Cache** — `singleflight.Group` coalesces concurrent requests; serves stale data if a fresh fetch fails
4. **API** — `POST /api/auth` for token validation and `GET /api/sockets` for socket data with protocol and IP version filters
5. **Frontend** — Pinia store with `shallowRef` for efficient updates, composable for polling with visibility API awareness, optional token-based authentication
6. **Docker Monitor** (optional) — queries Docker Engine API via unix socket for container metadata, maps containers to sockets by PID (host-network) or port binding (bridge-network)

## Deployment

The project ships as a single Docker container with the Vue.js frontend embedded in the Go binary:

| Service | Base Image | Notes |
|---------|-----------|-------|
| `ports` | `distroless/static-debian12` | Runs as non-root (UID 65534) with minimal capabilities. Host `/proc` mounted read-only. Published to `ghcr.io/macedot/ports` |

### Security

- **Authentication**: Set the `ADMIN_TOKEN` environment variable to enable token-based authentication. When set, a login form is shown and all API requests require the token via `Authorization: Bearer` header. Token is stored in `sessionStorage` (cleared on tab close).
- **Hardened container**: No host modes (`pid: host`, `network_mode: host`). Host `/proc` mounted read-only at `/host-proc`. Read-only root filesystem, all capabilities dropped except `SYS_PTRACE` (required for process name resolution via `/proc/[pid]/fd/`). No-new-privileges enforced. Resource limits: 128MB memory, 0.5 CPU.
- **No auth mode**: When `ADMIN_TOKEN` is unset, the application is freely accessible. Only use this on trusted networks.
- **Security headers**: `X-Content-Type-Options: nosniff` and `X-Frame-Options: DENY` are set on all responses.
- **Non-root container**: The process runs as UID 65534 (nobody) inside the container.
- **TLS**: The server listens on plain HTTP. For production, run behind a TLS-terminating reverse proxy (nginx, Caddy, Traefik).

## CI/CD

GitHub Actions builds and pushes Docker images to GHCR on every published release (prereleases excluded). Images are tagged with both the release version and `latest`.

## License

This project is licensed under the [GNU Affero General Public License v3.0](LICENSE).
