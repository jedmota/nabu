# Lessons Learned

## Thread safety must match access patterns
`FlowStore` had a mutex from the start, but `MapRuleStore` did not — even though both are written from the UI goroutine and read from proxy handler goroutines. When adding a new concurrent data structure, check who reads and who writes before deciding on locking.

## Duplication drifts silently
`matchGlob` in `proxy.go` and `matchGlobPattern` in `model/mapping.go` implement the same logic with slightly different names and minor differences (e.g. `?` wildcard support). Same for `stripJSONComments` in two packages. Duplication that starts as "just a quick copy" becomes a maintenance liability as the copies evolve independently.

## Generics work well for config store deduplication
`maplocal.go` (182 lines) and `mapremote.go` (160 lines) had 85% structural duplication. Extracting a `ConfigStore[E ConfigEntry]` generic reduced them to ~60 lines each (entry struct + thin wrappers) while consolidating Load/Save/Add/Remove/Toggle/Update into a single `store.go` (162 lines). The `ConfigEntry` interface only needs `GetPattern()` and `GetEnabled()` — keep the constraint surface minimal.

## License compliance matters for binary distribution
Source distribution via Go modules is implicitly compliant (each module carries its LICENSE). Binary distribution is not — BSD-3-Clause and Apache-2.0 require bundling copyright notices with binaries. A `THIRD_PARTY_NOTICES` file and updating the release script fixed this.

## Wayland clipboard tools silently fail on wrong display server
`xsel` from linuxbrew was found in PATH and reported success on Wayland, but clipboard content didn't persist. Always prioritize the native clipboard tool for the current display server (`wl-copy` for Wayland) over X11 tools.

## url.Host vs url.Hostname() in Go
`url.URL.Host` includes the port (e.g. `example.com:443`), while `url.Hostname()` strips it. When parsing URLs for display (e.g. HAR import), use `Hostname()` to avoid port leaking into host fields.

## Atomic flags for cross-goroutine pause gates
Using `atomic.LoadUint32`/`StoreUint32` for a pause flag on FlowStore avoids mutex contention on the hot path (every Add/Update call). The proxy goroutines check the flag without locking, while the UI goroutine toggles it. For bypassing specific callers (HAR import), provide an `AddDirect` method that skips the check.
