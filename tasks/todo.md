# proxy-tui — Next Steps

## Completed

- [x] **Fix data race in `MapRuleStore`** — Added `sync.RWMutex`, `All()` now returns a copy.
- [x] **Add `THIRD_PARTY_NOTICES`** — Full license texts for all 10 dependencies, bundled in release.
- [x] **Write `README.md`** — Project overview, usage, keybindings, architecture, configuration.

---

## Phase 1 — Tests (zero coverage today, prerequisite for safe refactoring)

There are currently **no test files** in the project. Tests should come first — they lock in existing behavior so the refactoring in Phase 2 can be done with confidence.

### 1.1 `model` package (pure logic, no I/O — easiest to test)
- [ ] `model/mapping_test.go`
  - `MapRule.Match` — exact match, glob `*`, glob `?`, regex, trailing slash, disabled rule
  - `MapRule.Apply` — regex replacement, glob prefix replacement, simple replacement
  - `MapRuleStore.Add/Remove/Toggle/GetByID/Update` — basic CRUD
  - `MapRuleStore.FindMatch` — local rules checked before remote, first match wins, disabled rules skipped
  - `MapRuleStore` concurrency — parallel `FindMatch` + `Add`/`Remove` with `-race`
  - `DetectContentType` — common extensions, unknown fallback
- [ ] `model/flow_test.go`
  - `Flow.IsComplete`, `Flow.Duration`
- [ ] `model/filter_test.go`
  - `FilterState` matching logic (FilterAll, FilterWhitelist, FilterCustom)
  - `HostPattern` matching with enabled/disabled state

### 1.2 `proxy` package (needs HTTP test harness)
- [ ] `proxy/ssllist_test.go` (can be written now against existing code)
  - `SSLProxyList.Add/Remove/Clear/Patterns/Match`
  - `matchHostPattern` — exact, wildcard `*`, `*.example.com`, regex, port stripping
  - `matchGlob` — edge cases
- [ ] `proxy/flow_test.go`
  - `FlowStore.Add/Get/Update/Clear/Count`
  - Flow eviction when `maxFlows` exceeded
  - Event emission on add/update
  - Subscriber management (`Subscribe`, `Unsubscribe`)
  - Concurrency: parallel `Add` + `Get` with `-race`
- [ ] `proxy/proxy_test.go` (integration-level, uses `httptest`)
  - Start proxy, send HTTP request, verify flow captured
  - Map-local: request returns local file content
  - Map-remote: request forwarded to mock backend
  - JSONC response file parsing
  - Old HTTP format parsing
  - Conditional MITM: whitelisted host gets intercepted, non-whitelisted tunnels

### 1.3 `config` package (needs temp dir)
- [ ] `config/config_test.go`
  - `SaveWhitelist` → `LoadWhitelist` round-trip
  - JSONC comment stripping
  - Migration from old formats (plain text, old JSON)
  - `AddToWhitelist`, `RemoveFromWhitelist`, `ToggleWhitelistPattern`
- [ ] `config/maplocal_test.go`
  - `SaveMapLocal` → `LoadMapLocal` round-trip
  - `AddMapLocalEntry`, `RemoveMapLocalEntry`, `ToggleMapLocalEntry`
- [ ] `config/mapremote_test.go`
  - Same pattern as maplocal

### 1.4 `ipc` package (needs socket pair)
- [ ] `ipc/wire_test.go`
  - Marshal/unmarshal round-trip for each message type (`hello`, `sync`, `flow_event`, `config_reload`)
- [ ] `ipc/integration_test.go`
  - Start `Server` + `Client` on a temp socket
  - Verify hello handshake, flow sync, real-time event streaming
  - Client disconnect detection

### 1.5 `viewmodel` package (needs mock FlowSource)
- [ ] Requires `ConfigStore` interface from Phase 3.1 for clean mocking, OR use a temp config dir
- [ ] `viewmodel/viewmodel_test.go`
  - Filter switching (All → Whitelist → Custom)
  - Search query filtering
  - Whitelist add/remove/toggle/edit → verify SSLProxyList and config updated
  - Map rule add/remove/toggle
  - `ReloadConfig` — verify state is replaced, not appended
  - `FormatFlowSummary` — pending, complete, error states
  - `FormatFlowDetail` — tunneled, request-only, request+response

### 1.6 `pkg/ca` package
- [ ] `ca/ca_test.go`
  - `Load` — generates new CA if none exists, reuses existing
  - `GenerateCert` — returns valid cert for host, caches result
  - `CertPath`, `Fingerprint` — non-empty, stable

### 1.7 CI
- [ ] Add `go test -race ./...` to a GitHub Actions workflow or a `Makefile` target
- [ ] Run on every push / PR

---

## Phase 2 — Code health (bugs, duplication, SRP)

### 2.1 Extract duplicated utilities into `internal/util`
- [ ] `stripJSONComments` — duplicated in `proxy/proxy.go` and `config/config.go`
- [ ] `matchGlob` / `matchGlobPattern` — duplicated in `proxy/proxy.go` and `model/mapping.go`
- [ ] `isJSON` / `isJSONContent` — duplicated in `viewmodel/viewmodel.go` and `ui/app.go`

Single source of truth for each. Update all call sites.

### 2.2 Split `proxy/proxy.go` (867 lines, multiple concerns)
- [ ] Extract `SSLProxyList` + host pattern matching → `proxy/ssllist.go`
- [ ] Extract `serveLocalFile`, `parseJSONCResponseFile`, `buildResponseFromJSON`, `parseOldHTTPFormat` → `proxy/maplocal.go`
- [ ] Extract `fetchRemote` → `proxy/mapremote.go`
- [ ] Keep `proxy.go` focused on: `Proxy` struct, lifecycle, `handleRequest`, `handleResponse`

### 2.3 Extract JSONC response builder from `ui/app.go`
- [ ] `quickMapLocal()` and `createMapLocalWithPattern()` share ~80 lines of identical JSONC-building code (lines 1033-1109 vs 1144-1209)
- [ ] Extract into a shared function, e.g. `buildJSONCResponse(flow *model.Flow, pattern string) ([]byte, error)`

### 2.4 Replace boolean flags with state enum in `ui/App`
- [ ] Replace the 8 `bool` fields (`searching`, `addingWhitelist`, `addingMapLocalInput`, etc.) with a single `activePopup` enum
- [ ] Simplify `isPopupOpen()` and `getCurrentPopupPrimitive()` to a switch on the enum
- [ ] Prevents impossible states (two popups "open" at once)

---

## Phase 3 — Design improvements (DIP, ISP, testability)

### 3.1 Inject config persistence into ViewModel
- [ ] Define a `ConfigStore` interface:
  ```go
  type ConfigStore interface {
      LoadWhitelist() ([]WhitelistPattern, error)
      SaveWhitelist([]WhitelistPattern) error
      LoadMapLocal() ([]MapLocalEntry, error)
      AddMapLocalEntry(MapLocalEntry) error
      // ... etc
  }
  ```
- [ ] `config` package implements it
- [ ] `ViewModel.New()` takes `ConfigStore` as a parameter instead of calling `config.*` directly
- [ ] Enables unit testing ViewModel without filesystem

### 3.2 Reduce `FlowSource` surface area
- [ ] Current interface leaks concrete types (`*FlowStore`, `*SSLProxyList`, `*MapRuleStore`), letting consumers mutate internals freely
- [ ] Option A: Add methods to `FlowSource` (e.g. `AddSSLPattern(string)`, `RemoveSSLPattern(string)`) and stop exposing stores
- [ ] Option B: Define read-only sub-interfaces for the stores
- [ ] Evaluate impact on `ViewModel` and `Adapter` before choosing

### 3.3 Deduplicate CLI handlers in `main.go`
- [ ] The list/remove handlers for whitelist, map-local, map-remote (lines 70-191) repeat the same pattern 3 times
- [ ] Extract a generic `listConfig` and `removeConfig` helper

---

## Phase 4 — Features (from SPECIFICATION.md roadmap)

### 4.1 Replay requests
- [ ] Add a "replay" action to the TUI (keybinding `p`)
- [ ] Re-send the selected flow's request through the proxy
- [ ] Show the new response as a separate flow

### 4.2 Export flows
- [ ] Export selected flow as cURL command (copy to clipboard)
- [ ] Export selected flow as HAR format (save to file)
- [ ] Export all flows as HAR

### 4.3 Alerts
- [ ] Configurable alert rules (5xx status, latency threshold)
- [ ] Visual indicator in requests panel when an alert triggers
- [ ] Optional desktop notification

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
