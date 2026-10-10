# TASK P7c — supervisor · Phase Loop (smoke 2 defect D4)

Phase P7c of `PLAN.md` (row P7c and the paragraph "P7c (MorphV2 18dab5b …)"; DECISIONS 10.10 lines on D4 and P7c; the
record change approved in advance by the operator, landed in a9a7223, PLAN row de7a5d3). A re-cut of 2 already-built
cards (phase-loop, phase-loop-judge) with `morph plan --only`, run as one subset transaction. The cards are the deck
(`decks/P7c/deck.json`), not this file. Stop after P7c: the PM's smoke 2 check of D4.

## 1. Why this

- Smoke 2 re-run (DECISIONS 10.10): a start check "resume" on a loop with SessionID "" took a new id and spawned
  `--resume <new id>`; claude refused ("No conversation found with session ID") — 4 exits before the guard stopped it,
  0 useful turns.
- 1 of 27 Functions re-generated, 2 files (`supervisor/loop.go`, `supervisor/loop_examples_test.go`), 11 record
  examples: 1 changed (3), 2 new (10, 11), 8 unchanged (1, 2, 4–9).
- Tests before: 210 counted by the primer (9 in `loop_examples_test.go`); after: 210 − 9 + 11 = 212.

## 2. Contract

### 2.1. INPUT data shapes the code must build

- Fixture unchanged: `tests/fixtures/plan/queue-3.json` (P18 has `caps.claude_usd` 12; Hours from the defaults, 3).
- Callee shapes unchanged: `gitrules.Start{Mode, Reason, Branch, Local, Remote}` (`gitrules/start.go`),
  `queue.Queue` / `queue.Caps` (`queue/queue.go`), `control.Milestone`, `control.Stop`, `control.Exit`
  (`control/control.go`).
- No API change: the exported names of `supervisor/loop.go` stay as they are (grep of callers not needed).

### 2.2. OUTPUT data shapes

`StartChecked` with Mode "resume": SessionID non-empty → unchanged (spawn Resume true, post "<phase> resumed",
numbers "session <id> · <reason>", newID not called); SessionID "" → SessionID = newID(), State "running",
PhaseStartedAt = LastEventAt = now, actions [spawn Resume false BudgetUSD Caps.ClaudeUSD, first_line
"/morph-orchestrator <phase>", post start "<phase> resumed on a fresh session" numbers
"session <id> · cap $<ClaudeUSD> · <Hours> h · <reason>"]. Restarts and LastExit untouched in both.

### 2.3. Names

Package `supervisor`; `(*Loop).StartChecked`; headline "<phase> resumed on a fresh session"; tests
`TestPhaseLoopExample1` … `TestPhaseLoopExample11`.

### 2.4. What must not break

Frozen byte for byte (`== frozen`): `contour.yaml`, `morph-map.json`, `docs`, `decks`, `tests/fixtures`, `go.mod`,
`go.sum`, `internal`, `vendor` and every package but `supervisor`. `supervisor/guard.go` and its examples, Daemon Core
and Pump examples (they resume only with a non-empty SessionID) stay green in `== full`.

## 3. Acceptance

Built by `morph plan --checks decks/P7c/checks.json --only phase-loop,phase-loop-judge` (Go profile, vendor mode,
subset transaction).

- phase-loop: `== build` → `== vet` → `== gofmt` → `== guard` → `== probe` (`decks/P7c/parts/_phase-loop_probe_test.go`:
  P7b's probe with example 3 rewritten to the record, its old `id-3` case kept as a variant, plus the end-of-example-5
  path; examples 10 and 11 added) → `== imports` / `== clock` (as P7b) → `== full` → `== frozen` → no untracked file.
- phase-loop-judge: `== vet` → `== gofmt` → `== guard supervisor/loop_examples_test.go` (min 11 = examples, max 17 =
  min + 6; an existing file: the 9 old test names survive; literals in `checks.json`, each found in the record's
  examples or the map instruction) → `== full` → `== frozen` → no untracked file.
- Stubs: the current `loop.go` is red at `== probe` on examples 3 and 11 only.

## 4. Constraints

- Module `morphstudio`, Go 1.25 (`GOTOOLCHAIN=local`, Go 1.25 first on PATH), `GOFLAGS=-mod=vendor GOPROXY=off`;
  clock and ids injected.
- Generations [1, 1]; one owner per file; `--deadline 2400`.

## 7. Out of scope

- Any edit of the record or the map (a gap is a stop with a report).
- `supervisor/guard.go` (Runtime Guard), `daemon`, `cmd/morphd` and every other package.
- The smoke check of D4 (the PM after the run); the VPS rebuild; merge to `main` (the operator).

## 8. How to run

```
~/bin/morphv2 plan --root . --spec contour.yaml --map morph-map.json --component supervisor --judge \
  --checks decks/P7c/checks.json --only phase-loop,phase-loop-judge --out decks/P7c/deck.json
python3 decks/tools/scale_tokens.py decks/P7c/deck.json 3
~/bin/morphv2 deck check --root . --deck decks/P7c/deck.json        # errors 0
~/bin/morphv2 run --root . --deck decks/P7c/deck.json --processor ds --deadline 2400
```

## 9. Pre-registration

- 2 cards, 2 generations; processor `ds`, maxTokens ×3.
- Bill: PLAN forecasts $0.05; cap $5.
- Falsifiable: 2/2 written; `go build ./...`, `go vet ./...`, `go test ./...` green; Phase Loop examples 3, 10, 11 green.

## 10. What to record

Report to the team lead: prep minutes, deck check, stub check, run id, cards, retries, $, verify.
