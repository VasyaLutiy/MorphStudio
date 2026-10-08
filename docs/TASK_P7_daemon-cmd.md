# TASK P7 — daemon, cmd

Phase P7 of `PLAN.md` ("Epics and phases"): Components `daemon` (Daemon Core, Pump) and `cmd` (Config And Main). One
code card and one judge card per Function, 6 cards. The cards are the deck (`decks/P7/deck.json`), not this file. Stop
after P7: **smoke stop 2, final** (`PLAN.md`).

## 1. Why this

- Smoke 2 (`PLAN.md`) needs a running `morphd`: P1–P6 built 15 packages of parts (machines, loops, checks, an HTTP API
  and an MCP mount over `control.Control`) but nothing implements `control.Control`, nothing reads the claude process,
  and there is no `main`. Without P7 every scenario row S1–S5 of the brief stops at a fake.
- 2 of 17 packages, 3 of 27 Functions, 22 record examples (Daemon Core 10, Pump 7, Config And Main 5; one added:
  Daemon Core 10, CreateProject).
- The tree after P6: 15 product packages, 179 example tests. After P7: 17 packages (`daemon`, `cmd/morphd`), 4 code
  files, 3 example test files, 201 example tests.
- Record size (YAML dump of the group): daemon 17 560 → 29 079 B (under the 30 KB limit), cmd 5 087 → 6 313 B.
- Carried into P7 (AUTONOMY "State at handoff"): registry.ErrExists ≠ control.ErrExists (now translated, Daemon Core
  example 10); Begin/Continue's queue errors → 500 (now ErrNotWaiting / ErrBadInput, probe); telegram Post panics on a
  nil Do (main passes `http.DefaultClient.Do`); Runtime Guard `Exited` prunes `Resumes` (Pump example 5 pins 1 resume
  after one exit).
- No open issue is labelled `P7-daemon` or `P7-cmd` (none labelled at all for P7 in the repository).

## 2. Contract

### 2.1. INPUT data shapes the code must build

- **Fixtures** (unchanged since P0):
  - `tests/fixtures/plan/queue-3.json` — one `queue.Plan` as text (approved "7b31dfe"; P17 none, no caps; P18 smoke caps
    {claude_usd 12}; P19 none). Daemon Core 2, 4, 5, 9 and every Pump example decode it with `json.Unmarshal` into
    `queue.Plan` and hand it to `PlanLoad`; `queue.Load` with the project's Caps {30, 3, 5} gives P17 {30, 3, 5}, P18
    {12, 3, 5}, P19 {30, 3, 5}.
  - `tests/fixtures/git/measure.md` — a MEASURE table with rows `| P16 primer-go |` and `| P17 debt |`; every judge
    writes its bytes to `<tmp>/projects/<name>/docs/MEASURE.md`, so the end check of P17 is OK with the fresh script.
  - `tests/fixtures/stream/probe.jsonl` — Pump examples 1, 2, 4 emit the `msg` bytes of lines 4, 5, 6, 7 (init,
    assistant "PONG", rate_limit_event 0.22/0.6, result $0.0021784900000000004), 27 (AskUserQuestion), 10 (thinking
    tokens); example 2 compares the written answer with line 28. Examples 3 and 7 emit literals the record holds.
- **Preconditions of the callees** (judges): Phase Start Check · the runner scripts rev-parse --abbrev-ref HEAD,
  rev-parse HEAD, rev-parse origin/main, status --porcelain (fetch falls to the Fake's zero Default) · else Mode
  "error" and no spawn (Daemon Core 2–10, Pump 1–7). Phase End Check · MEASURE.md holds the phase row · else
  "no MEASURE row" joins the reason (Daemon Core 5, 6). Project Registry · Open creates base · tests pass
  `filepath.Join(t.TempDir(), "state")`. supervisor · Exited resumes while fewer than MaxResumesPerHour; StartChecked
  "resume" keeps the SessionID (Pump 5); "fresh" mints a new one and clears Resumes (Daemon Core 4, 5). claude.Fake ·
  Kill finishes the fake (Lines closed, exit −1), Write after the finish is `claude.ErrExited`.
- **Harness skeleton** (in both daemon judge instructions; each file its own names `dcHarness`/`dc…`, `puHarness`/`pu…`):
  `reg, _ := registry.Open(filepath.Join(tmp, "state")); reg.Add(registry.Project{…})`; MEASURE.md written from the
  fixture; `&runner.Fake{Script: …}`; Spawn appends the Launch and a `claude.NewFake()`; Post appends `[4]string`; a
  clock +1 s per call; NewID `id-<n>`; the four under one mutex; `d, _ := Open(context.Background(), deps)`.
- **Distinct markers**: the post headlines "P17 started", "P18 started", "P17 resumed", "P17 not pushed",
  "P17: wait_operator smoke", "P17: usage limit, paused", "P17: resumed after the limit", "P17: session exited (code
  4), resuming", "P17 turn ended: PONG" share no substring a comparison could confuse (tests compare whole tuples).
- **Text an environment can change**: tmp paths are built by `filepath.Join`, never literal; the clock count is not
  pinned (CheckedAt, CreatedAt, AskedAt, LimitsAt, Stop.At are "not zero").
- **No declared dependency** used: `daemon` and `cmd/morphd` import only the module and the standard library (the SDK
  reaches main through `mcpserver.Handler`).

### 2.2. OUTPUT data shapes

The exported signatures are the record's, verbatim (`contour.yaml`, groups daemon and cmd). Tables: example → given →
result are the record's examples (22), rendered into the judge cards by `morph plan`.

**Gaps decided here** (each also one line in `docs/DECISIONS.md`):

- Daemon Core, Pump · RECORD CHANGED: daemon.go cannot call pump.go's `d.pump` — generation 1 (daemon-core) builds the
  package without pump.go. daemon.go declares `var startPump func(d *Daemon, ctx, project, p, l)` (nil), the spawn
  action calls it when not nil, and pump.go's init sets it; daemon.go declares no pump and no Wait.
- Daemon Core · RECORD CHANGED: one daemon mutex `d.mu` (not one per project) taken by every exported method for its
  whole call; the unexported `execute` and `writeLine` (pinned names) run with it held and never lock, so the pump can
  call them; the unexported `Daemon` and `proj` fields are pinned (pump.go reads them) · a per-project mutex plus a
  cross-project MaxParallel count deadlocks two PlanLoads; an `Execute` that locks deadlocks the pump.
- Daemon Core · RECORD CHANGED: the MCP config file is `<base>/<project>/morphd-mcp.json`, the path
  `claude.WriteMCPConfig(base/<project>, "morphd", …)` returns (the record said `mcp.json`, which that call never
  writes).
- Daemon Core · RECORD CHANGED: the session ids: examples 4, 5, 9 and Pump 6 spawn "id-3" (token "id-4"), not "id-4"
  (token "id-5") — each example builds its own daemon after example 2, where Interrupt's "int-id-3" is not drawn.
- Daemon Core · RECORD CHANGED: every example inlines its harness (the Phase Start Check script, the fake Control, the
  Bash line of Parse Event 7, the tools/list body of MCP Mount 1): an example never names another Function's example.
  The constants vary: ClaudeBin "/opt/claude/bin/claude", MorphBin "/usr/local/bin/morph", MCPBaseURL
  "http://127.0.0.1:7181", Build's token "tok-7".
- Daemon Core · Status.SessionID is the machine's SessionID when not empty (the init line's), else the loop's · Pump 1
  says "967e8f4d-…" (the probe's init) while Daemon Core 2 says "id-1" before any line.
- Daemon Core · Usage.StretchCostUSD = loop StretchUSD + SessionUSD; LimitsAt the machine's · no example pinned Usage;
  Pump 1 now does.
- Daemon Core · RECORD CHANGED: example 10 (CreateProject): a registered name → control.ErrExists before any command;
  bootstrap's refusal with no step run → ErrBadInput wrapping its text; a failed step → its error as is (probe);
  registry.ErrExists → control.ErrExists, ErrBadName → ErrBadInput · the P4 carried risk (registry.ErrExists ≠
  control.ErrExists) and "the error returned as is" → 500 for a bad language.
- Daemon Core · Continue: a non-sentinel error of loop.Continue (a queue error) → `fmt.Errorf("%w: %v",
  control.ErrNotWaiting, err)` (probe: after the plan's end, `not waiting: queue: nothing to continue (state "done")`,
  409); PlanLoad's Begin error → ErrBadInput · the P5 carried risk (500 "internal").
- Daemon Core · the spawn action kills a still-held process first · Continue after wait_operator and PlanLoad on a
  waiting project would otherwise leak a live claude process.
- Daemon Core · Events of a project without a log → `Entries: []eventlog.Entry{}` (not nil) and Last 0; Projects of an
  empty registry → `[]` · the JSON reads `[]`, as the P5/P6 list decisions.
- Daemon Core · a "kill" leaves the process current until its pump sees the exit; Order/Answer/Interrupt need a current
  process (else ErrNoSession) · the record's "nil process ignored" and Restart's "a process alive".
- Pump · RECORD CHANGED: example 5 re-scripts the start check to resume (" M notes.md" → "dirty tree (1 paths)"): with
  the fresh script the loop mints a new id and clears Resumes, contradicting "Resume true, SessionID id-1, 1 resume";
  Status after a resume is "ready" (no first line is written on a resume), not "busy".
- Pump · RECORD CHANGED: example 7 Ticks with the running clock (past 14:20:00Z), not "the clock at 14:20" (the clock
  starts at 15:00 and only advances).
- Pump · RECORD CHANGED: Wait waits for the pumps of the project whose process is no longer current (a pump of the
  current process never ends in a test); example 5 polls for the second Spawn before Wait (until then the exited
  process may still be current).
- Pump · lines of a process that is no longer current are appended to its own log and change nothing else · the
  record's "pumped to its end but its exit changes nothing" said nothing of its lines.
- Pump · RECORD CHANGED: the whole handling of a line, the "out" append included, runs under d.mu · measured in the
  scratch tree: with the append outside the mutex a test polling Events saw the line before its save, and the save
  then raced `t.TempDir` cleanup ("directory not empty", 1 of 10 runs); under the mutex 40 plain and 10 -race runs
  are green. The probes wait for a killed-but-current process to be cleared before they end.
- Daemon Core · RECORD CHANGED: Config.Defaults in the harness is {25, 2, 4}, distinct from the projects' Caps
  {30, 3, 5} (PlanLoad must load with the project's Caps; CreateProject stores the Defaults, example 10); example 7
  adds Continue after wait_operator → the held process killed by the spawn, P17 spawned again (a mutant that skipped
  that kill survived).
- Config And Main · RECORD CHANGED: MORPHD_CLAUDE_EXTRA_ARGS with no field → nil (example 2 says ExtraArgs nil;
  `strings.Fields("")` is `[]string{}`); example 5 inlines the fake Control's Status literal and its exact JSON, adds
  GET /projects/demo/status without a token → 401.
- Config And Main · `go build ./cmd/morphd` (the profile's `== build`) writes the binary `morphd` at the root, which the
  last step "files left in the tree" refused: `.gitignore` gains `/morphd` (measured on the reference in the scratch tree).
- Imports · RECORD CHANGED: daemon adds net/http (Deps.GitHubDo), sort, strconv, unicode/utf8; `== imports` allows
  exactly the record's list.
- Map · budgets (before ×3): daemon-core 24 000 kept (reference 15.9 KB), pump 10 000 kept (2.6 KB), config-and-main
  12 000 kept (5.3 KB); daemon-core-judge 32 000 (the probe derived from it 26 KB), pump-judge 28 000 (15 KB),
  config-and-main-judge 16 000 → 18 000 (9.4 KB with a 17-method fake). Slices: pump adds supervisor/loop.go; config adds
  claude/args.go, registry/secrets.go, supervisor/loop.go; daemon-core-judge adds session/orders.go and the measure
  fixture; pump-judge adds the measure fixture; config-and-main-judge adds github/access.go, session/orders.go and its
  target `cmd/morphd/config_examples_test.go` (the cut derived `morphd/…`).

### 2.3. Names

- Code (code only, no smoke test target): `daemon/daemon.go`, `daemon/pump.go`, `cmd/morphd/config.go` +
  `cmd/morphd/main.go` (one card). Layers already in `decks/tools/layers.json` (daemon: every package but api,
  mcpserver; cmd/morphd: every package).
- Judge files, one test per record example, in example order: `daemon/daemon_examples_test.go`
  (`TestDaemonCoreExample1..10`), `daemon/pump_examples_test.go` (`TestPumpExample1..7`),
  `cmd/morphd/config_examples_test.go` (`TestConfigAndMainExample1..5`).
- Probes (data, inlined into the code cards' acceptances, removed after the step): `decks/P7/parts/_<card>_probe_test.go`,
  tests `TestProbe<Function>Example<N>` plus `TestProbeDaemonCoreBehaviour`; helpers prefixed `pDC`, `pPU`, `pCM`.

### 2.4. What must not break

Frozen byte for byte (every acceptance's `== frozen`): `contour.yaml`, `morph-map.json`, `docs`, `decks`,
`tests/fixtures`, `go.mod`, `go.sum`, `internal`, `vendor`, and the code of P1–P6: `stream`, `runner`, `queue`,
`session`, `eventlog`, `telegram`, `claude`, `github`, `bootstrap`, `registry`, `gitrules`, `control`, `supervisor`,
`api`, `mcpserver`. The suite before P7: 179 example tests; after: 201. No ripple: every target is a new file in a new
package.

## 3. Acceptance

Built by `morph plan --checks decks/P7/checks.json` (Go profile, vendor mode), every stage printing `== <stage>`.

- Code cards: `== build` → `== vet` → `== gofmt` → `== guard` → `== probe` → `== imports` (daemon: context,
  encoding/json, errors, fmt, net/http, os, path/filepath, sort, strconv, strings, sync, time, unicode/utf8 and the 13
  module packages of the record; cmd/morphd: bufio, context, crypto/rand, encoding/hex, errors, flag, fmt, log,
  net/http, os, path/filepath, strconv, strings, time and every module package) → daemon: `== clock` (no time.Now,
  time.Since, time.Until) and `== net` (no http.DefaultClient, Get, Post, PostForm, Head, ListenAndServe(TLS)); cmd:
  `== listen` (main.go calls http.ListenAndServe and holds the literal "127.0.0.1:") → `== full` → `== frozen` → no
  untracked file.
- Judges: `== vet` → `== gofmt` → `== guard <file>` (min = examples, max = min + 6: daemon 10..16, pump 7..13, config
  5..11; the literals in `checks.json`, none holding a quote or a backslash) → `== own` → `== full` → `== frozen` → no
  untracked file.
- Measured on a scratch worktree (`/tmp/p7-scratch`, the data committed there, each card in the tree of its
  generation, removed after): every code acceptance red at `== probe` on stubs (a daemon whose methods return zero
  values, a pump that reads nothing, config/Build returning zero/nil), one `--- FAIL: TestProbe…ExampleN` per example
  with a readable line and first-difference lines, no panic: 22 of 22 (daemon-core 10 + the behaviour test; pump 7,
  examples 2–7 at the shared example-1 setup "Status.CostUSD still 0 after 2 s"; config 5); judges red at `== guard` on
  a one-test stub (53, 41, 32 guard lines); all 6 green on a reference written in the scratch tree only (judges green
  with test files derived from the probes); the slowest chain 39.7 s (daemon-core, cold Go cache), warm ≤ 12.8 s.

## 4. Constraints

- Module `morphstudio`, Go 1.25, `GOFLAGS=-mod=vendor GOPROXY=off`.
- `daemon` reads no clock (Deps.Now), no random source (Deps.NewID), runs no command but through Deps.Runner and
  Deps.Spawn, makes no HTTP call but through Deps.GitHubDo and Deps.Post; only `cmd/morphd` reads `time.Now`,
  `crypto/rand`, `http.DefaultClient` and listens (127.0.0.1 only); the token is never printed.
- Tests open no listener, run no claude, reach no network; they write only under `t.TempDir()`; the pump's goroutines
  use fakes guarded by one mutex.
- A judge writes only its test file; a code card writes only its code file(s).
- Generations (measured by the cut): [daemon-core], [pump], [config-and-main, daemon-core-judge, pump-judge],
  [config-and-main-judge]. The two daemon judges write two files of one package in one generation: the overlay hides
  each from the other's acceptance; helper prefixes `dc` / `pu`.

## 7. Out of scope

- Signals and graceful shutdown of morphd; log files of morphd itself; a systemd unit; TLS (Caddy) — PLAN "Out of scope".
- A data race between the MCP mount's `known`/`session` lookups (registry reads in main) and CreateProject's registry
  write (no registry lock) — a later phase.
- Executor $ enforcement, parallel sessions beyond MaxParallel, periodic GitHub re-checks.
- The first line of a resumed session (the loop writes none on a resume; the stall nudge is the only prod) — recorded
  as a known risk for smoke 2.
- Any record example beyond the 22 (the probes' variants follow the behaviour text; they are not examples).

## 8. How to run

```
node /tmp/morph-bin-P7/dist/cli.js plan --root . --spec contour.yaml --map morph-map.json \
  --component daemon --component cmd --judge --checks decks/P7/checks.json --out decks/P7/deck.json
python3 decks/tools/scale_tokens.py decks/P7/deck.json 3
node /tmp/morph-bin-P7/dist/cli.js deck check --root . --deck decks/P7/deck.json        # errors 0
node /tmp/morph-bin-P7/dist/cli.js run --root . --deck decks/P7/deck.json --processor ds --deadline 2400
```

The cut holds exactly the 6 P7 cards: no filter script is needed.

## 9. Pre-registration

- 6 cards, 4 generations [1, 1, 3, 1]; processor `ds`, maxTokens ×3: daemon-core 72 000, pump 30 000,
  config-and-main 36 000, daemon-core-judge 96 000, pump-judge 84 000, config-and-main-judge 54 000 (sum 372 000).
- Bill: forecast ≈ $0.06 (P1–P6 actuals $0.4072 for 49 cards incl. the P6 fix, ≈ $0.0083 a card, × 6 = $0.05; P7's
  cards are 2–3× larger). Ceiling if every card used its whole budget once: 372 000 × $1.20 / M = $0.45 + ≈ 120 000
  input × $0.30 / M = $0.04 → $0.49; with one regeneration of the two largest cards ≈ $0.70. Cap $5; gate ≤ $1.
- Largest slice at the cut: pump-judge, 132 809 B of existing files (≈ 155 KB with the written daemon.go and pump.go);
  under 200 KB.
- Expected regenerations: daemon-core (the biggest answer: a per-project mutex or an Execute that locks; the MCP path;
  SessionID from the loop only: "example 2 …", "example 1 Status"); pump (the exit of a replaced process handled:
  "example 6 …"); the daemon judges (the clock count, a race on the fakes, a missing guard literal).
- Falsifiable: 6/6 written within one fix; 201 example tests green on `go test ./...`; no card over 1 regeneration.

## 10. What to record

The `docs/MEASURE.md` row and §11: attempts per card, first red per burned variant, neighbour-red, judge defects,
guard rejections, lines by hand, max slice bytes, minutes, $.

## 11. Actual

### Preparation (before the gate)

- Orchestrator: Opus 5.5 agent (fresh context); the preparation facts in §3, §9 and `docs/DECISIONS.md` 08.10 · P7.
- `morph plan` exit 0; `morph deck check` errors 0, warnings 0, hazards 0; generations [1, 1, 3, 1]; the cut holds
  exactly the 6 phase cards.
- Largest slice at the cut: pump-judge, 132 809 B of existing files.
- Probes red per example on stubs: 22 / 22; judges red at `== guard` on a one-test stub.
- Acceptance chains: the slowest 39.7 s (daemon-core, cold Go cache).
- Mutants: 30 on the scratch reference (daemon.go 20, pump.go 7, config.go 3), each under `timeout 120`, two rounds:
  24 / 30 killed (0.57 min) → 3 record/probe changes (§2.2) → 27 / 30 killed (0.82 min). Survivors (DECISIONS known
  risks): PlanLoad with Caps hard-coded to {30, 3, 5}; lines of a replaced process applied to the new machine; Wait
  returning at once.
- Forecast ≈ $0.06 (ceiling $0.49) ≤ $1; largest slice pump-judge 132 809 B at the cut, ≈ 151 338 B with the
  written daemon.go and pump.go (the reference sizes).
- `.gitignore` gains `/morphd` (the profile's `go build ./cmd/morphd` writes it at the root).
- 0 lines of product code by hand.
