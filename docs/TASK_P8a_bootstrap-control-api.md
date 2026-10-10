# TASK P8a — bootstrap · Project Adopt, control · Adopt Contract + Event View, api · HTTP Handlers + Router

Phase P8a of `PLAN.md` (row P8a and the paragraph "P8a / P8b — the minimal PM MCP harness"; design 50a83fd approved
by the operator, DECISIONS 10.10; QUESTIONS Q23–Q29 accepted as proposed). Three new Functions and a re-cut of two
built cards with their judges, cut with `morph plan --only` as one subset transaction. The cards are the deck
(`decks/P8a/deck.json`), not this file. After P8a, `POST /projects/adopt` answers 500 `internal` until P8b gives the
daemon its Adopter; everything else stays green.

## 1. Why this

- Operator items 2 and 3 (10.10): the PM must adopt an existing repository (MorphStudio itself, smoke 3) and read a
  session's progress over MCP without raw lines; today neither exists (0 adopt routes, `events` returns raw entries
  only).
- 5 Functions, 10 cards, 10 files: 3 new code files (`bootstrap/adopt.go`, `control/adopt.go`, `control/events.go`),
  3 new test files, 2 re-cut code files (`api/handlers.go`, `api/router.go`) and their 2 test files.
- Record examples: Project Adopt 7, Adopt Contract 3, Event View 6 (all new); HTTP Handlers 9 (1 new, 8 unchanged);
  Router 8 (1 new, 7 unchanged).
- Tests before: 212 top-level (`go test -v ./...`, 10.10 on 8b6930d); after: 212 + 7 + 3 + 6 + 1 + 1 = 230.

## 2. Contract

### 2.1. INPUT data shapes the code must build

- `bootstrap.Report{Dir, Steps}` (`bootstrap/create.go`); `runner.Runner`, `runner.Key`, `runner.Result`
  (`runner/runner.go`); `github.Parse(repoURL) (owner, repo, error)` (`github/access.go`).
- `control.Control` (17 methods, unchanged), `control.ProjectView`, `control.Events{Entries, Last}`, `control.Percent`,
  `control.Code`, `control.ErrBadInput`, `control.ErrExists` (`control/control.go`).
- `eventlog.Entry{Seq, T, Dir, Msg}` (`eventlog/log.go`); `stream.Parse` → `stream.Event{Type, Subtype, Tool, Text,
  Question, Result, Limits}` (`stream/parse.go`).
- Fixture unchanged: `tests/fixtures/stream/probe.jsonl` (37 lines; lines 3–7, 12–19, 27, 28, 34 are the examples'),
  read by `testhelp.ProbeLine` (`internal/testhelp/probe.go`); `tests/fixtures/plan/queue-3.json`.

### 2.2. OUTPUT data shapes

As the record says, verbatim: `bootstrap.AdoptSpec`, `bootstrap.ErrRefused`, `bootstrap.Adopt`; `control.Adopter`,
`control.ErrUnsupported` ("adopt not supported", Code 500 `internal`), `control.Adopt`; `control.EventExit`,
`control.EventView`, `control.EventsView`, `control.Cut`, `control.View`, `control.Views`; `api.Handlers.AdoptProject`
(201 the ProjectView); route `POST /projects/adopt` behind Bearer.

### 2.3. Names

Packages `bootstrap`, `control`, `api`. Tests `TestProjectAdoptExample1..7` (`bootstrap/adopt_examples_test.go`),
`TestAdoptContractExample1..3` (`control/adopt_examples_test.go`), `TestEventViewExample1..6`
(`control/events_examples_test.go`), `TestHTTPHandlersExample9`, `TestRouterExample8` (added to the existing files).

### 2.4. What must not break

Frozen byte for byte (`== frozen`): `contour.yaml`, `morph-map.json`, `docs`, `decks`, `tests/fixtures`, `go.mod`,
`go.sum`, `internal`, `vendor`, `tools` and every package but `bootstrap`, `control`, `api`. `control.Control` keeps its
17 methods (every judge's fake). HTTP Handlers 1–8 and Router 1–7 stay green; `bootstrap/create.go` and
`control/control.go` are not targets.

## 3. Acceptance

Built by `morph plan --checks decks/P8a/checks.json --only <the 10 ids>` (Go profile, vendor mode, subset transaction).

- Code cards (project-adopt, adopt-contract, event-view, http-handlers, router): `== build` → `== vet` → `== gofmt` →
  `== guard` → `== probe` (`decks/P8a/parts/_<id>_probe_test.go`; the api probes are P5's with HTTP Handlers 9 /
  Router 8 added) → `== imports` (the record's import list of the package) / `== clock` (no time.Now / time.Since) /
  `== net` (control, api: no HTTP client, no listener) → `== full` → `== frozen` → no untracked file.
- Judges: `== vet` → `== gofmt` → `== guard <file>` (min = examples, max = min + 6; the two existing api files keep every
  old test name; literals in `checks.json`, each found in the card's text) → `== full` → `== frozen` → no untracked file.
- Stubs (`decks/P8a/_stubs/`): new files are typed exports returning zero values and a "stub" error; `api/handlers.go`
  is the current file plus an empty `AdoptProject`; `api/router.go` is the current file. Every new probe example is red;
  the unchanged HTTP Handlers 1–8 and Router 1–7 are green on the stubs by design.

## 4. Constraints

- Module `morphstudio`, Go 1.25 (`GOTOOLCHAIN=local`, Go 1.25 first on PATH), `GOFLAGS=-mod=vendor GOPROXY=off`.
- Generations [3, 4, 2, 1]; one owner per file; `--deadline 2400`.
- `AdoptProject` lives on the optional `control.Adopter`, never on `control.Control`.

## 7. Out of scope

- Any edit of the record or the map (a gap is a stop with a report).
- P8b: `mcpserver` (the `events` and `project_adopt` tools), `daemon` (the Adopter, the last-turn wait), `pump`, `cmd`.
- `bootstrap/create.go`, `control/control.go`, every other package; the VPS; merge to `main` (the operator).

## 8. How to run

```
~/bin/morphv2 plan --root . --spec contour.yaml --map morph-map.json --component bootstrap --component control \
  --component api --judge --checks decks/P8a/checks.json \
  --only project-adopt,project-adopt-judge,adopt-contract,adopt-contract-judge,event-view,event-view-judge,http-handlers,http-handlers-judge,router,router-judge \
  --out decks/P8a/deck.json
python3 decks/tools/scale_tokens.py decks/P8a/deck.json 3
~/bin/morphv2 deck check --root . --deck decks/P8a/deck.json        # errors 0
~/bin/morphv2 run --root . --deck decks/P8a/deck.json --processor ds --deadline 2400
```

## 9. Pre-registration

- 10 cards, 4 generations; processor `ds`, maxTokens ×3.
- Bill: PLAN forecasts $0.25; cap $5.
- Falsifiable: 10/10 written; `go build ./...`, `go vet ./...`, `go test ./...` green with 230 top-level tests.

## 10. What to record

Report to the team lead: prep minutes, deck check, stub check, run id, cards, retries, $, verify.
