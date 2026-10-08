# TASK P1 — stream, runner, queue

Phase P1 of `PLAN.md` ("Epics and phases"): Components `stream` (Parse Event, Encode Lines), `runner` (Exec Runner),
`queue` (Phase Queue). One code card and one judge card per Function, 8 cards. The cards are the deck
(`decks/P1/deck.json`), not this file.

## 1. Why this

- Every later phase stands on these three packages: `session` imports `stream` (P2), `bootstrap` and `gitrules` import
  `runner` (P3, P4), `registry`, `control`, `supervisor`, `api`, `mcpserver` import `queue` (P4–P6). 3 of 17 packages,
  4 of 27 Functions, 31 of 199 record examples (16 + 7 + 8).
- The tree has 0 product packages after P0 (`go build ./...` green on `internal/` only); after P1: 3 packages, 4 code
  files, 4 example test files, 31 example tests.
- Record size of the Components (YAML dump of the group): stream 10 678 → 10 852 B, runner 3 365 → 3 613 B, queue
  5 868 → 5 868 B; all under the 30 KB limit.
- No open issue is labelled for P1 (`P1-stream`, `P1-runner`, `P1-queue`: checked before preparing).

## 2. Contract

### 2.1. INPUT data shapes the code must build

- **The claude probe log** `tests/fixtures/stream/probe.jsonl` (37 lines, 81 752 B, CLI 2.1.294): one JSON object per
  line `{"t": <float>, "dir": "in"|"out", "msg": <object>}`. `Parse` takes ONE line's `msg` bytes (as
  `testhelp.ProbeLine(t, path, n)` returns them: the raw bytes, with the spaces of the file), never the whole JSONL
  line. Lines the examples use, counted: 1 (in, initialize init-1), 3 (in, user PONG), 4 (out, system init),
  5 (out, assistant, 1 text part "PONG"), 6 (out, rate_limit_event, two unifiedWindows), 7 (out, result success,
  `result` "PONG", no `errors` key), 12 (out, assistant, 1 thinking part), 13 (out, assistant, 1 tool_use part),
  16 (out, control_response to int-1), 19 (out, result error_during_execution, no `result` key, 1 error string),
  27 (out, control_request can_use_tool AskUserQuestion, 1 question with 2 options), 28 (in, the answer to line 27).
  The other 25 lines are read by no example of this phase.
- **The plan fixture** `tests/fixtures/plan/queue-3.json`: exactly one JSON object, the `Plan` that Load takes after
  `json.Unmarshal` into `queue.Plan`: Approved "7b31dfe", 3 phases P17 (stop_after none, no caps), P18 (smoke, caps
  claude_usd 12), P19 (none). Examples 1, 3, 4, 5 of Phase Queue start from it; examples 2, 6 (second half), 7 (zero
  queue), 8 have their literal in the record.
- **Exec Runner** builds no fixture: examples 1–5 run `sh -c` in `t.TempDir()` (example 2 writes one file `f.txt`
  there first); examples 6–7 are literals. `sh` is dash on the VPS (`/bin/sh -> dash`), measured: `sh -c "sleep 5"`
  does NOT exec `sleep` in place, so killing `sh` leaves `sleep` holding the output pipes.
- **Declared dependencies**: none in this phase (the go-sdk is used by `mcpserver` only). Shared test helpers:
  `internal/testhelp` (`Equal`, `WriteFile`, `ProbeLine`, `ProbeOut`), in the slice of every judge.
- **Text an environment can change**, pinned by prefix only: the `runner: ` error of a command that cannot start (the
  os/exec text follows), the `stream: bad json` and `stream: bad input` errors (the encoding/json text follows).
  `$HOME` of example 3 is read with `os.Getenv("HOME")`, never written.

### 2.2. OUTPUT data shapes

The exported signatures are the record's, verbatim (`contour.yaml`, groups stream, runner, queue: "Declares in … exactly").
Tables: example → given → result are the record's examples (31), rendered into the judge cards by `morph plan`.

**Gaps decided here** (each also one line in `docs/DECISIONS.md`):

- Parse Event · an object whose "type" is the empty string "" passes as a string type (Type "", only SessionID and Raw
  set) · the record says "without a string type"; "" is a string; no example pins it.
- Parse Event · error order: empty → bad json → not an object → no type; on every error the Event is the zero Event
  (Raw nil) · example 8 fixes each case; the order makes `[1,2]` "not an object", not "no type".
- Parse Event · `errors` present as `[]` gives `[]string{}`, absent or `null` gives nil · what `encoding/json` decodes;
  example 4 pins only the absent case.
- Parse Event · a control_request whose subtype is not can_use_tool keeps Tool "" and Input nil; an AskUserQuestion
  with zero questions keeps Question nil but Input set · the record's "when" clauses read literally.
- Encode Lines · `parent_tool_use_id` is a field marshalled as JSON null (a nil pointer or a RawMessage "null", no
  omitempty) · example 3 pins the bytes.
- Encode Lines · Allow validates the input with `json.Valid` first; an empty or nil input is "not valid JSON" →
  error beginning "stream: bad input" · json.Marshal would otherwise write `null` for an empty RawMessage.
- Encode Lines · Answer with input that is not JSON at all (not only a non-object) → "stream: input is not an object"
  · the record says "must decode into map[string]any, else"; one error for both.
- Exec Runner · `OS.Run` sets `Cmd.WaitDelay = 500 * time.Millisecond` and may import `time` for it (never
  `time.Now`) · RECORD CHANGED: example 5 ("returns within 2 s") is unreachable without it on dash (measured: 5.0 s
  without, 0.7 s with); the runner description and behaviour now say so.
- Exec Runner · the "cannot start" error is `fmt.Errorf("runner: %s: %w", name, err)`-shaped, pinned by the prefix
  "runner: " only; Stdout/Stderr empty in that Result and in the ctx-done Result · example 4 pins prefix and Code.
- Exec Runner · `OS` has a value receiver, `Fake` a pointer receiver (`*Fake` implements Runner; its mutex makes it
  non-copyable); Fake records `Args` as given (nil for no args) and never records env · example 6 compares Calls with
  non-empty Args only.
- Exec Runner · RECORD CHANGED: example 6's `Default` is `Result{Stdout: "default\n", Code: 4}` (was `Result{Code: 0}`,
  the zero Result, which a Fake that ignores Default also returns: a hard-coded constant the examples did not vary).
- Phase Queue · Load stores the trimmed Approved and trimmed IDs, checks duplicates on trimmed IDs, copies the phases
  (never aliases the caller's slice); on any error it returns the zero Queue · no example pins the stored form.
- Phase Queue · Load checks in this order: approved, phases, then per phase in index order: empty id, duplicate,
  stop_after · example 2 has one fault per plan; the order decides a plan with two.
- Phase Queue · State "empty" is never produced by this package in v1 (Load of no phases is an error) · reserved for
  callers that display a queue not yet loaded.
- Phase Queue · in Done the "running" id is Current().ID only while State is "running", else "" · examples 5 and 6
  ("but running \"\"" while stopped or queued).
- Phase Queue · Done to the end keeps Reason as it was ("" in every example) · nothing reads Reason after done.
- Phase Queue · example 6's "a running queue at P17" is the queue of example 1 after Start(); example 7's "done queue"
  is the queue at the end of example 5 · the record names the states, not how to reach them.
- Phase Queue · RECORD CHANGED: example 8's defaults are `{20, 2, 4}` and its Caps `{20, 1, 4}` (was `{30, 3, 5}` /
  `{30, 1, 5}`, the same defaults as example 1: a code hard-coding 30/3/5 passed every example). The probe adds one
  zero-caps phase under `{20, 2, 4}` so the Hours default is varied too.
- Encode Lines · RECORD CHANGED: the behaviour said "< > & become < > &" and example 4's literal showed the raw
  characters; both now carry the six-character escapes `< > &` that encoding/json writes (measured on
  Go 1.25.14), and example 4 says the Go want is a raw string literal.
- Map · every judge instruction (27 entries) said "per example of `<file>` taken from go.mod"; now "per example …
  listed below (the examples of the Function)" · the examples come from the record, never from go.mod.

### 2.3. Names

- Code (one file per card, code only — no smoke test target): `stream/parse.go`, `stream/encode.go`,
  `runner/runner.go`, `queue/queue.go`; packages `stream`, `runner`, `queue` of module `morphstudio`.
- Judge files, one test per record example, in example order: `stream/parse_examples_test.go`
  (`TestParseEventExample1..8`), `stream/encode_examples_test.go` (`TestEncodeLinesExample1..8`),
  `runner/runner_examples_test.go` (`TestExecRunnerExample1..7`), `queue/queue_examples_test.go`
  (`TestPhaseQueueExample1..8`).
- Probes (data, inlined into the code cards' acceptances, removed after the step): `decks/P1/parts/_<card>_probe_test.go`
  (the Go probe name `morph plan` reads), tests `TestProbe<Function>Example<N>`.

### 2.4. What must not break

Frozen byte for byte (every acceptance's `== frozen`): `contour.yaml`, `morph-map.json`, `docs`, `decks`,
`tests/fixtures`, `go.mod`, `go.sum`, `internal`, `vendor`. The suite before P1: 0 tests outside the vendored tree
(`internal/testhelp` has none); after: 31 example tests. No ripple: every target is a new file (no scout run, no
spike: there is no existing code a new file can redden).

## 3. Acceptance

Built by `morph plan --checks decks/P1/checks.json` (Go profile, vendor mode), every stage printing `== <stage>`.

- Code card: `== build` (overlay hides the generation's sibling targets) → `== vet` → `== gofmt` → `== guard`
  (layers of `decks/tools/layers.json`: stream, runner, queue import no module package) → `== probe` (the card's probe,
  red per example on a stub) → `== imports` (allow-list per package: stream `bytes, encoding/json, errors, fmt,
  strings`; runner `bytes, context, errors, fmt, os, os/exec, strings, sync, time`; queue `encoding/json, errors, fmt,
  strings` — the `stdlib_rules` of layers.json: os/exec only in runner; time.Now, math/rand, crypto/rand, net, the
  http client calls, syscall, unsafe in none of the three) → `== clock` (`gofmt -l -r 'time.Now -> time.Time'` on the
  code file) → `== full` (`go test -count=1 ./...`) → `== frozen` → no untracked file.
- Judge card: `== vet` → `== gofmt` → `== guard <file>` (min = examples, max = min + 6: parse 8..14, encode 8..14,
  runner 7..13, queue 8..14; the literals: every `Test<Function>Example<N>` plus distinct values of the examples, in
  `checks.json`) → `== own` (`go test` of the package) → `== full` → `== frozen` → no untracked file.
- Measured on a scratch worktree: every acceptance red on stubs at its first meaningful stage (code: `== probe`, one
  `--- FAIL: TestProbe…ExampleN` per example, 31 of 31, each with an `example N …` got/want line, no panic; judges:
  `== guard`, one "does not mention the example literal Test…ExampleN" line per missing example); green on a reference
  implementation written in the scratch tree only; the longest chain 16.8 s (cold Go cache), under 2.5 s warm.

## 4. Constraints

- Module `morphstudio`, Go 1.25, `GOFLAGS=-mod=vendor GOPROXY=off`; no `any`-typed public API beyond `json.RawMessage`.
- Stubs and helpers only from `morphstudio/internal/testhelp`; a test writes only under `t.TempDir()`.
- `stream` and `queue` are pure (no I/O, no clock, no environment); `runner` is the only package that runs a process.
- A judge writes only its test file; a code card writes only its code file (no test target, so no smoke cap).
- Generations (measured by the cut): [exec-runner, parse-event, phase-queue], [encode-lines, exec-runner-judge,
  parse-event-judge, phase-queue-judge], [encode-lines-judge]. `stream/parse_examples_test.go` and
  `stream/encode.go` are written in the same generation in the same package: the overlay hides each from the other's
  acceptance; neither is in the other's slice.

## 7. Out of scope

- `session` (the machine that consumes `stream.Event`), `eventlog`, `telegram` — P2.
- `claude` (launch args, the process behind an interface; the only other `os/exec`/`syscall` user), `github`,
  `bootstrap` — P3. Smoke stop 1 (after P3) exercises `stream` against a real claude.
- Persisting a `queue.Queue` (`registry`), the control surface over it (`control`), the phase loop (`supervisor`) — P4–P5.
- Process groups / killing grandchildren in `runner` (WaitDelay only bounds the wait; the orphaned `sleep` dies on its
  own); stdin for commands; streaming output.
- Any record example beyond the 31 (the probes' extra variant checks follow the behaviour text; they are not examples).

## 8. How to run

```
node /home/morph/MorphV2/dist/cli.js plan --root . --spec contour.yaml --map morph-map.json \
  --component stream --component runner --component queue --judge \
  --checks decks/P1/checks.json --out decks/P1/deck.json
python3 decks/tools/scale_tokens.py decks/P1/deck.json 3
node /home/morph/MorphV2/dist/cli.js deck check --root . --deck decks/P1/deck.json        # errors 0
node /tmp/morph-bin-P1/dist/cli.js run --root . --deck decks/P1/deck.json --processor ds --deadline 2400
```

The cut holds exactly the 8 P1 cards: no filter script is needed.

## 9. Pre-registration

- 8 cards, 3 generations [3, 4, 1]; processor `ds` (deepseek/deepseek-v4.1-flash), maxTokens ×3: exec-runner 30 000,
  parse-event 36 000, phase-queue 36 000, encode-lines 24 000, exec-runner-judge 42 000, parse-event-judge 60 000,
  phase-queue-judge 54 000, encode-lines-judge 48 000.
- Bill: forecast $0.10–0.15 (P15L on the same model: $0.0852 for 14 cards, 59 726 output tokens; prices derived from
  its report: ≈ $0.30 / M input, ≈ $1.20 / M output). Ceiling if every card used its whole budget once: 330 000 output
  tokens ≈ $0.40 + ≈ 70 000 input ≈ $0.02 = $0.42. Cap $5.
- Expected regenerations: exec-runner (example 5: an author who omits WaitDelay reddens `== probe` at "example 5
  returns within 2 s"); encode-lines (example 4: an author who disables HTML escaping reddens "example 4 User bytes");
  encode-lines-judge (example 4: a want written as an interpreted Go string reddens `== own` on correct code — the
  record now says "raw string literal").
- Falsifiable: 8/8 written within one fix; 31 example tests green on `go test ./...`; no card over 1 regeneration.

## 10. What to record

The `docs/MEASURE.md` row and §11: attempts per card, first red per burned variant, neighbour-red, judge defects,
guard rejections, lines by hand (0), max slice bytes, minutes, $.

## 11. Actual

### Preparation (before the gate)

- `morph plan` exit 0; `morph deck check` errors 0, warnings 0, hazards 0; generations [3, 4, 1].
- Largest slice: parse-event-judge and encode-lines-judge, 84 876 B before targets (the probe log is 81 752 B of it).
- Probes red per example on stubs: 31 / 31 (parse 8, encode 8, runner 7, queue 8); judges red at `== guard` on a
  one-test stub (every missing example named).
- Acceptance chains: max 16.8 s (cold cache), green on the scratch reference for all 8.
- Mutants: 30 on the reference (parse 10, encode 5, runner 7, queue 8), 30 killed, every one under a 120 s timeout,
  ≈ 0.5 min in total over two rounds (round 1: 2 mutants did not compile and 1 survived — Hours default hard-coded —
  fixed by valid mutants and by a probe variant; round 2: 30 / 30 killed). Survivors: none.
- Preparation: 12 min wall clock (orchestrator agent, Opus 5.5); 0 lines of product code by hand (the reference and the stubs lived only in a scratch worktree, removed).

### Run 20261008-171107 (processor ds, binary copy /tmp/morph-bin-p1 of MorphV2 bc311aa)

- 8 / 8 written, 0 burned; 9 requests; 92 977 input / 59 534 output tokens; $0.0495 executor; 318 s (5.3 min).
- Attempts: phase-queue 2 (v1 red at `== build`: "queue/queue.go:107:9: cannot use q.Current() (value of type bool)
  as error value in return statement"; r1 green); every other card 1. None of the three pre-registered regenerations
  happened (exec-runner set WaitDelay, encode-lines kept HTML escaping, encode-lines-judge wrote example 4 green).
- Judges: 4 / 4 green at `== own` on the accepted code; judge defects 0; guard rejections 0; neighbour-red 0.
- Verify on the run branch: `git status --short` empty; `go vet ./...`, gofmt, `go build ./...` clean;
  `go test ./...` green (queue, runner, stream; 31 example tests).
- Read against §2.2 (defects recorded, not fixed by hand; no example covers them, so the acceptances stay green):
  1. queue/queue.go `Load` stores Approved and IDs untrimmed and checks duplicates on the untrimmed IDs (§2.2: trimmed),
     so "P1" and " P1" both load;
  2. runner/runner.go: the cannot-start error is `runner: <err>` without the name (§2.2 shape `runner: <name>: <err>`,
     only the prefix pinned), and the ctx-done Result keeps the captured Stdout/Stderr (§2.2: empty).
- Falsifiable claims of §9: 8/8 within one fix — held (no fix needed); 31 tests green — held; no card over 1
  regeneration — held.
- Lines by hand: 0. Largest slice 84 876 B. `.morph/primer.md` re-generated by `morph primer` after the run.
