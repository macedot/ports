<h1 align="center">Listen Ports</h1>

<p align="center"><strong>Real-time network socket viewer for Linux</strong></p>

<p align="center">
  <img src="https://img.shields.io/github/license/macedot/ports?color=blue" alt="License" />
  <img src="https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white" alt="Go" />
  <img src="https://img.shields.io/badge/Vue.js-3-4FC08D?logo=vue.js&logoColor=white" alt="Vue.js" />
  <img src="https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker&logoColor=white" alt="Docker" />
</p>

---

**Listen Ports** reads `/proc/net/tcp`, `/proc/net/tcp6`, `/proc/net/udp`, `/proc/net/udp6` directly — no dependency on `ss`, `netstat`, or `lsof` — and presents every TCP/UDP socket (IPv4 + IPv6) in a dark-themed, auto-refreshing web dashboard with process name resolution and optional Docker container detection.

## Features

- **All sockets at a glance** — TCP, UDP, IPv4, IPv6 in a single view
- **Process resolution** — maps socket inodes to owning PID and process name, with full command line and executable path
- **Docker container detection** — shows container name, image, and network mode for sockets (opt-in via Docker socket mount)
- **Regex search** — filter sockets by any field including command line and executable path
- **Client-side filtering** — by protocol (TCP / UDP), IP version (4 / 6), and container (All / With Container / No Container)
- **Freeze/Pause** — pause live updates to inspect a snapshot; resume with one click
- **CSV & JSON export** — download the current filtered view as CSV or JSON
- **Column sorting** — click any column header to sort ascending or descending
- **Port grouping** — sockets grouped by local port with collapsible groups
- **Virtual scrolling** — handles thousands of sockets without lag
- **Auto-refresh** — polls every 5 seconds, pauses when tab is hidden
- **Color-coded badges** — protocol (TCP blue, UDP green) and connection state (LISTEN green, ESTABLISHED blue, TIME_WAIT orange)
- **Server-side caching** — configurable TTL with singleflight deduplication and stale-serve on failure
- **Authentication** — optional token-based auth via `ADMIN_TOKEN` environment variable

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
| `CACHE_TTL` | `10s` | Response cache TTL (Go duration: `5s`, `1m`, etc.) |
| `PROC_PATH` | `/proc` | Base path for procfs. Set to `/host-proc` when using the hardened docker-compose. |
| `ADMIN_TOKEN` | _(unset)_ | Shared secret for authentication. When set, all API requests require `Authorization: Bearer <token>` header. When unset, authentication is disabled. |
| `DOCKER_HOST` | _(unset)_ | Path to Docker unix socket. Set to `/var/run/docker.sock` when mounting the Docker socket for container monitoring. |

### Docker Compose

| Variable | Default | Description |
|----------|---------|-------------|
| `GHCR_OWNER` | `macedot` | GHCR image namespace |
| `IMAGE_TAG` | `latest` | Image tag for both services |
| `ADMIN_TOKEN` | _(unset)_ | Uncomment in `docker-compose.yml` to enable authentication |

### Required Configuration for Process Names

> **Process name resolution requires `pid: host` and `SYS_PTRACE` capability.** Without these, socket data (addresses, ports, states) works fine but all process names will be empty.
>
> The Linux kernel only exposes socket→PID mappings via `/proc/[pid]/fd/` symlinks. Reading these symlinks requires `ptrace_may_access` in the **host's** PID namespace — container capabilities don't transfer to the host namespace. `pid: host` makes the container share the host's PID namespace so capabilities apply.
>
> ```yaml
> pid: host                # share host PID namespace (required for process resolution)
> cap_add:
>   - SYS_PTRACE           # read /proc/[pid]/fd/ symlinks
> ```
>
> **Optional — root user for full resolution:**
>
> The default container user is `65534:65534` (nobody). Process names for **root-owned host processes** (sshd, nginx, etc.) will be empty because ptrace requires matching UID or root. To see all process names, uncomment `user: "0:0"` in `docker-compose.yml`:
>
> ```yaml
> user: "0:0"  # run as root — all host processes readable (less secure)
> ```
>
> The provided `docker-compose.yml` already includes these. If using a custom compose, add them to your service definition.

The docker-compose mounts the host's `/proc` read-only at `/host-proc` via `PROC_PATH`. No `network_mode: host` required. Socket data (addresses, ports, states) is fully functional without any special configuration — only process name resolution needs `pid: host`.

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
      "process": "nginx",
      "command": "nginx: worker process",
      "exe": "/usr/sbin/nginx",
      "container": "nginx",
      "c_image": "nginx:alpine",
      "c_network": "bridge"
    }
  ],
  "updated_at": "2026-04-25T12:00:00Z",
  "count": 42
}
```

The `command`, `exe`, `container`, `c_image`, and `c_network` fields are only present when the data is available (omitted when empty).

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
      "status": "Up 2 hours",
      "state": "running",
      "network_mode": "bridge",
      "pid": 1234,
      "ports": [
        {"host_ip": "0.0.0.0", "host_port": 8080, "container_port": 80, "protocol": "tcp"}
      ]
    }
  ]
}
```

## Process Name Resolution

The mapper resolves process names using a multi-source fallback chain:

1. **`/proc/[pid]/status`** — reads the `Name:` field (kernel comm name, limited to 15 chars)
2. **`/proc/[pid]/comm`** — reads the comm file directly
3. **`/proc/[pid]/cmdline`** — reads the full command line, extracts basename of first argument
4. **`/proc/[pid]/exe`** — readlink on the exe symlink, extracts basename (e.g. `/usr/bin/docker-proxy` → `docker-proxy`)
5. **`pid:1234`** — last resort showing only the PID number

The `command` field always contains the full command line with arguments (when readable). The `exe` field contains the full executable path. Hovering over the process name in the UI shows a tooltip with the full command and executable path.

> **Note:** Process name resolution requires `pid: host` in docker-compose. The kernel only exposes socket→PID ownership via `/proc/[pid]/fd/` symlinks, which requires ptrace access in the host PID namespace. Without `pid: host`, process names will be empty. The startup log will print a clear warning when this is misconfigured.

## Docker Container Monitoring (Optional)

Listen Ports can optionally detect Docker containers associated with network sockets. When enabled, the socket table shows:
- **Container name** — displayed as a purple badge in the Container column
- **Image** — shown as a tooltip on hover over the container badge
- **Network mode** — `host`, `bridge`, or other

### Enabling

1. Find your host's Docker group GID:
   ```bash
   getent group docker | cut -d: -f3
   # → 999 (or 1001, etc.)
   ```

2. Uncomment the Docker socket lines in `docker-compose.yml` **and** set `user`:

   ```yaml
   volumes:
     - /proc:/host-proc:ro
     - /var/run/docker.sock:/var/run/docker.sock:ro   # ← uncomment
   user: "65534:999"                                   # ← replace 999 with your host docker GID
   environment:
     DOCKER_HOST: "/var/run/docker.sock"               # ← uncomment
   ```

3. Restart: `docker compose up -d`

> **Why `user` is required:** The container runs as non-root (UID 65534). The Docker socket on the host is owned by `root:docker` with mode `660`. Setting `user: "65534:<gid>"` makes `docker` the primary group of the process so the socket is readable. Without this, container detection silently fails.

### How it works

- **Host-network containers** — matched by PID. The container's host PID (from Docker inspect API) is matched against the process owning each socket.
- **Bridge-network containers** — matched by port binding. The container's published host port is matched against the socket's local port.
- Container data is cached for 10 seconds to minimize Docker API calls.

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
┌──────────────────────────────────────────────────────────┐
│ Single Container (ports)                                 │
│ ┌──────────────────────────────────────────────────────┐ │
│ │ Go Binary :8080                                      │ │
│ │ ┌──────────────────────────────────────────────────┐ │ │
│ │ │ Embedded Vue.js SPA (go:embed)                   │ │ │
│ │ │ Polls /api/sockets every 5s                      │ │ │
│ │ │ Freeze toggle + CSV/JSON export                  │ │ │
│ │ └──────────────────────────────────────────────────┘ │ │
│ │ ┌──────────────────────────────────────────────────┐ │ │
│ │ │ Go Backend                                       │ │ │
│ │ │ /proc/net/{tcp,tcp6,udp,udp6}                    │ │ │
│ │ │ /proc/[pid]/fd → process map                     │ │ │
│ │ │ /proc/[pid]/{status,comm,cmdline,exe}            │ │ │
│ │ │ singleflight cache (TTL)                         │ │ │
│ │ │ ADMIN_TOKEN auth                                 │ │ │
│ │ │ Docker monitor (optional)                        │ │ │
│ │ └──────────────────────────────────────────────────┘ │ │
│ └──────────────────────────────────────────────────────┘ │
│ Volume: /proc → /host-proc (read-only)                   │
│ Volume: /etc/localtime (read-only)                        │
│ Volume: docker.sock (optional, :ro)                      │
└──────────────────────────────────────────────────────────┘
```

**How it works:**

1. **Parser** — reads `/proc/net/{tcp,tcp6,udp,udp6}` via `PROC_PATH` and decodes hex addresses, ports, and connection states
2. **Mapper** — walks `/proc/[pid]/fd/*` symlinks, matches `socket:[inode]` patterns to resolve the owning process. Falls back through status → comm → cmdline → exe for name resolution.
3. **Cache** — `singleflight.Group` coalesces concurrent requests; serves stale data if a fresh fetch fails
4. **Docker Monitor** (optional) — queries Docker Engine API via unix socket for container metadata, maps containers to sockets by PID (host-network) or port binding (bridge-network)
5. **API** — `POST /api/auth` for token validation, `GET /api/sockets` for socket data with process info and container enrichment, `GET /api/containers` for raw container data
6. **Frontend** — Pinia store with `shallowRef` for efficient updates, composable for polling with visibility API awareness and freeze toggle, regex search across all fields, CSV/JSON export, container column and filter

## Deployment

The project ships as a single Docker container with the Vue.js frontend embedded in the Go binary via `//go:embed`:

| Service | Base Image | Notes |
|---------|-----------|-------|
| `ports` | `distroless/static-debian12` | Runs as non-root (UID 65534) with minimal capabilities. Host `/proc` mounted read-only. Published to `ghcr.io/macedot/ports` |

### Security

- **Authentication**: Set the `ADMIN_TOKEN` environment variable to enable token-based authentication. When set, a login form is shown and all API requests require the token via `Authorization: Bearer` header. Token is stored in `sessionStorage` (cleared on tab close). When unset, the application is freely accessible.
- **Hardened container**: No `network_mode: host`. Host `/proc` mounted read-only at `/host-proc`. Read-only root filesystem. `pid: host` is required for process name resolution (the kernel only exposes socket→PID mappings via `/proc/[pid]/fd/` readlink). Default user is `65534:65534` (nobody) — process names for root-owned host processes will be empty unless you add `user: "0:0"` (less secure). `SYS_PTRACE` capability included for fd symlink access. No-new-privileges enforced. Resource limits: 128MB memory, 0.5 CPU.
- **Security headers**: `X-Content-Type-Options: nosniff` and `X-Frame-Options: DENY` are set on all responses.
- **Non-root container**: The process runs as UID 65534 (nobody) inside the container.
- **TLS**: The server listens on plain HTTP. For production, run behind a TLS-terminating reverse proxy (nginx, Caddy, Traefik).

## CI/CD

GitHub Actions builds and pushes Docker images to GHCR on every published release (prereleases excluded). Images are tagged with both the release version and `latest`.

## License

This project is licensed under the [GNU Affero General Public License v3.0](LICENSE).
