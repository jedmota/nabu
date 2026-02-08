# proxy-tui

HTTP/HTTPS debugging proxy with a terminal UI. Intercepts, inspects, and manipulates HTTP traffic via MITM. Supports URL whitelisting, mapping requests to local files, and redirecting requests to different servers.

## Quick Start

```bash
# Build
go build -o proxy-tui ./cmd/proxy-tui

# Run (default port 9090, all interfaces)
./proxy-tui
./proxy-tui --port 8080 --bind 127.0.0.1

# Run without TUI (headless / scripting)
./proxy-tui --headless

# Show CA certificate info
./proxy-tui --show-ca

# Install CA to system trust store
./scripts/install-ca.sh
```

Configure clients to use the proxy:

```bash
export HTTP_PROXY=http://localhost:9090
export HTTPS_PROXY=http://localhost:9090
```

## CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | `9090` | Proxy listening port |
| `--bind` | `0.0.0.0` | Bind address |
| `--verbose` | `false` | Verbose logging |
| `--headless` | `false` | Run without TUI |
| `--show-ca` | `false` | Print CA cert path and fingerprint, then exit |

## Whitelist Management

Whitelist patterns filter which requests are displayed in the UI. Supports glob (`*.example.com`) and regex (`^api\.example\.com$`) patterns.

```bash
# Add patterns (proxy starts normally after adding)
proxy-tui --whitelist "*.example.com"
proxy-tui --whitelist "*.api.io" --whitelist "*.cdn.io"

# List all patterns with their IDs
proxy-tui --list-whitelist
# Output:
# 1: *.example.com (enabled)
# 2: *.api.io (enabled)
# 3: *.cdn.io (disabled)

# Remove by ID
proxy-tui --rm-whitelist 2
```

## Map Local (Mock Responses)

Serve local files instead of forwarding requests to the origin server. Useful for mocking APIs.

```bash
# Add rules (proxy starts normally after adding)
proxy-tui --map-local "https://api.example.com/users=>/tmp/users.json"
proxy-tui --map-local "*/api/data*=>/home/user/mock.json"

# List rules
proxy-tui --list-map-local
# Output:
# 1: https://api.example.com/users => /tmp/users.json (enabled)

# Remove by ID
proxy-tui --rm-map-local 1
```

Mock files can be plain JSON or JSONC with response metadata:

```jsonc
{
  "status": 200,
  "headers": {
    "Content-Type": "application/json"
  },
  "body": {
    "users": [{"id": 1, "name": "Mock User"}]
  }
}
```

## Map Remote (Redirect Requests)

Transparently redirect requests to a different server (no HTTP redirect issued to the client).

```bash
# Add rules (proxy starts normally after adding)
proxy-tui --map-remote "https://api.prod.com/*=>http://localhost:3000"

# List rules
proxy-tui --list-map-remote
# Output:
# 1: https://api.prod.com/* => http://localhost:3000 (enabled)

# Remove by ID
proxy-tui --rm-map-remote 1
```

A request to `https://api.prod.com/users?page=1` gets fetched from `http://localhost:3000/users?page=1`. The client sees the response as if it came from the original host.

## Combining Flags

Add flags are repeatable and can be combined. The proxy starts normally after processing them:

```bash
proxy-tui --port 8080 \
  --whitelist "*.example.com" \
  --map-local "*/mock-endpoint=>/tmp/mock.json" \
  --map-remote "https://api.prod.com/*=>http://localhost:3000"
```

List and remove flags are fire-and-exit (they print output or modify config, then the process exits):

```bash
proxy-tui --list-whitelist    # prints and exits
proxy-tui --rm-map-local 2    # removes and exits
```

## Pattern Syntax

Patterns are used across whitelist, map-local, and map-remote rules:

| Pattern | Matches |
|---------|---------|
| `*.example.com` | Any subdomain of example.com |
| `api.*` | Any TLD for "api" |
| `*google*` | Any URL containing "google" |
| `example.com` | Exact match |
| `^(api\|www)\.example\.com$` | Regex (if starts with `^` or `(`) |

## Config Files

All configuration is stored in `~/.proxy-tui/`:

| File | Contents |
|------|----------|
| `ca.crt` / `ca.key` | Auto-generated CA certificate and key |
| `whitelist.jsonc` | Whitelist patterns |
| `maplocal.jsonc` | Map-local rules |
| `mapremote.jsonc` | Map-remote rules |

## Multi-Instance

Running `proxy-tui` on a port that already has an instance creates a secondary (view-only) connection via IPC:

```bash
# Terminal 1: primary instance
proxy-tui --port 9090

# Terminal 2: secondary instance (connects to existing)
proxy-tui --port 9090
```

## TUI Keybindings

| Key | Action |
|-----|--------|
| `q` | Quit |
| `Tab` | Toggle focus between list and detail |
| `j`/`k` | Navigate up/down |
| `1` | Show all requests |
| `2` | Show whitelist matches only |
| `/` | Search/filter by URL |
| `c` | Clear all captured flows |
| `w` | Quick-add selected host to whitelist |
| `W` | Open whitelist manager |
| `l` | Quick-create local mock from selected request |
| `L` | Open map-local manager |
| `r` | Quick-add map-remote rule |
| `R` | Open map-remote manager |
| `T` | Toggle raw/pretty JSON in detail view |
| `?` | Show help overlay |
