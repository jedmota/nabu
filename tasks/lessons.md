# Lessons Learned

## Thread safety must match access patterns
`FlowStore` had a mutex from the start, but `MapRuleStore` did not — even though both are written from the UI goroutine and read from proxy handler goroutines. When adding a new concurrent data structure, check who reads and who writes before deciding on locking.

## Duplication drifts silently
`matchGlob` in `proxy.go` and `matchGlobPattern` in `model/mapping.go` implement the same logic with slightly different names and minor differences (e.g. `?` wildcard support). Same for `stripJSONComments` in two packages. Duplication that starts as "just a quick copy" becomes a maintenance liability as the copies evolve independently.

## License compliance matters for binary distribution
Source distribution via Go modules is implicitly compliant (each module carries its LICENSE). Binary distribution is not — BSD-3-Clause and Apache-2.0 require bundling copyright notices with binaries. A `THIRD_PARTY_NOTICES` file and updating the release script fixed this.
