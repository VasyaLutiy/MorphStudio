# TASK P5 — supervisor, api

Phase P5 of `PLAN.md` ("Epics and phases"): Components `supervisor` (Phase Loop, Runtime Guard) and `api` (HTTP
Handlers, Router). One code card and one judge card per Function, 8 cards. The cards are the deck
(`decks/P5/deck.json`), not this file. No stop after P5: P6 follows.

## 1. Why this

- The daemon (P7) drives every phase only through `supervisor`: begin, the git checks of P4 (`gitrules`), the
  session's `phase_done` / `wait_operator`, the PM's `continue` / `restart` (scenarios S4, S5 of the brief; flow 5 and
  flow 7), and the runtime guard — exits resumed ≤ N per hour, the usage-limit pause and the 60 % alert, the one stall
  nudge, the wall-clock and stretch caps. The operator's curl and the final smoke reach morphd only through `api`
  (15 routes behind the bearer token, `/healthz` open, the MCP mount unwrapped). 2 of 17 packages, 4 of 27 Functions,
  32 record examples (9 + 8 + 8 + 7; no example added, six rewritten to inline what they named).
- The tree after P4: 12 product packages, 127 example tests. After P5: 14 packages, 4 more code files, 4 more example
  test files, 159 example tests.
- Record size of the Components (YAML dump of the group): supervisor 17 873 → 19 340 B, api 10 889 → 12 181 B; both
  under the 30 KB limit.
- No open issue is labelled for P5 (`gh issue list` empty).

## 2. Contract

### 2.1. INPUT data shapes the code must build

- **One fixture file**: `tests/fixtures/plan/queue-3.json` (unchanged since P0) — one `queue.Plan` object, text:
  `approved "7b31dfe"`, three phases `P17 none`, `P18 smoke` with `caps {claude_usd 12}`, `P19 none`. Phase Loop and
  Runtime Guard decode it with `json.Unmarshal` into `queue.Plan` and `queue.Load(plan, queue.Caps{30, 3, 5})`: a queue
  `State "queued"`, `Index 0`, phase caps P17 {30, 3, 5}, P18 {12, 3, 5}, P19 {30, 3, 5}. HTTP Handlers example 7 posts
  the file's bytes as the body of `POST /projects/demo/plan`; the decoded plan has 3 phases, `Phases[1].StopAfter
  "smoke"`, `Phases[1].Caps.ClaudeUSD 12` (every key of the file is a known field, so DisallowUnknownFields accepts it).
- **Every other literal lives in the record**: the supervisor's inputs are `gitrules.Start` / `gitrules.End` values,
  `stream.Result` / `stream.Limits` values and times; the api's inputs are `httptest.NewRequest` requests on a fake
  Control of the test file.
- **Preconditions of the callees** (supervisor judges): queue · `Start()` fails `queue: cannot start in state "<s>"`
  unless "queued" · Phase Loop example 1. queue · `Done(phase, next)` needs the queue "running" with `next` equal to the
  id after the current one; it returns `{Next, Stop}` — `Stop "smoke"`/`"operator"` from the finished phase's
  `stop_after`, `"end"` after the last phase (queue State "done") · examples 5, 7, 8. queue · `Stop(reason)` stops only
  a "running" or "queued" queue; `Continue()` only a "stopped" one, else `queue: nothing to continue (state "<s>")` ·
  examples 4, 7, 8, 9. queue · `Current()` holds the phase's Caps (defaults filled by Load) · examples 2, 3 (BudgetUSD
  30 and 12), Runtime Guard 7 (Hours 3). control · `Percent(0.6)` = 60 · Runtime Guard example 4. control ·
  `ErrNotWaiting`, `ErrNoSession` are package variables · Phase Loop examples 7–9.
  (api judges): control · `Code(err)` returns `(int, string)` and maps by errors.Is in the record's order; a 500 writes
  message "internal" · HTTP Handlers 1, 4, 8. session · `ErrBadOption` text "option out of range", so
  `fmt.Errorf("%w: 3 of 1..2", control.ErrBadOption)` prints "option out of range: 3 of 1..2" · HTTP Handlers 4.
  session · `OrderResult` marshals with omitempty (`{"queued":1}`, `{"sent":true}`) · HTTP Handlers 2. github ·
  `Access` marshals `checked, reachable, push, reason, checked_at` in that order · HTTP Handlers 8.
- **Harness skeletons** (in the judge instructions): supervisor — the queue above, `New("demo", q)`, `Begin(15:00:00Z)`,
  `StartChecked(gitrules.Start{Mode: "fresh"}, func returning "id-1", 15:00:01Z)` = the running loop; fields an example
  names are set directly. api — a struct fake (`hhFake` / `rtFake`) implementing the 17 methods with configurable results
  and a recorded call list; `httptest.NewRequest` + `SetPathValue("project", …)` for a handler, `r.ServeHTTP` for the
  router; never a listener.
- **Distinct markers**: the router's MCP stub answers status 299 with body "mcp " + path, distinct from every JSON body.
- **Carried known risks and P5**: none of P1–P4's read defects is reachable from a P5 example. queue `Load` untrimmed
  IDs — the loop uses the ids as stored; registry and control errors — the api passes any Control error to `Code`
  (registry.ErrExists would answer 500, the P7 translation stands).
- **Text an environment can change**: none; every message here is exact (the mux's own 404/405 bodies are not
  compared, statuses only).
- **Declared dependencies**: none in this phase. Shared test helpers: `internal/testhelp` (`Equal`), in every judge
  slice.

### 2.2. OUTPUT data shapes

The exported signatures are the record's, verbatim (`contour.yaml`, groups supervisor, api: "Declares in …"). Tables:
example → given → result are the record's examples (32), rendered into the judge cards by `morph plan`.

**Gaps decided here** (each also one line in `docs/DECISIONS.md`):

- Phase Loop · RECORD CHANGED: `Continue` on a "not_pushed" stop calls `Queue.Continue()` then `Queue.Start()` (the same
  phase runs again) before `[{end_check Phase}]`; example 7 adds `Queue.State "running"`, `Queue.Index 0` and the next
  `EndChecked(OK)` → `[{kill}, {start_check P18}]`, StretchUSD 1.92, SessionUSD 0 · the old text left the queue
  "stopped", so the next `Queue.Done` failed, its zero Outcome matched no branch and the loop stuck after a fixed push
  (scenario S5's continue).
- Phase Loop · RECORD CHANGED: `Begin` returns `Queue.Start()`'s error as is; the State is not checked apart from it (a
  starting, running or paused loop holds a running queue) · example 1's second Begin, made in State "starting", pins
  the queue's text; a separate State check with its own error would contradict it.
- Phase Loop · RECORD CHANGED: an action carries only the fields its literal names, every other field zero
  (start_check/end_check {Kind, Phase}; spawn {Kind, Phase, SessionID, Resume, BudgetUSD}; first_line/write {Kind,
  Line}; kill {Kind}; post {Kind, Milestone}) · the examples compare whole slices ("exactly"); the P2 decision for
  Session Machine actions.
- Phase Loop · RECORD CHANGED: the next phase begun inside `EndChecked` is the Begin transition (Stop nil, Nudged and
  Alerted false, SessionUSD 0); the "end" Stop carries `At now` · without the reset the next EndChecked adds the finished
  session's cost to StretchUSD twice (the stretch cap); probe variants pin both.
- Phase Loop, Runtime Guard · RECORD CHANGED: the examples inline the queue (`queue.Load` of the fixture with
  `queue.Caps{30, 3, 5}`) and the running loop (Begin 15:00:00Z, StartChecked fresh "id-1" 15:00:01Z) instead of naming
  "Phase Queue example 1" / "Phase Loop example 2"; Phase Loop examples 3 and 4 say how their starting loop is built ·
  an example never refers to another Function's example (the slice never holds it).
- HTTP Handlers, Router · RECORD CHANGED: the examples inline the Status and Question literals and the exact JSON of
  Control Contract examples 1 and 3; the Router examples get their own fake (Status{busy, P18, 37, s-1} for "demo") with
  the full body, plus `GET /projects/nope/status` → 404 (the router fills the path value) · the same lesson.
- Imports · RECORD CHANGED: supervisor adds `strconv`, `strings` ('f' -1 numbers, Problems joined with "; ") · a code
  card's allowed imports are its record list (P3 lesson); `== imports` allows exactly the record's lists.
- Phase Loop · a fresh start appends " · " + any non-empty Reason to the numbers (probe: "pulled to ccc333" with caps
  12.5 / 2.5 → "cap $12.5 · 2.5 h"); a resume keeps Resumes, a fresh start sets them nil; StartChecked outside
  "starting" is nil; the Loop's JSON is exactly the record's tags (probe: a New loop, "stop" omitted when nil); Action
  has no tags · the record's text.
- Runtime Guard · "within the last hour" is `now − t < 1 h` (a resume exactly 1 h old does not count — probe); Resumes
  are appended, never pruned; `Turn` sets SessionUSD in every state; the alert compares
  `SevenDay.Utilization*100 >= float64(UsageAlertPercent)`; `Limits` on a paused loop goes only to the alert branch;
  `Tick` checks the paused resume, then the wall-clock cap, then the stall; Hours is fractional (2.5 h) · the record read
  literally; probe variants (StallMinutes 45, Hours 2.5, cap 10 reached exactly, max 1 and 5, the 75 % threshold).
- HTTP Handlers · the error body keys are `code` then `message`; Content-Length 0 skips both the Content-Type check and
  the read (Restart only), an unknown length (-1) reads; a body over 1 MiB is 400 bad_json (probe: a 1 MiB text; a
  512 KiB text accepted); Events: an absent or empty `since`/`max` is 0, `since` is checked first; only Projects turns a
  nil list into `[]`; CreateProject passes `r.Context()` and answers 201; the 415 and 400 bodies are written with
  WriteJSON; a body followed by white space only is accepted (probe: a trailing newline) · the record's rules; probe
  variants for every body-reading handler (415) and every view.
- Router · `Bearer`: the header's first 7 bytes equal "Bearer " ignoring case, the rest compared whole with
  `subtle.ConstantTimeCompare` (no trim: "Bearer  tok-1" and "Bearertok-1" refused); an empty configured token refuses
  before comparing (`ConstantTimeCompare` of two empty slices is 1); every one of the 14 token routes refuses without
  the header (probe); the mux's 404/405 bodies are not pinned · examples 2, 4; probe variants.
- Map · budgets from the expected answers: phase-loop-judge 24 000 → 28 000 (the judge with the most examples, 9; ≈ 15
  KB of Action literals), runtime-guard-judge 20 000 → 24 000, router-judge 20 000 → 22 000 (a 17-method fake),
  http-handlers-judge 28 000 kept (a 17-method fake plus 8 request tables ≈ 18 KB); code cards kept (phase-loop 14 000,
  runtime-guard 12 000, http-handlers 16 000, router 8 000; the scratch reference is 5.9, 3.7, 5.4 and 1.7 KB).
- Map · two test files per package in different generations, neither in the other's slice: helper prefixes `pl` / `gd`
  (supervisor), `hh` / `rt` and fakes `hhFake` / `rtFake` (api); the instructions name every arity (Begin, PhaseDone,
  Continue, Restart → ([]Action, error); the rest []Action; Event nothing), ask for `[]Action(nil)` (a bare nil never
  equals a typed nil slice under reflect.DeepEqual) and for every headline, numbers string, write Line and JSON body as
  one whole literal (the guard's literals).

### 2.3. Names

- Code (one file per card, code only — no smoke test target): `supervisor/loop.go`, `supervisor/guard.go`,
  `api/handlers.go`, `api/router.go`; packages `supervisor`, `api` of module `morphstudio` (layers already in
  `decks/tools/layers.json`: supervisor → queue, gitrules, control, stream; api → control, queue, github, eventlog,
  session).
- Judge files, one test per record example, in example order: `supervisor/loop_examples_test.go`
  (`TestPhaseLoopExample1..9`), `supervisor/guard_examples_test.go` (`TestRuntimeGuardExample1..8`),
  `api/handlers_examples_test.go` (`TestHTTPHandlersExample1..8`), `api/router_examples_test.go`
  (`TestRouterExample1..7`).
- Probes (data, inlined into the code cards' acceptances, removed after the step): `decks/P5/parts/_<card>_probe_test.go`,
  tests `TestProbe<Function>Example<N>`; helpers prefixed `pPL`, `pRG`, `pHH`, `pRT`.

### 2.4. What must not break

Frozen byte for byte (every acceptance's `== frozen`): `contour.yaml`, `morph-map.json`, `docs`, `decks`,
`tests/fixtures`, `go.mod`, `go.sum`, `internal`, `vendor`, and the code of P1–P4: `stream`, `runner`, `queue`,
`session`, `eventlog`, `telegram`, `claude`, `github`, `bootstrap`, `registry`, `gitrules`, `control`. The suite before
P5: 127 example tests; after: 159. No ripple: every target is a new file in a new package.

## 3. Acceptance

Built by `morph plan --checks decks/P5/checks.json` (Go profile, vendor mode), every stage printing `== <stage>`.

- Code card: `== build` (overlay hides the generation's sibling targets) → `== vet` → `== gofmt` → `== guard` (layers)
  → `== probe` → `== imports` (allow-lists = the record's lists: supervisor `errors, fmt, strconv, strings, time,
  morphstudio/queue, morphstudio/gitrules, morphstudio/control, morphstudio/stream`; api `crypto/subtle, encoding/json,
  errors, io, mime, net/http, strconv, strings, morphstudio/control, morphstudio/queue` — no os, no os/exec, no
  syscall) → `== clock` (no `time.Now`, no `time.Since`, all four) → `== net` (api: no `http.DefaultClient`,
  `http.Get`, `http.Post`, `http.PostForm`, `http.Head`, `http.ListenAndServe`, `http.ListenAndServeTLS`) → `== full` →
  `== frozen` → no untracked file.
- Judge card: `== vet` → `== gofmt` → `== guard <file>` (min = examples, max = min + 6: loop 9..15, guard 8..14,
  handlers 8..14, router 7..13; the literals: every `Test<Function>ExampleN` plus distinct values of the examples, in
  `checks.json`, none holding a quote or a backslash) → `== own` → `== full` → `== frozen` → no untracked file.
- Measured on a scratch worktree (`/tmp/p5-scratch`, data committed there, each card run in the tree of its generation,
  removed after): every code acceptance red at `== probe` on zero-value stubs (untagged structs, methods returning nil,
  `ErrPhaseMismatch = errors.New("stub")`, a Bearer that passes everything, an empty mux), one `--- FAIL:
  TestProbe…ExampleN` per example, 32 of 32 (loop 9, guard 8, handlers 8, router 7), each with an `example N …` line,
  no panic; judges red at `== guard` on a one-test stub (39, 33, 38, 30 guard lines); all 8 green on a reference
  written in the scratch tree only (judges green with test files derived from the probes); chains: the first on a cold
  Go cache 43.7 s (phase-loop) and 42.0 s (router-judge), warm ≤ 4.4 s green, red ≤ 1.1 s.

## 4. Constraints

- Module `morphstudio`, Go 1.25, `GOFLAGS=-mod=vendor GOPROXY=off`.
- Stubs and helpers only from `morphstudio/internal/testhelp`; no test opens a listener or dials (httptest recorders
  only); a test writes nothing.
- `supervisor` is a pure machine: no clock (every time is a `now` argument), no I/O, ids from `newID`; `api` never
  listens and reads no clock.
- A judge writes only its test file; a code card writes only its code file (no test target, so no smoke cap).
- Generations (measured by the cut): [http-handlers, phase-loop], [http-handlers-judge, phase-loop-judge, router,
  runtime-guard], [router-judge, runtime-guard-judge]. `supervisor/guard.go` and `supervisor/loop_examples_test.go`,
  `api/router.go` and `api/handlers_examples_test.go` are written in the same generation in the same package: the
  overlay hides each from the other's acceptance; neither is in the other's slice.

## 7. Out of scope

- Who executes the actions: the daemon (P7) spawns, kills, writes lines, posts milestones, runs the git checks and
  feeds Exited / Turn / Limits / Event / Tick; it implements `control.Control` behind the handlers and mounts the MCP
  handler (P6) into `NewRouter`.
- The MCP endpoints and their tokens (P6); `cmd/morphd` (P7): the listener on 127.0.0.1, the `.env`, the token.
- Persisting the Loop (the daemon saves it through `registry.SaveState`; P5 only declares its JSON tags).
- Any record example beyond the 32 (the probes' extra variants follow the behaviour text; they are not examples).

## 8. How to run

```
node /tmp/morph-bin-P5/dist/cli.js plan --root . --spec contour.yaml --map morph-map.json \
  --component supervisor --component api --judge \
  --checks decks/P5/checks.json --out decks/P5/deck.json
python3 decks/tools/scale_tokens.py decks/P5/deck.json 3
node /tmp/morph-bin-P5/dist/cli.js deck check --root . --deck decks/P5/deck.json        # errors 0
node /tmp/morph-bin-P5/dist/cli.js run --root . --deck decks/P5/deck.json --processor ds --deadline 2400
```

The cut holds exactly the 8 P5 cards: no filter script is needed. `/tmp/morph-bin-P5` is the copy of the MorphV2
binary made for this phase (MorphV2 bc311aa; `dist` copied, `package.json`, `node_modules`, `templates` linked).

## 9. Pre-registration

- 8 cards, 3 generations [2, 4, 2]; processor `ds`, maxTokens ×3: http-handlers 48 000, phase-loop 42 000, router
  24 000, runtime-guard 36 000, http-handlers-judge 84 000, phase-loop-judge 84 000, router-judge 66 000,
  runtime-guard-judge 72 000 (sum 456 000).
- Bill: forecast ≈ $0.055 (P1–P4 actuals $0.2269 for 34 cards, ≈ $0.0067 a card, × 8). Ceiling if every card used its
  whole budget once: 456 000 output × $1.20 / M = $0.55 + ≈ 60 000 input × $0.30 / M = $0.02 → $0.57. Cap $5; gate
  ≤ $1.
- Largest slice at the cut: runtime-guard-judge, 22 662 B of existing files (≈ 32 KB with the written loop.go and
  guard.go); no slice near 200 KB.
- Expected regenerations: phase-loop (the not_pushed continue that must restart the queue: "example 7 after Continue:
  State, Queue.State"; the next-phase reset of SessionUSD: "example 5 variant StretchUSD, SessionUSD"); http-handlers
  (the 1 MiB limit or the second Decode); router (the empty token: ConstantTimeCompare("", "") is 1); the judges
  (a bare `nil` against a typed nil slice; a 17-method fake with a wrong signature; guard literals).
- Falsifiable: 8/8 written within one fix; 159 example tests green on `go test ./...`; no card over 1 regeneration.

## 10. What to record

The `docs/MEASURE.md` row and §11: attempts per card, first red per burned variant, neighbour-red, judge defects,
guard rejections, lines by hand, max slice bytes, minutes, $.

## 11. Actual

### Preparation (before the gate)

- Orchestrator: Opus 5.5 agent (fresh context), ≈ 55 tool calls, ≈ 300k tokens, ≈ 25 min.
- `morph plan` exit 0; `morph deck check` errors 0, warnings 0, hazards 0; generations [2, 4, 2]; the cut holds exactly
  the 8 phase cards.
- Largest slice at the cut: runtime-guard-judge, 22 662 B of existing files (≈ 32 KB with the written files).
- Probes red per example on stubs: 32 / 32 (loop 9, guard 8, handlers 8, router 7); judges red at `== guard` on a
  one-test stub (every missing literal named). Runtime Guard example 8 (all its answers nil) first passed on the
  do-nothing stub; a contrast variant (the same Turn and Exited on a running loop act) makes it red.
- Acceptance chains: the slowest 43.7 s (phase-loop, cold Go cache); all 8 green on the scratch reference; red chains
  ≤ 1.1 s.
- Mutants: 30 on the reference (loop 12, guard 8, handlers 7, router 3), each under `timeout 120`, run against the
  card's probe: 30 / 30 killed at `== probe`, 0.25 min for the final round, ≈ 0.6 min in all. The first round was
  discarded (the derived judge files in the tree collided with the probes' helper names: every "kill" was a build
  error); in the second, one survivor ("no MaxBytesReader": the probe's 1 MiB body also carried an unknown field) was
  killed after the variant became a 1 MiB text, and two mutants that only broke the build were re-done as real
  mutations. Survivors: none.
- Record changes: Phase Loop (not_pushed continue restarts the queue; Begin returns the queue's error; the fields of an
  action; the next-phase reset and the end Stop's At; examples 1, 3, 4, 7 inline or extended), Runtime Guard example 1
  (the running loop inlined), HTTP Handlers examples 1, 6 and Router examples 1, 3 (literals inlined), supervisor's
  import list. Map: 3 budgets, 4 judge instructions.
- 0 lines of product code by hand (the reference and the stubs lived only in a scratch worktree, removed).
