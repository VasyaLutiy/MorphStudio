# TASK P6 — mcpserver

Phase P6 of `PLAN.md` ("Epics and phases"): Component `mcpserver` (PM Tools, Session Tools, MCP Mount) on the declared
SDK `github.com/modelcontextprotocol/go-sdk` v1.8.0. One code card and one judge card per Function, 6 cards. The cards
are the deck (`decks/P6/deck.json`), not this file. No stop after P6: P7 follows, then smoke stop 2.

## 1. Why this

- The PM reaches morphd from the laptop only through the remote MCP server (`/mcp/<project>`: status, order, pending,
  answer, interrupt, usage, restart, plan_load, continue, stop_check; `/mcp`: projects_list, project_create), and the
  VPS session reports itself only through `/mcp/<project>/session` (phase_done, wait_operator, milestone): flows 7 and
  the MCP binding row of the brief (`PLAN.md` rows "session per phase", "MCP binding"). Without P6, P7's `cmd/morphd`
  has nothing to mount behind `api.NewRouter`'s MCP slot, and smoke 2's "the same `status` over the remote MCP" fails.
- 1 of 17 packages, 3 of 27 Functions, 20 record examples (7 + 6 + 7; none added, sixteen rewritten to inline what they
  named or to pin a gap). The first and only package on the SDK (PLAN risk 2: "P6 is the SDK alone, so a red there is
  isolated").
- The tree after P5: 14 product packages, 159 example tests. After P6: 15 packages, 3 more code files, 3 more example
  test files, 179 example tests.
- Record size of the Component (YAML dump of the group): mcpserver 12 484 → 17 892 B, under the 30 KB limit.
- No open issue is labelled `P6-mcpserver` (checked before the preparation).

## 2. Contract

### 2.1. INPUT data shapes the code must build

- **One fixture file**: `tests/fixtures/plan/queue-3.json` (unchanged since P0) — one `queue.Plan` object as text:
  `approved "7b31dfe"`, three phases `P17 stop_after none` (no caps key), `P18 smoke` with `caps {claude_usd 12}`
  (no hours, no executor_usd), `P19 none` (no caps key). PM Tools example 3 decodes it with `json.Unmarshal` into
  `map[string]any` and sends it as the `plan_load` Arguments; the tool hands `Control.PlanLoad("demo", plan)` exactly
  `queue.Plan{Approved: "7b31dfe", Phases: [{P17 none {0 0 0}}, {P18 smoke {12 0 0}}, {P19 none {0 0 0}}]}` (no
  default filled: `queue.Load` fills them later in the daemon). The schema must accept the missing caps and caps
  fields, so they are optional input fields.
- **Every other literal lives in the record**: Control results (Status, Usage, QueueView, ProjectView, Question, Stop)
  are the record's literals, inlined in the examples; tool arguments are `map[string]any` literals; HTTP bodies are
  JSON-RPC literals.
- **Declared dependency** (`System.dependencies`: go-sdk v1.8.0, vendored at P0; `docs/deps/go-sdk.md` rides in every
  slice of the Component by `morph plan`). Calls the code makes, as in the digest: `mcp.NewServer(&mcp.Implementation{Name,
  Version}, nil)`, `mcp.AddTool[In, Out](s, &mcp.Tool{Name, Description}, handler)` with `Out = any` and
  `&mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text}}, IsError}`, `mcp.NewStreamableHTTPHandler(getServer,
  &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true})`. Tests add `mcp.NewInMemoryTransports`,
  `Server.Connect`, `mcp.NewClient(...).Connect`, `ClientSession.ListTools / CallTool / Close / InitializeResult`,
  `ServerSession.Wait`. Measured on v1.8.0 in the scratch tree: input failing the schema (a string for an int, a
  missing required field) comes back as a result with `IsError true` and text `validating "arguments": …`, never a
  call error; a tool name the server lacks is a call error (`calling "tools/call": unknown tool "status"`); `Out = any`
  with own Content leaves `StructuredContent` nil; `Out` a struct would add structuredContent and a key-sorted text.
- **Preconditions of the callees** (judges): control · `Code(err)` returns `(int, string)`; the second value is the code
  of the text "<code>: <err.Error()>" · PM Tools 4, Session Tools 3. control · `ErrNoQuestion` is
  `session.ErrNoQuestion`, text "no pending question" · PM Tools 4. control · `ErrBadToken` text "session token
  mismatch" · Session Tools 3. supervisor · `ErrPhaseMismatch` text "phase mismatch" (a non-sentinel for `Code`: 500
  "internal") · Session Tools 3. session · `OrderResult` marshals with omitempty (`{"queued":1}`, `{"sent":true}`) ·
  PM Tools 3. github · `Access` marshals `checked, reachable, push, reason, checked_at` · PM Tools 2, MCP Mount 2. SDK ·
  a stateless POST needs `Accept: application/json, text/event-stream` (else 400) and Content-Type application/json
  (else 415) · MCP Mount 1–6.
- **Harness skeletons** (in the judge instructions): PM/Session — a struct fake (`ptFake` / `stFake`) implementing the
  17 methods of `control.Control` with configurable results and a recorded call list; `ct, st :=
  mcp.NewInMemoryTransports()`; `ss, _ := s.Connect(ctx, st, nil)`; `cs, _ := mcp.NewClient(&mcp.Implementation{Name: "t",
  Version: "0"}, nil).Connect(ctx, ct, nil)`; `cs.CallTool(ctx, &mcp.CallToolParams{Name, Arguments: map[string]any{…}})`;
  `res.Content[0].(*mcp.TextContent).Text`; `cs.Close(); ss.Wait()`. Mount — fake `mmFake`; `httptest.NewRequest("POST",
  path, strings.NewReader(body))` with Content-Type and Accept, `h.ServeHTTP(httptest.NewRecorder(), req)`; never a
  listener.
- **Distinct markers**: the error texts "no_question: no pending question", "internal: disk full", "bad_session_token:
  session token mismatch", "bad_input: kind must be one of …", "bad_input: unknown milestone kind \"party\"" share no
  substring that a test could confuse; the 401 and 404 bodies are distinct.
- **Carried known risks and P6**: Control Contract's `registry.ErrExists ≠ control.ErrExists` — `project_create` passes
  any Control error to `Code` (a registry error would read "internal: registry: …", P7 translates); P5's Begin/Continue
  queue errors reach `continue` as "internal: queue: nothing to continue (state …)" (the P7 mapping stands).
- **Text an environment can change**: none; every text is exact. The SDK's own validation texts are not compared
  (IsError only).
- **Shared test helpers**: `internal/testhelp` (`Equal`), in every judge slice.

### 2.2. OUTPUT data shapes

The exported signatures are the record's, verbatim (`contour.yaml`, group mcpserver: "Declares in …"). Tables: example →
given → result are the record's examples (20), rendered into the judge cards by `morph plan`.

**Gaps decided here** (each also one line in `docs/DECISIONS.md`):

- PM Tools, Session Tools, MCP Mount · RECORD CHANGED: the examples inline what they named — the fake Control ("the fake
  Control of HTTP Handlers" → a fake of the test file), the Status of Control Contract example 1 and the Usage of
  example 7 (the literal and its exact JSON), "as PM Tools example 1" (the in-memory harness spelled out) · an example
  never refers to another Function's example (P5 lesson).
- PM Tools · RECORD CHANGED: every tool's output type is `any` and the result is the text alone (StructuredContent nil),
  so the Text is `json.Marshal` of the view in field order; examples 2 and 6 compare the whole text exactly · with
  `Out` a struct the SDK adds structuredContent and a key-sorted text, which contradicts "nothing structured" and
  example 6's field order.
- PM Tools · RECORD CHANGED: required inputs are name, language, repo_url, text, option, approved, phases, a phase's id;
  optional (omitempty) are force, a phase's stop_after and caps, and each caps field; plan_load hands the Control the
  queue.Plan of exactly the values given · a field without omitempty is required in the inferred schema, and the
  fixture's P17 has no caps (a `queue.Plan` input would fail it).
- PM Tools · RECORD CHANGED: example 3 pins the QueueView the fake returns and its exact text, and the whole recorded
  plan; example 5 the order of the recorded calls; example 6 the fake's project list and the exact projects_list text;
  example 4 runs on project "beta" (a constant the code must not hard-code) · "the fake's views", "the QueueView JSON
  the fake returned" were not literals.
- PM Tools · RECORD CHANGED: example 7 adds a valid `answer {"option": 2}` after the refused one (→ {"ok":true},
  Answer("demo", 2) only) · its every answer was "an error" — tautological on a server without tools (P5 lesson).
- PM Tools · projects_list turns a nil list into `[]` (probe: `{"projects":[]}`) · the HTTP Handlers decision of P5.
- Session Tools · RECORD CHANGED: required inputs phase, kind, reason, headline; optional next (absent ""), issue_url,
  numbers; example 2 adds `phase_done {"phase": "P19"}` → PhaseDone("demo", "id-2", "P19", "") · the last phase has no
  next; example 4's second call has no issue_url and must reach the kind check, not the schema.
- Session Tools · RECORD CHANGED: example 3 runs on SessionServer(c, "beta", "id-9") and pins PhaseDone("beta",
  "id-9", …); examples 5 and 6 pin the whole Milestone · the project and the token are not constants.
- Session Tools · the unknown milestone kind is quoted by `%q` (probe: `a"b` → `"a\"b"`); the kind checks are exact
  (probe: "Smoke" refused) and come before the Control (the fake not called); every kind of MilestoneKinds and of the
  four wait kinds is accepted (probe) · the record's text.
- MCP Mount · RECORD CHANGED: the token is checked before the project (example 5 adds POST /mcp/nope without a token
  → 401); the scheme is "Bearer " ignoring case with the rest compared whole by ConstantTimeCompare (no trim); an
  empty expected token refuses; both error bodies carry Content-Type application/json · example 4's /mcp/other/session
  (an unknown project answering 401) already implied the order; the Router's rules of P5.
- MCP Mount · probe variants: a session token known for an unknown project → 404; another project's session with its
  own token reaches SessionServer(c, project, token); a known project's PM path under another API token; GET /mcp and
  DELETE /mcp/demo/session → 405 · the record's text.
- Imports · RECORD CHANGED: mcpserver adds errors, slices and names the SDK path
  `github.com/modelcontextprotocol/go-sdk/mcp` · a code card's allowed imports are its record list (P3 lesson);
  `== imports` allows exactly it.
- Guard · `decks/tools/guard.mjs` test mode now allows the direct requirements of `go.mod` (only the go-sdk) besides the
  standard library and the module · the P0 copy of the V2 template refused every external import in a test file, so
  the PM/Session judges (which must import `mcp` for the in-memory transports) were red at `== guard` whatever they
  wrote — measured on the derived judge file in the scratch tree. Upstream MorphV2 `templates/go/decks/tools/guard.mjs`
  has the same gap (not edited).
- Map · budgets from the expected answers: pm-tools-judge 28 000 kept (a 17-method fake + 7 examples ≈ 17 KB, the
  probe derived from it is 16.9 KB), session-tools-judge 18 000 → 20 000 (≈ 11 KB + the fake), mcp-mount-judge 20 000 →
  22 000 (≈ 13 KB); code cards kept (pm-tools 16 000, session-tools 10 000, mcp-mount 10 000; the scratch reference is
  5.2, 2.2 and 2.2 KB). Slices: session-tools-judge adds queue/queue.go, session/orders.go; mcp-mount-judge adds
  queue/queue.go, github/access.go, session/orders.go (the fake's signatures and the Status literal).
- Map · three test files in one package over three generations: fakes `ptFake` / `stFake` / `mmFake`, helper prefixes
  `pt` / `st` / `mm`; the instructions name every arity (Connect, ListTools, CallTool return two values) and every short
  literal the guard wants.

### 2.3. Names

- Code (one file per card, code only — no smoke test target): `mcpserver/pm.go`, `mcpserver/session.go`,
  `mcpserver/mount.go`; package `mcpserver` of module `morphstudio` (layer already in `decks/tools/layers.json`: control,
  queue, github, eventlog, session, supervisor).
- Judge files, one test per record example, in example order: `mcpserver/pm_examples_test.go` (`TestPMToolsExample1..7`),
  `mcpserver/session_examples_test.go` (`TestSessionToolsExample1..6`), `mcpserver/mount_examples_test.go`
  (`TestMCPMountExample1..7`).
- Probes (data, inlined into the code cards' acceptances, removed after the step): `decks/P6/parts/_<card>_probe_test.go`,
  tests `TestProbe<Function>Example<N>`; helpers prefixed `pPM`, `pST`, `pMM`.

### 2.4. What must not break

Frozen byte for byte (every acceptance's `== frozen`): `contour.yaml`, `morph-map.json`, `docs`, `decks`,
`tests/fixtures`, `go.mod`, `go.sum`, `internal`, `vendor`, and the code of P1–P5: `stream`, `runner`, `queue`,
`session`, `eventlog`, `telegram`, `claude`, `github`, `bootstrap`, `registry`, `gitrules`, `control`, `supervisor`, `api`.
The suite before P6: 159 example tests; after: 179. No ripple: every target is a new file in a new package.

## 3. Acceptance

Built by `morph plan --checks decks/P6/checks.json` (Go profile, vendor mode), every stage printing `== <stage>`.

- Code card: `== build` (overlay hides the generation's sibling targets) → `== vet` → `== gofmt` → `== guard` (layers;
  the declared module allowed) → `== probe` → `== imports` (allow-list = the record's: `context, crypto/subtle,
  encoding/json, errors, fmt, net/http, slices, strings, github.com/modelcontextprotocol/go-sdk/mcp, morphstudio/control,
  morphstudio/queue` — no os, no os/exec, no time) → `== clock` (no `time.Now`, no `time.Since`) → `== net` (no
  `http.DefaultClient`, `http.Get`, `http.Post`, `http.PostForm`, `http.Head`, `http.ListenAndServe`,
  `http.ListenAndServeTLS`) → `== full` → `== frozen` → no untracked file.
- Judge card: `== vet` → `== gofmt` → `== guard <file>` (min = examples, max = min + 6: pm 7..13, session 6..12, mount
  7..13; the literals: every `Test<Function>ExampleN` plus the distinct values of the examples, in `checks.json`, none
  holding a quote or a backslash) → `== own` → `== full` → `== frozen` → no untracked file.
- Measured on a scratch worktree (`/tmp/p6-scratch`, the data committed there, each card run in the tree of its
  generation, removed after): every code acceptance red at `== probe` on stubs (servers with no tools, an empty mux,
  `MilestoneKinds` empty, ServerName "stub"), one `--- FAIL: TestProbe…ExampleN` per example, 20 of 20 (pm 7, session 6,
  mount 7), each with an `example N …` line and a first-difference line, no panic; judges red at `== guard` on a
  one-test stub (38, 30, 39 guard lines); all 6 green on a reference written in the scratch tree only (judges green with
  test files derived from the probes); chains warm ≤ 9.3 s green and red (pm-tools); the first on a cold Go cache
  53.4 s (mcp-mount).

## 4. Constraints

- Module `morphstudio`, Go 1.25, `GOFLAGS=-mod=vendor GOPROXY=off`; the SDK only from `vendor/`.
- Stubs and helpers only from `morphstudio/internal/testhelp`; no test opens a listener or dials (in-memory transports
  and httptest recorders only); a test writes nothing.
- `mcpserver` reads no clock and holds no state between requests (a server per request in the mount); tokens never
  appear in a result or an error text.
- A judge writes only its test file; a code card writes only its code file (no test target, so no smoke cap).
- Generations (measured by the cut): [pm-tools], [pm-tools-judge, session-tools], [mcp-mount, session-tools-judge],
  [mcp-mount-judge]. `mcpserver/session.go` and `mcpserver/pm_examples_test.go`, `mcpserver/mount.go` and
  `mcpserver/session_examples_test.go` are written in the same generation in the same package: the overlay hides each
  from the other's acceptance; neither is in the other's slice.

## 7. Out of scope

- Mounting `Handler` into `api.NewRouter`, the listener, the token files and the per-session token store (P7:
  `cmd/morphd`, `daemon`); `Tokens.Session` and `known` are the daemon's functions.
- The `--mcp-config` file the session gets (P3 Launch Args, done).
- Sessions over SSE / a stateful server, OAuth, resources and prompts; a tool output schema.
- Mapping `registry` and `queue` errors to control's sentinels (P7).
- Any record example beyond the 20 (the probes' extra variants follow the behaviour text; they are not examples).

## 8. How to run

```
node /tmp/morph-bin-P6/dist/cli.js plan --root . --spec contour.yaml --map morph-map.json \
  --component mcpserver --judge --checks decks/P6/checks.json --out decks/P6/deck.json
python3 decks/tools/scale_tokens.py decks/P6/deck.json 3
node /tmp/morph-bin-P6/dist/cli.js deck check --root . --deck decks/P6/deck.json        # errors 0
node /tmp/morph-bin-P6/dist/cli.js run --root . --deck decks/P6/deck.json --processor ds --deadline 2400
```

The cut holds exactly the 6 P6 cards: no filter script is needed. `/tmp/morph-bin-P6` is the copy of the MorphV2
binary made for this phase (MorphV2 bc311aa; `dist` copied, `package.json`, `node_modules`, `templates` linked).

## 9. Pre-registration

- 6 cards, 4 generations [1, 2, 2, 1]; processor `ds`, maxTokens ×3: pm-tools 48 000, session-tools 30 000, mcp-mount
  30 000, pm-tools-judge 84 000, session-tools-judge 60 000, mcp-mount-judge 66 000 (sum 318 000).
- Bill: forecast ≈ $0.044 (P1–P5 actuals $0.3086 for 42 cards, ≈ $0.0073 a card, × 6). Ceiling if every card used its
  whole budget once: 318 000 output × $1.20 / M = $0.38 + ≈ 50 000 input × $0.30 / M = $0.02 → $0.40. Cap $5; gate
  ≤ $1.
- Largest slice at the cut: session-tools-judge, 25 872 B of existing files (≈ 35 KB with the written pm.go and
  session.go); no slice near 200 KB.
- Expected regenerations: pm-tools (an `Out` struct instead of `any`: "example 2 status" with a key-sorted text and
  Structured true; `queue.Plan` as the plan_load input: "example 3 plan_load" with `validating "arguments"`); session-tools
  (next or issue_url required: "example 2 phase_done P19"); mcp-mount (the project checked before the token: "example 5
  POST /mcp/nope without a token"; the empty token); the judges (the SDK's two-value Connect; a guard literal; the 17-
  method fake's signatures).
- Falsifiable: 6/6 written within one fix; 179 example tests green on `go test ./...`; no card over 1 regeneration.

## 10. What to record

The `docs/MEASURE.md` row and §11: attempts per card, first red per burned variant, neighbour-red, judge defects,
guard rejections, lines by hand, max slice bytes, minutes, $.

## 11. Actual

### Preparation (before the gate)

- Orchestrator: Opus 5.5 agent (fresh context), 73 tool calls, 228k tokens, 17 min (measured by the session from the
  agent's run); the preparation facts in §3, §9 and `docs/DECISIONS.md` 08.10 · P6.
- `morph plan` exit 0 (a fresh re-cut byte-identical to the committed deck); `morph deck check` errors 0, warnings 0,
  hazards 0; generations [1, 2, 2, 1]; the cut holds exactly the 6 phase cards.
- Largest slice at the cut: session-tools-judge, 25 872 B of existing files.
- Probes red per example on stubs: 20 / 20 (pm 7, session 6, mount 7); judges red at `== guard` on a one-test stub.
- Acceptance chains: the slowest 53.4 s (mcp-mount, cold Go cache).
- Mutants: 30 on a scratch reference (pm 16, session 7, mount 7), each under `timeout 120`, 30 / 30 killed, ≈ 1.2 min.
- Deck tool changed: `decks/tools/guard.mjs` lets a test file import go.mod's direct requirements (only the go-sdk),
  so the PM/Session judges can use the SDK's in-memory transports (accepted by the session; posted to the operator).
- 0 lines of product code by hand.

### Run 20261008-200807 (processor ds, binary copy /tmp/morph-bin-P6 of MorphV2 bc311aa)

- 5 / 6 written; 8 requests; 99 757 input / 74 910 output tokens; $0.0580 executor; ≈ 9.3 min.
- pm-tools-judge red ×3 at `== own`, every answer whole (finish stop): v1 assigned a CallTool result to a
  *mcp.ListToolsResult variable (build); r1 expected example 6's call list to be [CreateProject] only (the record said
  "the fake saw CreateProject", while projects_list was called first); r2 wrote an unqualified `Milestone` (build).
  Class data; the code card pm-tools is right.
- Every other card green on its first attempt.

### Fix run 20261008-201843 (pm-tools-judge only, decks/P6/deck-fix.json)

- One re-cut of the record wording (example 6: the whole call list Projects() then CreateProject(…); example 1's fake
  with package-qualified control types). 0 / 1 written; 3 requests; $0.0406; ≈ 7.3 min.
- Red ×3: the single-card re-cut kept the generation-2 overlay that blanks mcpserver/session.go, and with mount.go
  accepted every build said "mount.go:49: undefined: SessionServer"; v1 and r2 also repeated the variable mix-up.
  The stub check before this run counted the guard lines and missed the vet line.
- Emergency stop by AUTONOMY "Failure" → the debt.

### Debt (pm-tools-judge, Claude Fable 5.1, effort xhigh)

- Debt deck decks/P6/deck-debt.json = deck-fix with the overlay of the accepted session.go emptied
  (decks/P6/debt_overlay.py); red on the tree without the target only at its absence.
- `morph card --md` brief + the six attempts' reasons; `claude -p --model claude-fable-5-1 --effort xhigh`: 13 turns,
  $2.4027, 2.8 min, acceptance green on its first run, 7 tests; `morph accept --commit` green → a1d948a with
  `Morph-Debt: true`.
- Mutations of pm.go against the new tests: error result not IsError killed; Restart force dropped killed; a nil
  project list not rendered as [] survived (no example pins a nil list; known risk); one build-only mutant not counted.

### Verify on the run branch

- `git status --short` empty; gofmt, `go vet ./...`, `go build ./...` clean; `go test -count=1 ./...` green, 179
  example tests (P1 31 + P2 32 + P3 29 + P4 35 + P5 32 + P6 20).
- Read against §2.2 and the record (a fresh read-only agent, 76k tokens/12 calls/1 min): no behaviour defect. pm.go
  exports `type Empty struct{}` beyond the record's declared names (unpinned). Test weaknesses: Session Tools 2 does not
  check the call-list length; Session Tools 3 does not pin the second PhaseDone call; `stText` does not check one
  Content entry; tool names compared in the SDK's order (stricter than "sorted"); the empty expected token, a
  lower-case "bearer", a session token on an unknown project and a wrong-case kind are pinned by probes only.
