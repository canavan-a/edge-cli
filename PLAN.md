# Edge CLI — Implementation Plan

## Overview

A Go CLI tool (`edge-cli`) that runs directly on a Linux device hosting ClearBlade Edge. It talks to the Edge's local HTTP server (default port `:9000`) using a developer token, giving operators a fast terminal interface to inspect and manage services without needing the web console or SSH tunnelling tricks.

---

## Background: How the Edge Exposes Services

The Edge runs the same HTTP server as the platform, listening on `:9000` by default (configurable via `--edge-listen-port`).

**Relevant endpoints (all require `ClearBlade-DevToken` header — `router.DevOnly`):**

| Endpoint | Purpose |
|---|---|
| `GET /codeadmin/v/2/{systemKey}` | List all deployed services (metadata) |
| `GET /codeadmin/v/3/running/{systemKey}` | List currently running service instances |
| `GET /codeadmin/v/2/logs/{systemKey}/{name}` | Fetch stored logs for a service |
| `GET /codeadmin/v/2/logs/{systemKey}/{name}/{logID}` | Fetch logs for one specific run |
| `GET /codeadmin/v/4/heapstats/{systemKey}/{serviceId}` | Heap stats for a running instance |

**Key data structures (from the platform source):**

- `RunningServiceInfo` — `CodeName`, `SystemKey`, `Started` (unix ts), `Node`, `IsTerminating`, `EngineType`, `HeapStatistics`
- `DBCodeMeta` — `Name`, `Version`, `ExecutionTimeout`, `Concurrency`, `LoggingEnabled`, `LogLevel`, `AutoScale`, `RunOnEdge`

---

## Auth & Config

### `auth login`

**Command:** `edge-cli auth login [--email <email>] [--url <edge-url>]`

Prompts for email and password (password input is hidden), hits the Edge's auth endpoint, and saves the returned dev token to the config file:

```
$ edge-cli auth login --email dev@example.com
Password: ••••••••
Logged in as dev@example.com. Token saved to ~/.edge-cli/config.yaml
```

**Endpoint:** `POST /api/v/1/authenticate`
```json
{ "email": "dev@example.com", "password": "..." }
```
**Response fields saved to config:** `dev_token`, plus `email` for display.

### `auth logout`

**Command:** `edge-cli auth logout`

Clears the saved token and email from `~/.edge-cli/config.yaml`.

```
$ edge-cli auth logout
Logged out.
```

### `auth status`

**Command:** `edge-cli auth status`

Shows who is currently authenticated and whether the token appears valid.

```
$ edge-cli auth status
Logged in as dev@example.com
Edge URL: http://localhost:9000
```

---

### Config resolution order

Once logged in, commands resolve credentials automatically in this priority order:

1. CLI flags (`--token`, `--system-key`, `--url`)
2. Environment variables (`CB_DEV_TOKEN`, `CB_SYSTEM_KEY`, `CB_EDGE_URL`)
3. CLI config file (`~/.edge-cli/config.yaml`) — written by `auth login`
4. Edge TOML config file — the edge's own config (passed via `-config` at startup) contains `ParentSystemKey`, which the CLI can read to determine the system key automatically. Default search paths: `/etc/clearblade/config.toml`, `./config.toml`, or overridden with `--edge-config <path>`.

---

## Tech Stack

- **Language:** Go (matches the rest of the platform codebase)
- **CLI framework:** `cobra` — standard, extensible, good help generation
- **HTTP client:** stdlib `net/http`
- **Output:** `tablewriter` or plain `text/tabwriter` for table output; `--json` flag for machine-readable output
- **Config:** `viper` (pairs naturally with cobra)

---

## Project Structure

```
edge-cli/
├── main.go
├── cmd/
│   ├── root.go          # root command, persistent flags, config loading
│   ├── auth.go          # `auth` subcommand group
│   ├── auth_login.go    # `auth login`
│   ├── auth_logout.go   # `auth logout`
│   ├── auth_status.go   # `auth status`
│   ├── services.go      # `services` subcommand group
│   ├── services_list.go # `services list`
│   └── services_show.go # `services show <name>`
├── client/
│   └── edge.go          # HTTP client wrapper (auth headers, base URL)
├── models/
│   └── service.go       # Go types matching API responses
└── config/
    └── config.go        # viper config loading, token read/write helpers
```

---

## Phase 1 — `services list`

**Command:** `edge-cli services list [--system <key>]`

**What it does:**
1. `GET /codeadmin/v/2/{systemKey}` — fetch all deployed service metadata
2. `GET /codeadmin/v/3/running/{systemKey}` — fetch running instances
3. Join on service name; count instances per service name from the running map
4. Render a table:

```
NAME                 INSTANCES   ENGINE   TIMEOUT   LOGGING
my-long-running      3           v8       never     on
data-processor       0           v8       30s       off
sensor-ingest        1           duk      60s       on
```

**Flags:**
- `--running` — show only services with at least one running instance
- `--json` — output raw JSON

---

## Phase 2 — `services show <name>`

**Command:** `edge-cli services show <service-name> [--system <key>]`

**What it does:**
1. Fetch service metadata from `/codeadmin/v/2/{systemKey}/{name}`
2. Fetch all running instances from `/codeadmin/v/3/running/{systemKey}`, filter by name
3. For each running instance, optionally fetch heap stats from `/codeadmin/v/4/heapstats/{systemKey}/{serviceId}`
4. Display a detail view:

```
Service: my-long-running
  System Key:   abc123
  Engine:       v8
  Timeout:      never (long-running)
  Concurrency:  5
  Logging:      enabled (level: info, TTL: 60 min)
  Auto-scale:   off
  Run on edge:  yes

Running Instances (3):
  ID                                   STARTED              TERMINATING   HEAP USED
  a1b2c3d4-...                         2026-06-12 14:02:01  no            12.4 MB
  e5f6a7b8-...                         2026-06-12 13:58:44  no            11.1 MB
  c9d0e1f2-...                         2026-06-12 13:55:12  yes           9.8 MB
```

---

## Phase 3 — `services logs <name>` (polling tail)

**Command:** `edge-cli services logs <name> [--follow] [--system <key>]`

**What it does (non-follow):**
- `GET /codeadmin/v/2/logs/{systemKey}/{name}` — fetch recent log runs
- Display the most recent N log entries

**What it does (`--follow` mode):**
- The Edge does not expose a streaming/websocket log endpoint, so implement polling:
  1. Fetch recent runs, record the latest `logID` seen
  2. On each poll interval (default 2s), re-fetch and print any new log entries not yet shown
  3. Track by `logID` + `time` cursor to avoid re-printing
- This is essentially `tail -f` via polling — acceptable for an ops tool

**Flags:**
- `--follow` / `-f` — poll continuously (Ctrl-C to stop)
- `--interval 2s` — polling interval when using `--follow`
- `--lines 50` — number of historical lines to show on first fetch

---

## Phase 4 — Additional Commands (Future)

These are out of scope for now but the cobra structure makes them easy to add:

- `services stop <name> <instanceId>` — `DELETE /codeadmin/v/3/running/{systemKey}`
- `services log-level <name> <level>` — `POST /codeadmin/v/2/logs/{systemKey}/{name}`
- `config set` / `config show` — manage the local config file
- `systems list` — list available system keys on this edge
- `adaptors list` — adaptor management (separate from services)

---

## Build & Distribution

- Single static binary (CGO_ENABLED=0) for easy `scp` to edge devices
- Cross-compile targets: `linux/amd64`, `linux/arm64`, `linux/arm` (common edge hardware)
- Makefile targets: `build`, `build-all`, `install`

---

## Open Questions

None — all questions resolved.
