# proxy-tui — Next Steps

## Completed

- [x] **Fix data race in `MapRuleStore`** — Added `sync.RWMutex`, `All()` now returns a copy.
- [x] **Add `THIRD_PARTY_NOTICES`** — Full license texts for all 10 dependencies, bundled in release.
- [x] **Write `README.md`** — Project overview, usage, keybindings, architecture, configuration.

---

## Phase 1 — Tests

All packages now have comprehensive test coverage with `-race` enabled.

### 1.1 `model` package
- [x] `model/mapping_test.go` — Match (exact, glob, regex, trailing slash, disabled), Apply, CRUD, FindMatch with priority, concurrency, DetectContentType
- [x] `model/flow_test.go` — IsComplete, Duration, FlowEventType iota
- [x] `model/filter_test.go` — FilterAll/Whitelist/Custom, search query, host patterns, methods, status codes, combined

### 1.2 `proxy` package
- [x] `proxy/proxy_test.go` — SSLProxyList CRUD + Match, parseJSONCResponseFile, buildResponseFromJSON, parseOldHTTPFormat, DefaultConfig
- [x] `proxy/flow_test.go` — Add/Get/Update/Clear/Count, Filter, eviction, Subscribe/Unsubscribe, non-blocking emit, AddWithID, UpdateFromRemote, Last, concurrency

### 1.3 `config` package
- [x] `config/config_test.go` — Whitelist round-trip, migration (old JSON, plain text), Add/Remove/Edit/Toggle/SetEnabled/Clear, MapLocal CRUD, MapRemote CRUD, escapeJSON, JSONC slash-in-pattern

### 1.4 `ipc` package
- [x] `ipc/wire_test.go` — FlowToWire/WireToFlow round-trip, MarshalMessage/UnmarshalMessage for all message types
- [x] `ipc/ipc_test.go` — Server+Client hello handshake, flow sync, real-time streaming, config reload, client disconnect, server stop, IsInstanceRunning

### 1.5 `viewmodel` package
- [x] `viewmodel/viewmodel_test.go` — Initial state, FormatFlowSummary (complete/nil/pending/error), FormatFlowDetail (tunneled/response), formatDuration, getBaseDomain, SelectFlow, SetSearchQuery, SetFilterType, ClearFlows, whitelist CRUD, IsSecondary

### 1.6 `pkg/ca` package
- [x] `ca/ca_test.go` — Generate, Load (missing/existing), GenerateCert (host/IP/cached), CertPath, Fingerprint, concurrent GenerateCert

### 1.7 CI
- [x] Added `Makefile` with `build`, `test`, `lint`, `clean` targets
- [x] Added `.github/workflows/ci.yml` — runs vet, test (`-race`), and build on push/PR

---

## Phase 2 — Code health (bugs, duplication, SRP)

### 2.1 Extract duplicated utilities into `internal/util`
- [x] `stripJSONComments` → `util.StripJSONComments` (removed from `proxy/proxy.go` and `config/config.go`)
- [x] `matchGlob` / `matchGlobPattern` → `util.MatchGlob` (removed from `proxy/proxy.go`, `model/filter.go`, `model/mapping.go`)
- [x] `matchHostPattern` / `matchPattern` → `util.MatchHostPattern` (removed from `proxy/proxy.go`, `model/filter.go`)
- [x] Deduplicated `config/maplocal.go` + `config/mapremote.go` with generic `ConfigStore[E]` in `config/store.go`
- [x] `isJSON` / `isJSONContent` → `util.IsJSON` (removed from `viewmodel/format.go` and `ui/app.go`)

Single source of truth for each. Update all call sites.

### 2.1b Split `viewmodel/viewmodel.go` by SRP (706 → 4 files)
- [x] Extract `viewmodel/format.go` — FormatFlowSummary, FormatFlowDetail, formatBody, isJSON, getBaseDomain (186 lines)
- [x] Extract `viewmodel/whitelist.go` — whitelist CRUD methods (134 lines)
- [x] Extract `viewmodel/maprules.go` — map local + remote rule CRUD (158 lines)
- [x] Slim `viewmodel.go` to core only (244 lines)

### 2.2 Split `proxy/proxy.go` (773 lines after Phase 2 extraction, multiple concerns)
- [x] Extract `SSLProxyList` + host pattern matching → `proxy/ssllist.go`
- [x] Extract `serveLocalFile`, `parseJSONCResponseFile`, `buildResponseFromJSON`, `parseOldHTTPFormat` → `proxy/serve_local.go`
- [x] Extract `fetchRemote` → `proxy/serve_remote.go`
- [x] Keep `proxy.go` focused on: `Proxy` struct, lifecycle, `handleRequest`, `handleResponse` (774 → 329 lines)

### 2.3 Extract JSONC response builder from `ui/app.go`
- [x] Extracted `writeJSONCMapping` method from `quickMapLocal()` and `createMapLocalWithPattern()`
- [x] Both methods now call shared helper with different comment lines

### 2.4 Replace boolean flags with state enum in `ui/App`
- [x] Replace the 8 `bool` fields (`searching`, `addingWhitelist`, `addingMapLocalInput`, etc.) with a single `activePopup PopupState` enum
- [x] Simplify `isPopupOpen()` and `getCurrentPopupPrimitive()` to a switch on the enum
- [x] Prevents impossible states (two popups "open" at once)

---

## Phase 3 — Design improvements (DIP, ISP, testability)

### 3.1 Inject config persistence into ViewModel
- [x] Defined `ConfigPersistence` interface in `viewmodel` package (15 methods: whitelist + map-local + map-remote)
- [x] `config.DefaultPersistence` struct implements it by delegating to package-level functions
- [x] `ViewModel.New()` takes `ConfigPersistence` as a parameter — all direct `config.*` calls replaced with `vm.config.*`
- [x] Enables unit testing ViewModel without filesystem

### 3.2 Reduce `FlowSource` surface area (ISP)
- [x] Split `FlowSource` into `FlowProvider` (FlowStore, Port, BindAddress) + full `FlowSource` (embeds FlowProvider + Events, SSLProxyList, MapRules)
- [x] IPC Server now accepts the narrower `FlowProvider` interface
- [x] ViewModel still uses the full `FlowSource`

### 3.2b Make MapRuleStore.FindMatch open/closed with Priority
- [x] Added `Priority` field to `MapRule` (higher = checked first)
- [x] Default priorities: `PriorityMapLocal=100`, `PriorityMapRemote=50`
- [x] `FindMatch` now picks the highest-priority match in a single pass (no more hardcoded two-pass by type)

### 3.3 Deduplicate CLI handlers in `main.go`
- [x] Extracted generic `listConfig[E]` and `removeConfig[E]` helper functions
- [x] Replaced 6 repetitive handler blocks with concise calls to these helpers

---

## Phase 4 — Features (from SPECIFICATION.md roadmap)

### 4.1 Replay requests
- [x] Added `ReplayFlow` method in `viewmodel/replay.go` — sends request through proxy via HTTP transport
- [x] Keybinding `.` in both list and detail contexts
- [x] Status bar feedback: "Replaying request..." → "Request replayed" / error

### 4.2 Export / Import flows
- [x] `FormatCURL` in `viewmodel/export.go` — generates cURL command with headers, body, method
- [x] `FormatHAR` in `viewmodel/export.go` — generates HAR 1.2 JSON for one or more flows
- [x] `ParseHAR` in `viewmodel/export.go` — parses HAR JSON back to `[]*model.Flow`
- [x] Keybinding `x` → copy cURL to clipboard (wl-copy/pbcopy/xclip/xsel)
- [x] Keybinding `e` → export selected flow as HAR, `E` → export all filtered flows as HAR
- [x] Keybinding `i` → import HAR file with file picker (`ui/filepicker.go`)
- [x] File picker: real-time directory listing that filters as you type, Tab completion, `~` expansion
- [x] `ImportFlows` uses `AddDirect` to bypass pause state
- [x] Round-trip tests in `viewmodel/export_test.go` (export → import → verify)

### 4.3 Alerts
- [x] `AlertRule` model (`model/alert.go`) — `status_code` (match by class, e.g. 5xx) and `latency` (ms threshold)
- [x] Config persistence (`config/alerts.go`) — saves to `alerts.json`, defaults to 5xx enabled + latency 5s disabled
- [x] ViewModel integration (`viewmodel/alerts.go`) — `CheckAlerts`, `ToggleAlertRule`, `GetAlertRules`, `SetAlertRules`
- [x] Visual indicator `!` prefix on status column in requests panel when alert matches
- [x] Alert manager UI (`ui/alerts.go`) — keybinding `a`, toggle rules with Enter/Space

### 4.4 Star flows
- [x] `ToggleStar`, `StarFlows`, `IsStarred`, `StarredCount` methods on ViewModel
- [x] `FilterStarred` filter type in `model/filter.go` — only shows starred flows
- [x] `StarredIDs` map on `FilterState`, checked in `Match`
- [x] Keybinding `s` → toggle star on selected flow, `S` → star all listed flows
- [x] Keybinding `3` → filter to show only starred flows
- [x] Yellow `*` indicator in dedicated column at end of each row
- [x] Filter bar updated: `1:All  2:Whitelist  3:Starred  /:Custom`

### 4.5 Pause / Resume
- [x] `SetPaused` / `IsPaused` on FlowStore (atomic flag)
- [x] `Add` and `Update` are no-ops when paused — no flows recorded
- [x] Map-local and map-remote rules bypassed when paused — pure passthrough
- [x] `AddDirect` method bypasses pause (used by HAR import)
- [x] Keybinding `p` → toggle pause/resume
- [x] Red `PAUSED` label in Requests panel title bar

### 4.6 Bug fixes
- [x] Fixed clipboard on Wayland — reordered to try `wl-copy` first
- [x] Fixed data race in `FlowStore.emit()` — hold `subMu` lock while sending
- [x] Fixed HAR import setting host with port — use `u.Hostname()` instead of `u.Host`

---

## Phase 5 — Distribution

### 5.1 Release binaries
- [ ] GitHub Actions CI for cross-platform builds
- [ ] Attach `LICENSE` + `THIRD_PARTY_NOTICES` to each release archive
- [ ] Homebrew formula or tap

### 5.2 Breakpoints
- [ ] Pause a request before forwarding, allow editing headers/body
- [ ] Pause a response before returning to client

### 5.3 WebSocket support
- [ ] Detect WebSocket upgrade
- [ ] Display frames in detail panel
