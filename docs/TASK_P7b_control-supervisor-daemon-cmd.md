# TASK P7b — control, supervisor, daemon, pump, cmd (smoke 2 defects D1–D3)

Phase P7b of `PLAN.md` (row P7b and the paragraph after the phase table; the record approved at cc46ed1, unchanged
since). A re-cut of 12 already-built cards (6 code, 6 judge) from the record with `morph plan --only`; MorphV2
ec56481 runs the `--only` deck as one subset transaction (accepted on the fully written tree). The cards are the deck
(`decks/P7b/deck.json`), not this file. Stop after P7b: **smoke 2 re-run, final** (the PM).

## 1. Why this

- Smoke 2 (DECISIONS 08.10 · smoke stop 2) was RED on three defects, $0 of claude spent because no session ever started:
  - **D1** (blocker): the session id is 32 hex characters; claude 2.1.294 refuses it ("Invalid session ID. Must be a
    valid UUID.").
  - **D2**: the exit is invisible and unbounded — 57 spawns in ~2 min, no stderr logged, status "busy", `resumes`
    null (the ≤ 3 per hour guard never counted a fresh start).
  - **D3**: `five_hour` / `seven_day` show 0, not null, before the first rate_limit_event.
- 6 of 27 Functions re-generated, 13 files (7 code, 6 example tests), 55 record examples (Control Contract 10, Phase
  Loop 9, Runtime Guard 9, Daemon Core 11, Pump 9, Config And Main 7); 15 of them new or changed by cc46ed1.
- PLAN "what is checked", rows marked P7b: Config And Main 6, 7 (D1); Runtime Guard 1, 2, 8, 9, Phase Loop 1, 2, 9,
  Pump 8, 9, Daemon Core 11, Control Contract 10 (D2); Control Contract 8, 9, Daemon Core 1, Pump 1 (D3).
- The tree before: 201 example tests counted by the primer; after: 201 − 48 + 55 = 208 in these six files plus the
  rest unchanged.

## 2. Contract

### 2.1. INPUT data shapes the code must build

- Fixtures unchanged: `tests/fixtures/plan/queue-3.json`, `tests/fixtures/git/measure.md`,
  `tests/fixtures/stream/probe.jsonl` (as in `docs/TASK_P7_daemon-cmd.md` §2.1).
- Callee shapes: `claude.Process` / `claude.ExitStatus{Code, Stderr}` / `claude.Fake` (Kill finishes with −1),
  `claude.Start(ctx, bin, args, dir, env)`, `claude.Args(l)` (`claude/process.go`, `claude/args.go`);
  `eventlog.Entry{Seq, T, Dir, Msg}` (`eventlog/log.go`); `queue.Queue`, `gitrules.Start/End`, `stream.Result/Limits`.
- API changes across the group (the record's, verbatim): `supervisor.Loop.Resumes` → `Restarts` (json "restarts"),
  `Loop.LastExit *control.Exit`; `(*Loop).Exited(code int, stderr string, now time.Time, cfg Config)`; `control.Exit`,
  `control.Status.LimitsUnknown` (json "-") and `.LastExit`, `Status.MarshalJSON`, `Usage.MarshalJSON`;
  `cmd/morphd.UUIDv4([16]byte) string`, `func newID() string` in main.go. No caller outside the 13 targets (grep of
  `Resumes`, `.Exited(`, `newID`, `LastExit`, `LimitsUnknown` over `*.go` outside vendor).

### 2.2. OUTPUT data shapes

The exported signatures and texts are the record's (`contour.yaml`, groups control, supervisor, daemon, pump, cmd);
the 55 examples are rendered into the judge cards by `morph plan`.

### 2.3. Names

Packages `control`, `supervisor`, `daemon`, `cmd/morphd`. Post texts: "<phase>: session exited (code <c>),
restarting", "restart <n> of <max> this hour[ · <line>]", "<phase>: session keeps exiting", "exited <n+1> times within
an hour (last code <c>)[: <line>]"; Stop kind "crash"; the exit log entry Dir "exit", Msg {"code","stderr"}.

### 2.4. What must not break

Frozen byte for byte (`== frozen`): `contour.yaml`, `morph-map.json`, `docs`, `decks`, `tests/fixtures`, `go.mod`,
`go.sum`, `internal`, `vendor`, `stream`, `runner`, `queue`, `session`, `eventlog`, `telegram`, `claude`, `github`,
`bootstrap`, `registry`, `gitrules`, `api`, `mcpserver`. `api` and `mcpserver` tests build control.Status values without
LimitsUnknown: their JSON keeps the ints (LimitsUnknown false).

## 3. Acceptance

Built by `morph plan --checks decks/P7b/checks.json --only <12 ids>` (Go profile, vendor mode, subset transaction).

- Code cards: `== build` → `== vet` → `== gofmt` → `== guard` → `== probe` (`decks/P7b/parts/_<card>_probe_test.go`,
  one `TestProbe<Function>Example<N>` per record example, variants inside) → `== imports` (the record's list per
  package) → `== clock` / `== net` / `== listen` as in P4, P5, P7 → `== full` → `== frozen` → no untracked file.
- Judges: `== vet` → `== gofmt` → `== guard <file>` (min = examples, max = min + 6: control 10..16, phase-loop 9..15,
  runtime-guard 9..15, daemon-core 11..17, pump 9..15, config 7..13; literals in `checks.json`) → `== full` →
  `== frozen` → no untracked file.
- The probes are P4/P5/P7's, with the changed examples rewritten and the new ones added: control +8, 9, 10; phase-loop
  1, 2, 3, 9 (Restarts, LastExit); runtime-guard 1, 2, 8 rewritten, 9 added (texts "restarting"/"restart", stderr line,
  rune cut); daemon-core 1 (LimitsUnknown, null Usage), the behaviour test's post text, 11 added (a Spawn that always
  fails); pump 5, 6 rewritten, 7 (a real `sh` claude), 8 added, old 7 → 9; config 6, 7 added.

## 4. Constraints

- Module `morphstudio`, Go 1.25, `GOFLAGS=-mod=vendor GOPROXY=off`; the clock and ids are injected everywhere but
  `cmd/morphd`.
- Tests open no listener, reach no network, write only under `t.TempDir()`; Pump 7 runs `/bin/sh` scripts only.
- Generations [1,2,2,2,1,3,1]; one owner per file; a red group regenerates all 12 cards (`--deadline 2400`).

## 7. Out of scope

- Any edit of the record or the map (a record gap is a stop with a report).
- `api`, `mcpserver` and every package outside control, supervisor, daemon, cmd/morphd.
- The smoke itself (the PM after the run); merge to `main` (the operator).

## 8. How to run

```
~/bin/morphv2 plan --root . --spec contour.yaml --map morph-map.json --component control --component supervisor \
  --component daemon --component pump --component cmd --judge --checks decks/P7b/checks.json \
  --only control-contract,control-contract-judge,phase-loop,phase-loop-judge,runtime-guard,runtime-guard-judge,daemon-core,daemon-core-judge,pump,pump-judge,config-and-main,config-and-main-judge \
  --out decks/P7b/deck.json
python3 decks/tools/scale_tokens.py decks/P7b/deck.json 3
~/bin/morphv2 deck check --root . --deck decks/P7b/deck.json        # errors 0
~/bin/morphv2 run --root . --deck decks/P7b/deck.json --processor ds --deadline 2400
```

## 9. Pre-registration

- 12 cards, 7 generations; processor `ds`, maxTokens ×3.
- Bill: P7 was $0.0896 for 6 cards with retries; 12 cards ≈ $0.20, one full group regeneration ≈ $0.40. Cap $5.
- Falsifiable: 12/12 written; `go build ./...`, `go vet ./...` and `go test ./...` green; the D1–D3 examples green.

## 10. What to record

Report to the team lead: prep minutes, scout numbers, deck check, stub check, run id, cards, retries, $, verify.

## 11. Actual

### Preparation

- Scout: the first run (seed from git ownership) spent its 120 000-char budget reading contour.yaml, no answer
  ($0.0365); the second with a narrow seed (5 code files, "do not read contour.yaml") answered in 153 s, $0.0259,
  12 targets (every deck target but `cmd/morphd/config.go`), context `stream/parse.go`. Roles changed: 0.
- Stub check (scratch worktree `/tmp/P7b/wt`, removed after): the built code of P4/P5/P7 with only the record's API shapes
  put in by a throwaway edit (Exit, LimitsUnknown, LastExit, Restarts, Exited's stderr argument, UUIDv4/newID returning
  ""), the six judge files replaced by one-test stubs. Every code acceptance red at `== probe` exactly on the changed or
  new examples, each with a readable line and a first difference, the unchanged examples green on the old code:
  control 8, 9, 10; phase-loop 1, 2; runtime-guard 1, 2, 8, 9; daemon-core 1 + behaviour (11 kills the old binary with
  the D2 recursion itself — a fatal stack overflow, the last test of the file, after every other line); pump 5, 6, 7, 8;
  config 6, 7. Judges red at `== guard` (41, 45, 43, 65, 59, 42 lines). `decks/tools/stubcheck.mjs` exit 0 on all 12
  logs. Slowest chain 37 s (daemon-core, the overflow). The acceptances need Go 1.25 first on PATH
  (`GOTOOLCHAIN=local`; /usr/bin/go is 1.22).
