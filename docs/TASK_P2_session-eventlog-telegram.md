# TASK P2 — session, eventlog, telegram

Phase P2 of `PLAN.md` ("Epics and phases"): Components `session` (Session Machine, Order Queue), `eventlog` (Event
Log), `telegram` (Milestone Post). One code card and one judge card per Function, 8 cards. The cards are the deck
(`decks/P2/deck.json`), not this file.

## 1. Why this

- The daemon (P7) is glue over these three: `session` is the state of one live claude session (orders, the pending
  question, cost, limits), `eventlog` the per-session JSONL log the API serves (`GET events?since=N`, P5), `telegram`
  the milestone posts that replace `tools/tg.sh`. `control` (P4), `api` (P5), `mcpserver` (P6) import `session` and
  `eventlog`; `daemon` (P7) imports all three. 3 of 17 packages, 4 of 27 Functions, 32 record examples (14 + 10 + 8;
  31 before this phase's one added example).
- The tree after P1: 3 product packages, 31 example tests. After P2: 6 packages, 4 more code files, 4 more example test
  files, 63 example tests.
- Record size of the Components (YAML dump of the group): session 10 048 → 10 634 B, eventlog 4 215 → 4 793 B,
  telegram 4 615 → 4 615 B; all under the 30 KB limit.
- No open issue is labelled for P2 (`P2-session`, `P2-eventlog`, `P2-telegram`: checked before preparing).

## 2. Contract

### 2.1. INPUT data shapes the code must build

- **The claude probe log** `tests/fixtures/stream/probe.jsonl` (37 lines, 81 752 B; P1's fixture, unchanged). A
  session test reads ONE line's `msg` with `testhelp.ProbeLine(t, "../tests/fixtures/stream/probe.jsonl", n)` and
  turns it into a `stream.Event` with `stream.Parse` (P1, `stream/parse.go`, in the slice of both session cards).
  Lines the session examples use, counted: 4 (system init, session id 967e8f4d-…), 5 (assistant text PONG),
  6 (rate_limit_event, five_hour 0.22 / 1791469200), 7 (result success, num_turns 1, total_cost_usd
  0.0021784900000000004, result "PONG"), 12 (assistant thinking), 13 (assistant tool_use), 14 (user tool_result),
  16 (control_response to int-1), 19 (result error_during_execution, num_turns 3, total_cost_usd
  0.0028906000000000005, terminal_reason aborted_streaming, no `result` key), 27 (control_request can_use_tool
  AskUserQuestion, request_id 9f4ffa22-…, 1 question, 2 options), 28 (in: the answer to line 27), 31 (result success,
  num_turns 2, total_cost_usd 0.005775230000000001, result "PICKED=Option B"). The other 24 lines are read by no
  example of this phase.
- **Literal stream lines** (no file): Session Machine example 4 holds two control_request lines verbatim in the record
  (`r-7` Bash with input, `r-8` Read without input).
- **The callees in `stream`** (P1, accepted code): `stream.Allow(requestID, input) ([]byte, error)` errors on an input
  that is not valid JSON (empty included); `stream.Answer(requestID, input, question, label) ([]byte, error)` errors
  unless input decodes into an object; `stream.User(text) []byte`, `stream.Interrupt(requestID) []byte` never error.
  Every Line the session writes is one of these, byte for byte.
- **Event Log** builds its files under `t.TempDir()` only: example 6 writes a two-line file with `testhelp.WriteFile`,
  example 7 makes the parent a regular file, example 10 appends one 100 008-byte msg (a 100 052-byte line).
- **Milestone Post** never opens a listener: the judge's fake `Do` records the `*http.Request` (body by `io.ReadAll`)
  and answers a literal `*http.Response{StatusCode, Body: io.NopCloser(strings.NewReader(…))}`.
- **Declared dependencies**: none in this phase (the go-sdk is used by `mcpserver` only). Shared test helpers:
  `internal/testhelp` (`Equal`, `WriteFile`, `ProbeLine`), in the slice of every judge that needs them.
- **Text an environment can change**, pinned by prefix only: `eventlog: open ` (the os text follows), `eventlog: read
  <path> line <n>: ` (the encoding/json text follows), `eventlog: bad msg`.
- **Preconditions of the callees** (judges of session): stream · `Parse` of line 27 gives `Question` non-nil with
  Options ["Option A", "Option B"] and `Input` = the raw request.input · without it Pending cannot be built (Session
  Machine 3, 6; Order Queue 4, 5, 7). stream · `Answer` writes `answers: {question text: label}` into the input ·
  Order Queue 4 compares with line 28 by decoding both sides.

### 2.2. OUTPUT data shapes

The exported signatures are the record's, verbatim (`contour.yaml`, groups session, eventlog, telegram: "Declares in …
exactly"). Tables: example → given → result are the record's examples (32), rendered into the judge cards by
`morph plan`.

**Gaps decided here** (each also one line in `docs/DECISIONS.md`):

- Session Machine · RECORD CHANGED: example 4 named "the Bash can_use_tool line of Parse Event example 7", which no
  session card's slice holds; the line is now inlined verbatim, and a second line without `input` (`r-8`) is answered
  with `updatedInput {}` (Allowed 2); the behaviour says "an Input that is absent or not valid JSON is sent as {}" ·
  `stream.Allow` returns an error for an empty input, and the record wrote `Line: stream.Allow(…)` as one value: a
  nil Line would leave claude waiting on the permission for ever.
- Session Machine · an AskUserQuestion with `ev.Question == nil` (zero questions) takes the "any other tool" branch and
  is allowed · the record's condition read literally.
- Session Machine · a "result" with `ev.Result == nil` (stream.Parse never produces one) counts as the zero Result
  (Turns++, CostUSD 0, LastResult nil) · defensive; no example.
- Session Machine · a control_request of another subtype, a "system" of another subtype and every other type change
  only LastEventAt and return nil · "every other event"; the probe adds a `system`/`status` line (SessionID kept).
- Session Machine · the "turn" Numbers use the result's `NumTurns`, never `Machine.Turns`; `Turns` counts the results
  this machine applied · examples 5 and 6 tell them apart (line 19 says turns 3 on a machine whose Turns becomes 1).
- Session Machine · actions carry only the fields the record writes: `limit` → Kind only; `ask`, `turn` → no Line;
  `write` → Kind and Line only · the record writes them as Go composite literals.
- Session Machine · the ask Numbers are `"%d %s"` per option joined by `" · "` for any count; the probe adds a
  three-option question ("1 Red · 2 Green · 3 Blue") · the record's one example has two options.
- Session Machine · the starting machines of examples 3, 5, 6 are `New()` with fields set directly (State "busy" or
  "question", Queue, Pending built from line 27) · the record names states, not how to reach them.
- Session Machine · after the queue drains, Queue is the remaining slice (length 0, possibly non-nil); Pending.Input
  may alias ev.Input · only the length and the bytes are read.
- Order Queue · Answer when `stream.Answer` fails (Pending.Input not an object) → `(nil, that error)`, Pending and
  State unchanged · unreachable from a parsed AskUserQuestion; no example.
- Order Queue · Answer keeps TurnStartedAt (the turn began before the question); Interrupt never touches Queue · the
  record says "the state unchanged".
- Order Queue · Order in State "question" queues exactly as in "busy"; the probe adds a question machine with two
  queued texts → `{Queued: 3}` · the behaviour names both states, example 2 only "busy".
- Order Queue · the ErrBadOption range follows the option count; the probe adds three options → "option out of range:
  4 of 1..3" and Answer(3) → label "Blue" · example 5 has only two options.
- Order Queue · after the empty orders of example 3 the next order is sent at once (`{Sent: true}`, nothing queued);
  probe variant · example 3 alone ("nothing changes") passes on a do-nothing stub.
- Event Log · RECORD CHANGED: Open reads lines of any length; new example 10 (one 100 052-byte line, reopened whole);
  the description's imports add `bytes` and `io` · bufio.Scanner's 64 KiB default fails with "token too long" on a
  claude tool-result line, and a restarted daemon would refuse its own log.
- Event Log · "synchronously" means one write of the line before Append returns; no fsync required (measured on the
  VPS: 2 500 appends 3 ms without fsync, 2.3 s with) · both pass every example; durability is the caller's later choice.
- Event Log · an Append whose file write fails → `(Entry{}, error beginning "eventlog: write ")`, Last unchanged · no
  example.
- Event Log · Since and Append return copies whose Msg bytes are copied too · "Entries are copies".
- Event Log · the file is opened for append; Open of an existing file keeps the last 2 000 of its entries; the probe
  reopens the 2 500-line file of example 8 (Since(0, 0) still 501..700) · behaviour.
- Event Log · Open counts lines from 1 and a blank line inside the file is a line that fails to decode; the probe adds a
  bad third line ("line 3: ") · example 6 pins only line 2.
- Event Log · Dir is not validated; there is no Close in v1 (the record declares none; the file stays open for the
  life of the process) · the declared signatures are exact.
- Milestone Post · the response body is read to the end and closed on every status; the probe checks Close on 200 and
  403 · "read and closed".
- Milestone Post · only the first 200 bytes of a non-200 body go into the error; the probe adds a 300-byte body · the
  record's 403 body is shorter than 200 bytes.
- Milestone Post · every occurrence of the token is replaced by "***"; the probe adds an error holding it twice ·
  "every occurrence".
- Milestone Post · a request that cannot be built (`http.NewRequestWithContext` error) → `(Status{}, "telegram: post
  failed: " + the masked error)`; a nil Do with Token and ChatID set → `(Status{}, error "telegram: post failed: no
  Do")` · the same path as a Do error; callers always pass Do (cmd/morphd passes `http.DefaultClient.Do`).
- Milestone Post · any status other than 200 (201 included) is an error; kind "" is unknown → 💬; the cut is by runes
  before escaping (an escaped headline may exceed 300 runes) · the record read literally.
- Map · event-log-judge `max_tokens` 18 000 → 20 000 · it is now the judge with the most examples (10); TASK_TEMPLATE:
  the judge with the most examples ≥ 20 000.

### 2.3. Names

- Code (one file per card, code only — no smoke test target): `session/machine.go`, `session/orders.go`,
  `eventlog/log.go`, `telegram/post.go`; packages `session`, `eventlog`, `telegram` of module `morphstudio`.
- Judge files, one test per record example, in example order: `session/machine_examples_test.go`
  (`TestSessionMachineExample1..7`), `session/orders_examples_test.go` (`TestOrderQueueExample1..7`),
  `eventlog/log_examples_test.go` (`TestEventLogExample1..10`), `telegram/post_examples_test.go`
  (`TestMilestonePostExample1..8`).
- Probes (data, inlined into the code cards' acceptances, removed after the step): `decks/P2/parts/_<card>_probe_test.go`,
  tests `TestProbe<Function>Example<N>`; helpers prefixed `pSM`, `pOQ`, `pEL`, `pMP` so the two session probes and
  the session judges never clash.

### 2.4. What must not break

Frozen byte for byte (every acceptance's `== frozen`): `contour.yaml`, `morph-map.json`, `docs`, `decks`,
`tests/fixtures`, `go.mod`, `go.sum`, `internal`, `vendor`, and P1's code `stream`, `runner`, `queue`. The suite before
P2: 31 example tests (stream 16, runner 7, queue 8); after: 63. No ripple: every target is a new file in a new package.

## 3. Acceptance

Built by `morph plan --checks decks/P2/checks.json` (Go profile, vendor mode), every stage printing `== <stage>`.

- Code card: `== build` (overlay hides the generation's sibling targets) → `== vet` → `== gofmt` → `== guard`
  (layers of `decks/tools/layers.json`: session may import stream; eventlog, telegram no module package) → `== probe`
  → `== imports` (allow-list per package from the record and `layers.json` `stdlib_rules`: session `encoding/json,
  errors, fmt, strings, time, morphstudio/stream`; eventlog `bufio, bytes, encoding/json, errors, fmt, io, os,
  path/filepath, strconv, sync, time`; telegram `context, errors, fmt, io, net/http, net/url, strings`) → `== clock`
  (session, eventlog: `gofmt -l -r` finds no `time.Now` and no `time.Since`) or `== net` (telegram: no
  `http.DefaultClient`, `http.Get`, `http.Post`, `http.PostForm`, `http.Head`) → `== full` → `== frozen` → no
  untracked file.
- Judge card: `== vet` → `== gofmt` → `== guard <file>` (min = examples, max = min + 6: machine 7..13, orders 7..13,
  log 10..16, post 8..14; the literals: every `Test<Function>ExampleN` plus distinct values of the examples, in
  `checks.json`) → `== own` → `== full` → `== frozen` → no untracked file.
- Measured on a scratch worktree (data committed there, removed after): every code acceptance red at `== probe` on
  stubs, one `--- FAIL: TestProbe…ExampleN` per example, 32 of 32 (machine 7, orders 7, log 10, post 8), each with an
  `example N …` got/want line, no panic, no compile error; judges red at `== guard` on a one-test stub (the count line
  plus one "does not mention the example literal Test…ExampleN" per missing example); all 8 green on a reference
  written in the scratch tree only (judges green with test files derived from the probes); the longest chain 35.2 s
  (milestone-post, cold Go cache), ≈ 2 s warm.

## 4. Constraints

- Module `morphstudio`, Go 1.25, `GOFLAGS=-mod=vendor GOPROXY=off`.
- Stubs and helpers only from `morphstudio/internal/testhelp`; a test writes only under `t.TempDir()`; no test opens a
  listener or dials (telegram's Do is a fake).
- `session` is pure (no I/O, no clock: `now` is an argument); `eventlog` does file I/O but reads no clock; `telegram`
  makes no HTTP call except through `Client.Do`.
- A judge writes only its test file; a code card writes only its code file (no test target, so no smoke cap).
- Generations (measured by the cut): [event-log, milestone-post, session-machine], [event-log-judge,
  milestone-post-judge, order-queue, session-machine-judge], [order-queue-judge]. `session/orders.go` and
  `session/machine_examples_test.go` are written in the same generation in the same package: the overlay hides each
  from the other's acceptance; neither is in the other's slice.

## 7. Out of scope

- `control` (status JSON over session/eventlog), `api` routes (`GET events`), `mcpserver` tools — P4–P6.
- The daemon that executes the actions (writes Lines to claude's stdin, posts milestones, appends to the log) — P7.
- Log rotation, fsync policy, Close of a log (Q16: one file per session, never rotated in v1).
- Telegram retries, rate limits of the Bot API, message threading.
- Any record example beyond the 32 (the probes' extra variants follow the behaviour text; they are not examples).

## 8. How to run

```
node /tmp/morph-bin-P2/dist/cli.js plan --root . --spec contour.yaml --map morph-map.json \
  --component session --component eventlog --component telegram --judge \
  --checks decks/P2/checks.json --out decks/P2/deck.json
python3 decks/tools/scale_tokens.py decks/P2/deck.json 3
node /tmp/morph-bin-P2/dist/cli.js deck check --root . --deck decks/P2/deck.json        # errors 0
node /tmp/morph-bin-P2/dist/cli.js run --root . --deck decks/P2/deck.json --processor ds --deadline 2400
```

The cut holds exactly the 8 P2 cards: no filter script is needed. `/tmp/morph-bin-P2` is a copy of MorphV2 bc311aa
`dist/` with `node_modules` and `templates` linked beside it.

## 9. Pre-registration

- 8 cards, 3 generations [3, 4, 1]; processor `ds`, maxTokens ×3: event-log 36 000, milestone-post 30 000,
  session-machine 36 000, order-queue 24 000, event-log-judge 60 000, milestone-post-judge 48 000,
  session-machine-judge 72 000, order-queue-judge 54 000 (sum 360 000).
- Bill: forecast ≈ $0.06 (P1 on the same model: $0.0495 for 8 cards, 92 977 in / 59 534 out; P2's cards and slices
  are of the same size: the two session judges carry the 81 752-byte probe log as P1's two stream judges did).
  Ceiling if every card used its whole budget once: 360 000 output ≈ $0.43 + ≈ 120 000 input ≈ $0.04 = $0.47
  (P1's conservative prices, $1.20 / M out, $0.30 / M in). Cap $5; gate ≤ $1.
- Expected regenerations: event-log (example 10: an author who keeps bufio.Scanner's default reddens `== probe` at
  "example 10: reopen of a 100 052-byte line"); session-machine (example 4: an author who passes ev.Input to Allow
  unchecked writes a nil Line for `r-8`); milestone-post (example 7 variant: an author who keeps the whole body);
  session-machine-judge (example 4's two literal lines, written as interpreted strings with escapes, may redden
  `== own` on correct code).
- Falsifiable: 8/8 written within one fix; 63 example tests green on `go test ./...`; no card over 1 regeneration.

## 10. What to record

The `docs/MEASURE.md` row and §11: attempts per card, first red per burned variant, neighbour-red, judge defects,
guard rejections, lines by hand, max slice bytes, minutes, $.

## 11. Actual

### Preparation (before the gate)

- `morph plan` exit 0; `morph deck check` errors 0, warnings 0, hazards 0; generations [3, 4, 1].
- Largest slice: session-machine-judge and order-queue-judge, 95 667 B before targets (the probe log is 81 752 B of
  it; ≈ 105 KB with the written code and the instruction).
- Probes red per example on stubs: 32 / 32 (machine 7, orders 7, log 10, post 8); judges red at `== guard` on a
  one-test stub (every missing example named).
- Acceptance chains: max 35.2 s (cold cache), ≈ 2 s warm; green on the scratch reference for all 8.
- Mutants: 30 on the reference (machine 9, orders 7, log 8, post 6), 30 killed at `== probe`, every one under
  `timeout 120`, 0.4 min in total, one round. Three die only on probe variants (log: the bad line's number constant;
  post: the 200-byte cut, the token masked once). Survivors: none.
- Record changes: Session Machine example 4 and behaviour; Event Log description, behaviour and new example 10. Map:
  event-log-judge 20 000.
- 0 lines of product code by hand (the reference and the stubs lived only in a scratch worktree, removed).

### Run 20261008-173606 (processor ds, binary copy /tmp/morph-bin-P2 of MorphV2 bc311aa)

- 8 / 8 written, 0 burned; 10 requests; 133 097 input / 87 786 output tokens; $0.0722 executor; 619 s (10.3 min).
- Attempts: milestone-post 2 (v1 cut off with an unclosed fence at finish reason stop, 8 545 chars; r1 green);
  order-queue-judge 2 (v1 rejected at `== guard`: "session/orders_examples_test.go does not mention the example
  literal \"  third  \""; r1 green); every other card 1. No failure class (no card red after the run), no fix needed.
- Judges: 4 / 4 green at `== own` on the accepted code; judge defects 0; guard rejections 1 (above); neighbour-red 0.
- Verify on the run branch: `git status --short` empty; gofmt, `go vet ./...`, `go build ./...` clean;
  `go test -count=1 ./...` green, 63 example tests (P1 31 + P2 32).
- Read against §2.2 (defects recorded, not fixed by hand; no example covers them, so the acceptances stay green):
  1. session/machine.go `Apply`: a "result" with `ev.Result == nil` dereferences nil and panics (§2.2: count it as the
     zero Result) — stream.Parse never produces such an Event, so unreachable from the daemon's path;
  2. telegram/post.go `Post`: a nil `Do` with Token and ChatID set panics (§2.2: `telegram: post failed: no Do`) —
     every caller passes Do (cmd/morphd: `http.DefaultClient.Do`);
  3. eventlog/log.go `Append`: a failed file write returns `eventlog: open <path>: …` (§2.2: begins
     `eventlog: write `); Entry{} and Last unchanged as decided.
- Falsifiable claims of §9: 8/8 within one fix — held (no fix needed); 63 tests green — held.
- Lines by hand: 0. Largest slice 95 667 B before targets. Observability slip: the session posted one stray `run`
  🏁 line with "placeholder" numbers before the real one; corrected by the next post.
