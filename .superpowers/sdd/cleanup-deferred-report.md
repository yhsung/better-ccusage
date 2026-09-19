# Cleanup-deferred report (Plans 2–4 ledgers)

Branch `feat/cleanup-deferred`, base `03fe3af`. Module `github.com/cobra91/better-ccusage`, stdlib testing only, `-race`, go 1.24.4 (no directive change). No subagents. Nothing dropped — every item applied cleanly.

## Commit 1 — `36ba61c` test(cleanup): harden deferred test gaps (tests only, no behavior change)

- `internal/commands/common_test.go`: added `TestBindCommonFlags_InvalidUntil`, mirroring `InvalidSince` (`--until not-a-time` → PreRunE error).
- `internal/commands/daily_test.go`: added `TestNewDailyCmd_JQAlone` (`--jq` without `--json` yields filtered `2`) and `TestNewDailyCmd_BadJQ` (bad expr → Execute error).
- `internal/mcp/tools/encode_test.go` (new): `TestEncodeResult_Success` (2-space indent shape + round-trip `daily` key) and `TestEncodeResult_Unmarshallable` (func value → `IsError`).
- `internal/mcp/tools/blocks_test.go`: `TestBlocks_Integration` now decodes the payload and asserts 1 block row (mirrors monthly test); added `TestBlocks_InvalidMode` (`Mode: "bogus"` → `INVALID_ARGS`).
- `internal/mcp/server/server_test.go`: added `TestServer_CallMonthlySessionBlocks_Integration` — in-memory `CallTool` for monthly/session/blocks asserting non-error + `"daily"` key (mirrors daily call test).
- `internal/mcp/tools/codex_test.go`: added `TestCodexMonthly_Success` (fake shim JSON with `monthly` key; mirrors daily codex test).
- Tests: `go test ./apps/better-ccusage/... -race` → 102 passed in 16 packages. Targeted runs: 3/3 commands, 9/9 mcp (incl. subtests).
- Dropped: none.

## Commit 2 — `5bd0177` fix(cleanup): close deferred minor findings (≤5 lines each)

- `internal/mcp/server/server.go:29`: comment now lists all six tools (daily/session/monthly/blocks/codex-daily/codex-monthly).
- `internal/mcp/transport/errors.go`: `MapError` maps `ErrIncompatibleMode` → `INVALID_ARGS` (+ `incompatiblemode` table-test row in `errors_test.go`).
- `internal/mcp/tools/codex.go` `runCodexCli`: captures stderr (`var se bytes.Buffer; cmd.Stderr = &se`); on failure appends truncated stderr (first 500 bytes) to the `ToolError` message.
- `internal/commands/statusline.go`: `modelList` drops empty strings and is `sort.Strings`-sorted before `Render` (deterministic output).
- `internal/monitor/monitor.go`: removed dead `log *terminal.Logger` field + constructor assignment + now-unused `terminal` import (grep confirmed no other use).
- `internal/adapters/codex.go:13`: added `TestCodexDetect_PrefixWithoutSession` regression test pinning current behavior (prefix matches with empty SessionID; bare substring needs SessionID — passed pre-change), then added parens (`A || (B && C)`), same semantics.
- Tests: `go build ./...` success, `go vet ./apps/better-ccusage/...` clean, `go test ./... -race -count=1` → 173 passed in 21 packages. `gofmt -l` hits in repo are pre-existing (verified via stash; my files introduce no new ones).
- Dropped: none.

## Commit 3 — `80ce50e` docs(cleanup): tidy Go binary sections (docs only)

- `apps/mcp/README.md`, `apps/codex/README.md`, `apps/opencode/README.md`: moved each Go Binary section to before `## License` (block content verbatim, except mcp note below).
- `apps/mcp/README.md`: tool-args line now states `mode` applies to daily/session/monthly/blocks only; `codex-daily`/`codex-monthly` take `since`/`until` only.
- Tests: docs only, no test run needed (full suite already green at Commit 2; no code touched after).
- Dropped: none.
