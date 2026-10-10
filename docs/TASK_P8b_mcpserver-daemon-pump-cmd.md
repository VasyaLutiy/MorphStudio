# TASK P8b — mcpserver · PM Tools, daemon · Daemon Core, pump · Pump, cmd · Config And Main

Phase P8b of `PLAN.md` (row P8b and the paragraph "P8a / P8b — the minimal PM MCP harness"; design 50a83fd approved
by the operator, DECISIONS 10.10; QUESTIONS Q23–Q29 accepted as proposed; P8a merged). A re-cut of 9 built cards,
cut with `morph plan --only` as one subset transaction. The cards are the deck (`decks/P8b/deck.json`), not this file.
After P8b, `POST /projects/adopt` and `project_adopt` adopt for real (the daemon is the Adopter), `events` serves the
compact view over MCP, and the last turn's cost is counted after `phase_done`.

## 1. Why this

- Operator items 2, 3 and 5 (10.10): over MCP the PM has 0 adopt tools and 0 progress tools (PMServer 10 tools,
  UserServer 2); `POST /projects/adopt` answers 500 `internal` (no Adopter); after `phase_done` the end batch kills the
  session before the turn's `result` arrives, so `stretch $` misses the last turn (smoke 1: the session's last turn,
  $0.0058 of probe line 31, is the cost lost in Pump example 10).
- 4 Functions, 9 cards, 9 files: 5 re-cut code files (`mcpserver/pm.go`, `daemon/daemon.go`, `daemon/pump.go`,
  `cmd/morphd/config.go`, `cmd/morphd/main.go`) and 5 re-cut test files (`mcpserver/pm_examples_test.go`,
  `mcpserver/mount_examples_test.go`, `daemon/daemon_examples_test.go`, `daemon/pump_examples_test.go`,
  `cmd/morphd/config_examples_test.go`).
- Record examples: PM Tools 9 (1 and 6 change, 8–9 new), MCP Mount 7 (1 and 6 change), Daemon Core 14 (12–14 new),
  Pump 11 (10–11 new), Config And Main 7 (2–5 change).
- Tests before: 230 top-level (P8a); after: 230 + 2 + 3 + 2 = 237.

## 2. Contract

### 2.1. INPUT data shapes the code must build

- `control.Control` (17 methods, unchanged), `control.Adopter`, `control.Adopt`, `control.ErrUnsupported`
  (`control/adopt.go`); `control.Events`, `control.Views`, `control.EventsView` (`control/events.go`); `control.Code`,
  the sentinels (`control/control.go`).
- `bootstrap.AdoptSpec{Name, Language, RepoURL, Dir, CredentialsFile}`, `bootstrap.ErrRefused`, `bootstrap.Adopt`
  (`bootstrap/adopt.go`); `registry.Project`, `Registry.Add/SecretPath` (`registry/`).
- `supervisor.Loop` (`PhaseDone`, `Turn`, `EndChecked`, `Exited`, `Tick`), `supervisor.Action` (`supervisor/loop.go`,
  `supervisor/guard.go`); `claude.Process`, `claude.Fake` (`claude/process.go`); `stream.Event` (`stream/parse.go`).
- Fixtures unchanged: `tests/fixtures/stream/probe.jsonl` (lines 3, 5, 7 for PM Tools 8; line 31, the result with
  total_cost_usd 0.005775230000000001, for Pump 10), `tests/fixtures/plan/queue-3.json`, `tests/fixtures/git/measure.md`.

### 2.2. OUTPUT data shapes

As the record says, verbatim: PMServer tool `events {since, limit}` (limit 0 → 50, `control.Views(ev, 200)`);
UserServer tool `project_adopt {name, dir, repo_url, language}` (`control.Adopt`); `daemon.Config.DoneWait
time.Duration`, the unexported `pendingDone`, `proj.done`, `(*Daemon).AdoptProject`, `(*Daemon).finishDone`, the
`var _ control.Adopter = (*Daemon)(nil)`; pump calls `finishDone` after a `result` and on the exit of a process with a
pending done; `cmd/morphd.Config.DoneWait int` (`MORPHD_DONE_WAIT`, default 30), passed as seconds.

### 2.3. Names

Packages `mcpserver`, `daemon`, `main` (`cmd/morphd`). Tests `TestPMToolsExample1..9`, `TestMCPMountExample1..7`,
`TestDaemonCoreExample1..14`, `TestPumpExample1..11`, `TestConfigAndMainExample1..7` in the five existing files.

### 2.4. What must not break

Frozen byte for byte (`== frozen`): `contour.yaml`, `morph-map.json`, `docs`, `decks`, `tests/fixtures`, `go.mod`,
`go.sum`, `internal`, `vendor`, `tools` and every package but `mcpserver`, `daemon`, `cmd/morphd`. `control.Control`
keeps its 17 methods. Daemon Core 1–11 and Pump 1–9 keep their text and their tests (DoneWait 0 = the old behaviour);
every old test name of the five test files survives. `mcpserver/mount.go` and `mcpserver/session.go` are not targets.

## 3. Acceptance

Built by `morph plan --checks decks/P8b/checks.json --only <the 9 ids>` (Go profile, vendor mode, subset transaction).

- Code cards (pm-tools, daemon-core, pump, config-and-main): `== build` → `== vet` → `== gofmt` → `== guard` →
  `== probe` (`decks/P8b/parts/_<id>_probe_test.go`: the probes of P6 / P7b with the changed and new examples) →
  `== imports` (the record's import list of the package) / `== clock` / `== net` (config-and-main: `== listen`,
  `== donewait`: main.go sets `DoneWait`) → `== full` → `== frozen` → no untracked file.
- Judges: `== vet` → `== gofmt` → `== guard <file>` (existing files, `new: false`: every old test name kept; min = the
  examples, max = min + 6; literals in `checks.json`, each found in the card's text) → `== full` → `== frozen` → no
  untracked file.
- Stubs (`decks/P8b/_stubs/`): every code target is re-cut with intent `generate`, so its stub holds ONLY the names the
  record gives that file, typed (orchestrator skill §6.1, #22): bodies return a "stub <name>" error or a zero value
  (`PMServer`/`UserServer` an empty server), never the current file; each test file one passing test. `deck check` then
  builds the kept `mcpserver/session.go` against the pm.go stub. Every probe example is red on the stubs.

## 4. Constraints

- Module `morphstudio`, Go 1.25 (`GOTOOLCHAIN=local`, Go 1.25 first on PATH), `GOFLAGS=-mod=vendor GOPROXY=off`.
- Generations [2, 3, 3, 1]; one owner per file; `--deadline 2400`.
- Package daemon reads no clock (the deadline is `Now() + DoneWait`, checked by `Tick`); no test opens a listener.

## 7. Out of scope

- Any edit of the record or the map beyond the data fix of 10.10 after run 20261010-190913 (a gap is a stop with a report):
  PM Tools names the three unexported helpers `jsonResult`, `errorResult`, `okView` that the kept `session.go` calls; the
  instructions of `pump-judge` and `daemon-core-judge` list their guard literals and the int64 types; the lit "git branch"
  of `daemon-core-judge` became "--no-merged".
- `mcpserver/mount.go`, `mcpserver/session.go`, `control`, `bootstrap`, `api`, `supervisor` and every other package.
- Smoke stop 3 (the PM over the systemd morphd on the VPS) — the PM's, after the merge; the VPS; merge to `main`.
- The restart gap (a pending done lost when morphd restarts) — accepted by Q25.

## 8. How to run

```
~/bin/morphv2 plan --root . --spec contour.yaml --map morph-map.json --component mcpserver --component daemon \
  --component pump --component cmd --judge --checks decks/P8b/checks.json \
  --only pm-tools,pm-tools-judge,mcp-mount-judge,daemon-core,daemon-core-judge,pump,pump-judge,config-and-main,config-and-main-judge \
  --out decks/P8b/deck.json
python3 decks/tools/scale_tokens.py decks/P8b/deck.json 3
~/bin/morphv2 deck check --root . --deck decks/P8b/deck.json        # errors 0
~/bin/morphv2 run --root . --deck decks/P8b/deck.json --processor ds --deadline 2400
```

## 9. Pre-registration

- 9 cards, 4 generations; processor `ds`, maxTokens ×3.
- Bill: PLAN forecasts $0.40; cap $5.
- Falsifiable: 9/9 written; `go build ./...`, `go vet ./...`, `go test ./...` green with 237 top-level tests;
  `go test -race -count=3` green on daemon, mcpserver, cmd/morphd.

## 10. What to record

Report to the team lead: prep minutes, deck check, stub check, run id, cards, retries, $, verify.
