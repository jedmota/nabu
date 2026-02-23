# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

proxy-tui is a terminal-based HTTP/HTTPS debugging proxy (inspired by Proxyman). It intercepts traffic via MITM, displays requests/responses in a TUI, and supports traffic manipulation (map-local, map-remote), HAR import/export, request replay, and multi-instance viewing via IPC.

## Build & Development Commands

```bash
make test          # go test -race -count=1 ./...
make lint          # go vet ./...
make build         # cross-platform builds via scripts/release.sh → bin/
make run           # run local binary (bin/proxy-linux-arm64)

# Run a single test
go test -race -run TestName ./internal/proxy/

# Build for local dev
go build -o proxy-tui ./cmd/proxy-tui
```

## Architecture

**MVVM pattern with channel-based events:**

- **cmd/proxy-tui/main.go** — Entry point. Parses flags, detects primary vs secondary instance, launches proxy + TUI or IPC client + TUI.
- **internal/proxy/** — Core proxy (goproxy wrapper). FlowStore holds captured flows, emits FlowEvents via channels. Handles MITM, map-local, map-remote interception.
- **internal/viewmodel/** — ViewModel consumes FlowEvents, applies filters, syncs state to UI. Central orchestrator between proxy and TUI.
- **internal/ui/** — TUI built on tview/tcell. Two-panel layout (request list + detail). Vim-style navigation (j/k/g/G). Keybinding system with whichkey help overlay.
- **internal/model/** — Data types: Flow (request/response pair), FilterState, MapRule, AlertRule. Pattern matching (glob/regex).
- **internal/config/** — JSONC config files in `~/.proxy-tui/` (whitelist, maplocal, mapremote, alerts). Handles format migration.
- **internal/ipc/** — Unix domain socket IPC. Primary instance runs server; secondary instances connect as read-only viewers.
- **pkg/ca/** — Certificate authority generation and caching for MITM. Auto-generates CA on first run in `~/.proxy-tui/`.
- **internal/debug/** — File-based debug logging.

**Key interfaces:**
- `FlowSource` (proxy package) — abstracts flow access for both primary (direct proxy) and secondary (IPC client) modes.
- `ConfigPersistence` (config package) — abstracts config storage for testability.

**Concurrency model:** sync.RWMutex in ViewModel/FlowStore/MapRuleStore, atomic operations for pause state, buffered event channels (cap 1000) with non-blocking emission.

## Config Directory

`~/.proxy-tui/` contains: `ca.crt`, `ca.key`, `whitelist.jsonc`, `maplocal.jsonc`, `mapremote.jsonc`, `alerts.json`, and IPC sockets (`proxy-PORT.sock`).
# Agent rules

Read `~/.config/opencode/agents.md`
