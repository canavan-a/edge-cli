# edge-cli

A command-line tool for inspecting and managing services on a ClearBlade Edge device. Designed to be run directly on the edge device over SSH.

## Requirements

- A Linux device running **ClearBlade Edge**
- A ClearBlade developer account with access to the system deployed on that edge
- Network access to the edge HTTP server (default port `9000`) — satisfied automatically when SSH'd into the device

## Installation

Run the following on the edge device:

```sh
curl -fsSL https://github.com/canavan-a/edge-cli/releases/latest/download/install.sh | sh
```

Or install a specific version:

```sh
VERSION=v1.0.0 curl -fsSL https://github.com/canavan-a/edge-cli/releases/latest/download/install.sh | sh
```

Supported architectures: `amd64`, `arm64`, `armv7`

## Getting Started

### 1. Log in

```sh
edge-cli auth login --email you@example.com
```

This prompts for your password and saves a token to `~/.edge-cli/config.yaml`. You only need to do this once per device.

### 2. Check auth status

```sh
edge-cli auth status
```

### 3. Log out

```sh
edge-cli auth logout
```

---

## Services

### List all services

```sh
edge-cli services list
```

```
NAME                 INSTANCES   ENGINE   TIMEOUT   LOGGING
my-long-running      3           v8       never     on
data-processor       0           v8       30s       off
sensor-ingest        1           duk      60s       on
```

Show only services with at least one running instance:

```sh
edge-cli services list --running
```

### Inspect a service

```sh
edge-cli services show <service-name>
```

Displays service configuration and all currently running instances with uptime and memory usage.

### Start a service

```sh
edge-cli services start <service-name>
```

Pass optional params as JSON:

```sh
edge-cli services start <service-name> --params '{"key":"value"}'
```

### Stop a service

Stop all running instances:

```sh
edge-cli services stop <service-name>
```

Stop a specific instance:

```sh
edge-cli services stop <service-name> --instance <instance-id>
```

### View logs

```sh
edge-cli services logs <service-name>
```

Follow logs as new runs come in:

```sh
edge-cli services logs <service-name> --follow
```

---

## Global Flags

These flags work on every command and override saved config:

| Flag | Env var | Description |
|---|---|---|
| `--token` | `CB_DEV_TOKEN` | ClearBlade dev token |
| `--system-key` | `CB_SYSTEM_KEY` | System key |
| `--url` | `CB_EDGE_URL` | Edge URL (default: `http://localhost:9000`) |
| `--edge-config` | | Path to edge TOML config for auto-detecting system key |

---

## Upgrading

```sh
edge-cli upgrade
```

## Version

```sh
edge-cli version
```
