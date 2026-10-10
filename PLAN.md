# PLAN.md — morphd (Morph Studio, first goal): epics, phases, the Zero Contour

> Designed by `morph-architect` from `docs/START.md` (operator-approved 08.10) and `docs/DECISIONS.md`; approvals by
> commit hash: 98b1527 (P0–P7), cc46ed1 (P7b), P7c approved in advance (10.10), 50a83fd (P8a/P8b). This file is the
> clean rewrite of 10.10 (no design change): it holds the current contract and each phase's outcome, never its story —
> the story is in `docs/MEASURE.md`, `docs/DECISIONS.md` and the TASK files' §11. The record is `contour.yaml`, the map
> `morph-map.json`; open questions `docs/QUESTIONS.md`. After approval only phase data changes without a new approval.

**State (10.10):** P0–P8a built and merged (main 878c44b, 230 example tests green, 17 packages); smoke stops 1 and 2
green, the D4 check green, `morphd` runs as a systemd unit on the VPS. **Open: P8b, then Smoke stop 3.**

## Context

- The goal (operator 08.10): "Убрать ssh flow для взаимодействия с claude внутри VPS … сделать это профессионально на
  Golang." `morphd` is a Go daemon on the user's VPS that drives the claude CLI session of a Morph-built project in
  place of the PM's ssh + tmux flow. Users: the PM (Claude Code on the operator's laptop, through the remote MCP) and
  the operator (curl over the HTTP API). One user = one VPS = one morphd; many projects per user.
- The operator's answers that bind the design: Go, stdlib first, one declared module (the official Go MCP SDK,
  vendored at P0); the Morph-written Go skeleton (P15L gocrud) is the base repository; API token in a header, Caddy
  terminates TLS, morphd on localhost only; Telegram milestones by morphd itself (Bot API, token from `.env`), no
  `tg.sh`; the MCP endpoint carries the project (`/mcp/<project>`, user-level `/mcp`, the session's own
  `/mcp/<project>/session`); the GitHub fine-grained PAT reaches morphd only by `PUT /projects/{p}/github-token`;
  phase sessions fresh per phase, never `--resume` across phases; caps per phase Claude $30 list price and 3 h; the PM
  loads the whole approved queue (`plan_load`), morphd runs between stops and waits at stops (`continue`); one active
  session per VPS with a max-parallel setting (default 1); tests never reach the network or a real claude; executor
  cap $5 per phase, $35 for the stretch; phases of 8–12 cards.
- Builder: MorphV2 (`morph plan --spec … --judge --checks` → `morph deck check` → gate → `morph run --processor ds`),
  a binary copy outside the repository; the orchestrator agent with the `morph-orchestrator` skill prepares each phase
  (primer, scout, TASK, checks, probes); the architect only for a Start or new Components (DECISIONS 10.10).
- What the operator gets at the end: a daemon binary `morphd` (18 Components in 17 packages), an HTTP API of 16
  routes (15 under a bearer token + `/healthz`), three MCP endpoints with 17 tools after P8b (PM 11, user 3, session
  3), a JSONL event log per session, Telegram milestones, and 237 example tests (one per record example). **Not
  production**: no browser UI, no multi-user, no provisioning — the framework the PM and the operator drive from
  outside (TLS, the host name and the systemd unit are the PM's VPS data, `docs/RUNBOOK_VPS.md`).

## Brief → design (traceability)

| scenario of the brief | phase | Component · Function | examples carrying its values |
|---|---|---|---|
| S1 `status` → `{state: "busy", phase: "P18", minutes: 37, cost_usd: 1.92, five_hour: 22, seven_day: 60}` | P4, P7 | control · Control Contract; daemon · Daemon Core | Control Contract 1 (the exact JSON); Daemon Core 2 (busy, P17); Pump 1 (22 / 60 from the probe's rate_limit_event) |
| S2 `order` while the session waits → `{sent: true}`; while busy → `{queued: 1}`, sent after `result` | P2, P7 | session · Order Queue, Session Machine; daemon · Daemon Core | Order Queue 1–2; Session Machine 5 (the queue drains on the result); Daemon Core 3 |
| S3 the session asks "Gate waiver" with 2 options → TG, `pending` shows it, `answer {option: 1}` → the session continues | P1, P2, P7 | stream · Parse Event, Encode Lines; session · Session Machine, Order Queue; pump · Pump | Parse Event 6 (probe line 27); Encode Lines 6–7 (the answer line equals probe line 28); Session Machine 3; Order Queue 4–5; Pump 2 (TG `ask`, `answer` 2 → "Option B") |
| S4 `phase_done {phase: "P17", next: "P18"}`, push checked → a fresh session at once, TG "🚀 P18" | P4, P5, P7 | gitrules · Phase End Check; supervisor · Phase Loop; daemon · Daemon Core | Phase End Check 1; Phase Loop 5 (`kill`, `start_check P18`); Daemon Core 5 (the second spawn, "P18 started") |
| S5 `phase_done`, main not pushed → no new session, TG "🛑 P17 not pushed", state waiting | P4, P5, P7 | gitrules · Phase End Check; supervisor · Phase Loop; daemon · Daemon Core | Phase End Check 2; Phase Loop 7; Daemon Core 6 (no spawn, the stop post, `continue` re-checks) |
| flow 1 look: `tmux capture-pane` → `GET events?since=N`, `GET status` | P2, P5 | eventlog · Event Log; api · HTTP Handlers | Event Log 2–5 (seq, since); HTTP Handlers 1, 5 |
| flow 3 order only when idle, else queue | P2 | session · Order Queue | Order Queue 1–2 |
| flow 4 dialog answer by option; interrupt only explicit; every other tool allowed without a human | P1, P2 | stream · Encode Lines; session · Session Machine, Order Queue | Encode Lines 5 (Allow); Session Machine 4 (auto-allow Bash); Order Queue 6–7 (Interrupt) |
| flow 5 restart: refuse while busy unless force; git pull ff-only; new process, new session id; events → log file | P4, P5, P7 | gitrules · Phase Start Check; supervisor · Phase Loop; daemon · Daemon Core | Phase Start Check 3 (pull ff-only); Phase Loop 9 (Restart); Daemon Core 4 (ErrBusy, then force) |
| flow 7 the session tells the daemon (`phase_done` / `wait_operator`); the daemon acts after the push check; exit seen at once → restart ≤ N per hour; limit → pause until resetsAt; stall 30 min → TG, one nudge | P3, P5, P6 | claude · Launch Args, Process; supervisor · Phase Loop, Runtime Guard; mcpserver · Session Tools | Launch Args 2 (`--resume`); Process 1–3 (exit seen on a channel); Runtime Guard 1–2 (3 per hour), 5–6 (pause, resume at reset), 7 (one nudge, wall clock cap); Session Tools 2–5 |
| flow 6 `GET usage`: five_hour / seven_day + resets + the session's cost sum; TG alert at 50% weekly | P1, P4, P5 | stream · Parse Event; control · Control Contract; supervisor · Runtime Guard | Parse Event 3 (0.22 / 0.6, resetsAt); Control Contract 7 (the Usage JSON); Runtime Guard 4 (the 60% alert, once) |
| protocol: start argv, initialize, user line, interrupt, AskUserQuestion answer, rate_limit_event, result fields | P1, P3 | stream · Parse Event, Encode Lines; claude · Launch Args | every Parse Event / Encode Lines example cites a probe line; Launch Args 1 (the argv of START "Protocol facts" plus `--dangerously-skip-permissions`) |
| session per phase: fresh process, first message `/morph-orchestrator <phase>`, `plan_load {approved, phases: id, stop after, caps}`, stops enforced by morphd, `stop_check` / `continue` | P1, P5, P6 | queue · Phase Queue; supervisor · Phase Loop; mcpserver · PM Tools | Phase Queue 1, 4–5 (stop after smoke, end); Phase Loop 2 (`first_line`), 8 (stop, continue); PM Tools 1, 3, 5 |
| caps: Claude `--max-budget-usd`, wall clock; the stretch total | P3, P5 | claude · Launch Args; supervisor · Runtime Guard | Launch Args 1–2 (`--max-budget-usd 30`, `12.5`); Runtime Guard 3 (stretch $30), 7 (3 h) |
| Claude auto-memory in phase sessions: off, 100% (operator 08.10) | P3 | claude · Launch Args | Launch Args 4 (`Env` is exactly `["CLAUDE_CODE_DISABLE_AUTO_MEMORY=1"]`, whatever the Launch) |
| git of a phase: morphd never commits/merges/pushes; start: fetch; equal → fresh; ahead → resume; diverged → stop; end: pushed, clean, MEASURE row, no `morph/*` unmerged | P4 | gitrules · Phase Start Check, Phase End Check | Phase Start Check 1–8; Phase End Check 1–7 (every git call scripted, none run) |
| rule: token header, Caddy TLS, localhost only | P5, P7 | api · Router; cmd · Config And Main | Router 2–4 (401 bodies, an empty token refuses); Config And Main (`127.0.0.1:<port>`; Requirement "Bearer Token On Every Route") |
| rule: Telegram by morphd, `[<project>]` prefix, kinds and icons of AUTONOMY | P2 | telegram · Milestone Post | Milestone Post 1–5 (the exact text and the urlencoded body) |
| projects: registry, one folder each, `project` in every call, per-project queue/state/log/caps/sessions; max-parallel default 1 | P4, P7 | registry · Project Registry, Secret Files; daemon · Daemon Core | Project Registry 1–7; Daemon Core 9 (the second project waits) |
| `project_create {name, language, repo_url}`: morph init, first push | P3, P7 | bootstrap · Project Create; daemon · Daemon Core | Project Create 1–7 (`morph init --root <dir> …`, `--module` for Go; FirstPush at the token PUT — Q8) |
| MCP binding: `/mcp/<project>` tools without `project`; user-level `/mcp` with the registry tools; the session's `/mcp/<project>/session` with its own token via `--mcp-config` | P3, P6 | claude · Launch Args; mcpserver · PM Tools, Session Tools, MCP Mount | Launch Args 5–6 (the mcp-config file); PM Tools 1, 6; Session Tools 1; MCP Mount 1–6 (tokens per path) |
| GitHub token: HTTP only, stored mode 600, never returned; status reports reachable + push access with a plain reason | P3, P4, P5 | github · Repo Access; registry · Secret Files; api · HTTP Handlers | Repo Access 4–8 (401/403/404 reasons); Secret Files 1, 4 (0600); HTTP Handlers 8 (the token appears nowhere in the body) |
| D1: session ids the claude CLI accepts (2.1.294: "Invalid session ID. Must be a valid UUID."): RFC 4122 v4, lower-case 8-4-4-4-12 | P7b | cmd · Config And Main | Config And Main 6 (`UUIDv4` of the probe's bytes → `967e8f4d-a2eb-4aa6-968b-b9054a1c3d7e`, the zero and all-ff bytes), 7 (`newID` matches the v4 regexp, two differ) |
| D2: an exit before `phase_done` / `wait_operator` is a crash: logged with its stderr, counted whether the restart is fresh or `--resume`, ≤ N per hour, then a `crash` stop with a 🛑 naming the code and the first stderr line; no restart storm | P7b | supervisor · Phase Loop, Runtime Guard; daemon · Daemon Core; pump · Pump; control · Control Contract | Runtime Guard 1, 2, 8 (an exit while waiting is nothing), 9 (the first stderr line); Phase Loop 1, 2, 9 (what resets the count); Pump 8 (a real `sh` claude dying with exit 1: exactly 4 spawns, the `exit` entry, the stop), 9; Daemon Core 11 (a Spawn that always fails: 4 attempts, `continue` gives 4 more); Control Contract 10 (`last_exit` JSON) |
| D3: `five_hour` / `seven_day` null until the first `rate_limit_event` (`limits_at` null too) | P7b | control · Control Contract; daemon · Daemon Core | Control Contract 8, 9 (the exact null bodies); Daemon Core 1 (`LimitsUnknown`, `Usage{}` marshals the five nulls); Pump 1 (22 / 60 once seen) |
| D4: a start check "resume" on a loop with no session yet spawns a FRESH session (`--session-id <new id>`, the first line), never `--resume` of an id claude has not seen; a non-empty SessionID still resumes | P7c | supervisor · Phase Loop | Phase Loop 3 (SessionID "" → spawn `id-7` Resume false, "P18 resumed on a fresh session"), 11 (after Restart, "on branch morph/20261008-083312"), 10 (SessionID "id-1" → Resume true) |
| harness 2 (operator 10.10): adopt an existing repository — `project_adopt {name, dir, repo_url, language}` on `/mcp` and `POST /projects/adopt`; dir must be a git repo on `main` with `origin` = repo_url and a `contour.yaml`; registered without init, commit or push; only the credential helper is added; refusals → 400 `bad_input`, a known name → 409 `exists` | P8a, P8b | bootstrap · Project Adopt; control · Adopt Contract; api · HTTP Handlers, Router; mcpserver · PM Tools; daemon · Daemon Core | Project Adopt 1 (four git calls, no init/commit/push), 2–6 (every refusal text), 7 (ssh origin = https repo_url); Adopt Contract 1–3; HTTP Handlers 9 (201 / 400 / 409 / 500); Router 8; PM Tools 9; Daemon Core 14 |
| harness 3 (operator 10.10): session progress over MCP — `events {since, limit}` on `/mcp/<project>` returns a compact, bounded view (seq, dir, type/subtype, text ≤ 200 runes, tool names, result cost / terminal_reason, exit code / stderr); limit absent → 50 | P8a, P8b | control · Event View; mcpserver · PM Tools | Event View 1–6 (the five probe lines as exact JSON, tool names, the 327-rune cut, question/interrupt/aborted, exit entry, empty); PM Tools 8 |
| harness 5 (operator 10.10): the last turn's cost is not lost — after `phase_done` the daemon waits for that turn's `result` (≤ `MORPHD_DONE_WAIT` s, default 30) before the end check, the kill and the next phase; an exit after `phase_done` is not a crash | P8b | daemon · Daemon Core; pump · Pump; cmd · Config And Main | Pump 10 (`StretchUSD 0.005775230000000001`, "turn ended" before "P18 started"), 11; Daemon Core 12 (the 30 s deadline by `Tick`), 13 (a pending done of a replaced process is dropped); Config And Main 2–4 (`DoneWait` 30 / 45 / 0, "not a number: 1m") |
| harness 4 (operator 10.10): the regulation a morphd session reads — data, not code | — | `docs/AUTONOMY_MORPHD.md` (`docs/AUTONOMY.md` keeps the laptop/tmux text) | — |

Every rule of the brief is an example or a Guardrail: Guardrails "Injected Clock And Ids", "No Network In Tests",
"No Shell Outside Runner", "Secrets Never Logged"; Requirements "Claude Behind An Interface", "Bearer Token On Every
Route", "The Session Reports Only Itself". Every not-build item is in "Out of scope".

## What is known

- **Language profile**: MorphV2 `go` (P15): code `{component}/{name}.go`, judges `<name>_examples_test.go` next to the
  code, `go build ./...` + `gofmt -l`, `go vet ./...`, the Go guard (`decks/tools/guard.mjs` = V2's goguard, layers from
  `decks/tools/layers.json`, direct imports by `go list`, the declared module allowed per card since P19b),
  `go test -count=1 ./<package>/` then `./...`; helpers `internal/testhelp` (`Equal`, `WriteFile`, `ProbeLine`,
  `ProbeOut`). Dependencies: declared in the record (`System.dependencies`, `Component.uses`), vendored at P0,
  `GOFLAGS=-mod=vendor GOPROXY=off` in every acceptance. Each code card gets per-package import rules as `extra` lines
  of the phase's `checks.json` (the table is in "Scaffold"). Go 1.25 (`/usr/bin/go` on the laptop is 1.22: use the
  toolchain under `~/go/pkg/mod/golang.org/toolchain@…go1.25.0…`).
- **Comparable project**: P15L gocrud in this very repository: 14 cards, 5 generations, 14/14 written, $0.0852, 7.1 min
  on `deepseek/deepseek-v4.1-flash`. morphd's cards are larger (daemon-core ≈ 20 KB of answer): estimates are 2–4× P15L.
- **The claude protocol** is measured, not assumed: `docs/probe/stream-json-2.1.294.jsonl` (CLI 2.1.294, 37 lines,
  4 turns, $0.0064) is copied at P0 to `tests/fixtures/stream/probe.jsonl` and every stream/session example names the
  line it uses. Probe 2 (`stream-json-2.1.294-skip.jsonl`): under `--dangerously-skip-permissions` AskUserQuestion still
  arrives as `can_use_tool`; other tools need no answer.
- **The SDK**: `github.com/modelcontextprotocol/go-sdk` v1.8.0 (module proxy 08.10: the latest tag, 2026-09-04; `go
  1.25.0`; direct requirements jwt/v5, go-cmp, jsonschema-go, segmentio/encoding, uritemplate/v3, x/oauth2, x/time,
  x/tools). Verified in its source: `mcp.NewServer`, generic `mcp.AddTool[In, Out]`, `ToolHandlerFor` = `func(ctx,
  *CallToolRequest, In) (*CallToolResult, Out, error)`, `mcp.NewStreamableHTTPHandler(getServer func(*http.Request)
  *Server, *StreamableHTTPOptions{Stateless, JSONResponse})`, `mcp.NewInMemoryTransports()`, `Server.Connect`,
  `Client.Connect`, `ClientSession.ListTools / CallTool`; a stateless POST needs `Accept: application/json,
  text/event-stream` (else 400); the handler refuses a non-local `Host` ("Forbidden: invalid Host header"), so Caddy
  sets `header_up Host {upstream_hostport}`. Every SDK version needs Go ≥ 1.23, v1.6.1+ needs Go 1.25 (Q1).
- **MorphV2 features the re-cut phases rely on** (binary copy `/tmp/morphbin`, wrapper `~/bin/morphv2`): `morph plan
  --only` re-cut over the current tree (P20); a `--only` subset is one transaction with `_stubs/` built per generation
  by `deck check` (P21b/P21c); `morph gate` (P22b); a red group retries only the blamed cards (P24, issue #16); group
  `morph accept --from-run --pick` (P24b); Go test blame on indented `file:line` (issue #19); scout multi-action turns
  (aec2e80); re-cut stubs carry record names only (#22).
- **Risks of this design for autonomous building and how the cut handles each**:
  1. *The daemon glue* (daemon-core, pump): goroutines, a mutex, fakes of five kinds. Kept pure everywhere else
     (session and supervisor are state machines with literal examples); the daemon examples use only fakes declared in
     non-test files (`runner.Fake`, `claude.Fake`, recorder funcs); P7 holds nothing else but `cmd`.
  2. *The MCP SDK API at v1.8.0*: the mount examples compare decoded JSON-RPC fields, never bytes; the digest
     `docs/deps/go-sdk.md` is written at P0 from the vendored source; P6 is the SDK alone so a red there is isolated.
  3. *A real claude at a rate limit was never observed*: the pause rule is pinned on `utilization >= 1` (Runtime Guard
     5); the first real limit is a DECISIONS line, not a redesign.
  4. *Process tests* (claude · Process, Pump 8) run stub shell scripts: Linux only, no network.
  5. *Two Functions of one package in one phase* serialize generations (Go builds a package whole): every package has
     ≤ 3 Functions; a re-cut chain control → supervisor → daemon → cmd is one package behind another.
  6. *A re-cut that changes an exported or a private name used outside the subset* breaks the build of files no card
     owns: every re-cut phase checks the callers of the re-cut files' symbols before the gate and freezes every package
     outside the subset in `checks.json`.

## Epics and phases

| # | epic | phases | why here |
|---|---|---|---|
| E1 | wire & state leaves | P1–P2 | pure packages over the probe; everything above parses and holds the session through them |
| E2 | process, git, projects | P3–P4 | the claude process, GitHub access, project create; then the registry, the git rules of a phase and the control contract the servers share |
| E3 | loop & surfaces | P5–P6 | the supervisor's state machines and the HTTP API; then the MCP servers on the SDK alone (an isolated red) |
| E4 | daemon assembly | P7 | the glue of every package and `cmd/morphd`; last because it imports all |
| E5 | smoke-2 fixes | P7b, P7c | D1 (UUID ids), D2 (visible crash, per-hour restart cap), D3 (null limits), D4 (resume with no session) |
| E6 | PM MCP harness | P8a, P8b | the switch to morphd as the PM's working channel: adopt an existing repo, compact `events`, the last turn's cost |

Milestones: **Smoke stop 1** after P3 (the protocol on a real claude); **Smoke stop 2, final** after P7 (re-run after
P7b, the D4 check after P7c) — the framework is done; **cutover** (10.10): systemd `morphd.service` on the VPS behind
Caddy at `themorph.stream`, the cron watchdog off; **Smoke stop 3** after P8b — the harness is done and the PM may move
phases to morphd (DECISIONS 10.10: not before a minimal PM MCP).

Layer Guardrail checked by `guard.mjs` on every card (`decks/tools/layers.json`, direct imports): stream, runner,
queue, eventlog, telegram, github, claude import no module package; session → stream; bootstrap → runner, github;
registry → queue; gitrules → runner; control → queue, github, eventlog, session, stream; supervisor → queue,
gitrules, control, stream; api and mcpserver → the control side; daemon → all but api/mcpserver; `cmd/morphd` → all.
`os/exec` only in runner and claude; the clock and randomness only in `cmd/morphd`; HTTP clients only in `cmd/morphd`.

Hand data per phase, before the run: `docs/TASK_<phase>_<components>.md` (by `docs/TASK_TEMPLATE.md`),
`decks/<phase>/checks.json` (frozen packages, `extra` import rules, literal guards), probes
`decks/<phase>/parts/_<card>_probe_test.go` (red per example on stubs), `decks/<phase>/_stubs/` for subsets that add
or re-cut files, fixtures under `tests/fixtures/`; record and map edits by the orchestrator when the change stays
inside existing Functions.

### Phases

$ est. = the design estimate on `ds`; actual = executor dollars of the provider's bill (MEASURE). Every phase cap $5.

**E1–E2: leaves, process, git, projects**

| phase | Component | Functions (code + judge each) | cards / gens | oracle and fixtures | by hand before the run | $ est. / actual | risk → treatment | status |
|---|---|---|---|---|---|---|---|---|
| **P0** | scaffold (hand data) | — | 0 | `GOFLAGS=-mod=vendor GOPROXY=off go build ./...` on the empty tree | `go.mod` (Go 1.25, SDK v1.8.0), `vendor/`, testhelp, guard/layers, fixtures, `docs/deps/go-sdk.md`, regulation files | 0 / 0 | the SDK needs Go 1.25 → installed by hand (Q1) | done |
| **P1** | stream, runner, queue | Parse Event, Encode Lines, Exec Runner, Phase Queue | 8 / [3,4,1] | probe lines by number; `tests/fixtures/plan/queue-3.json` | TASK, checks, 4 probes | 0.15 / 0.0495 | protocol drift → every example cites a probe line | done, run 20261008-171107 |
| **P2** | session, eventlog, telegram | Session Machine, Order Queue, Event Log, Milestone Post | 8 / [3,4,1] | probe lines 4–31; exact TG text and urlencoded body | TASK, checks, 4 probes | 0.15 / 0.0722 | a > 64 KiB tool line → Event Log 10 | done, run 20261008-173606 |
| **P3** | claude, github, bootstrap | Launch Args, Process, Repo Access, Project Create | 8 / [2,4,2] | stub `sh` scripts for the process; fake `Do`; scripted `runner.Fake` | TASK, checks, 4 probes | 0.15 / 0.0608 | a real child process in tests → stub scripts, Linux only | done, run 20261008-181657; **Smoke stop 1** green |
| **P4** | registry, gitrules, control | Project Registry, Secret Files, Phase Start Check, Phase End Check, Control Contract | 10 / [3,5,2] | `tests/fixtures/git/measure.md`; every git call scripted, none run | TASK, checks, 5 probes | 0.20 / 0.0444 | git in tests → `runner.Fake` only | done, run 20261008-185347 |

**E3–E4: loop, surfaces, the daemon**

| phase | Component | Functions | cards / gens | oracle and fixtures | by hand | $ est. / actual | risk → treatment | status |
|---|---|---|---|---|---|---|---|---|
| **P5** | supervisor, api | Phase Loop, Runtime Guard, HTTP Handlers, Router | 8 / [2,4,2] | Loop Action literals; exact API bodies `{"error":{…}}` + "\n" | TASK, checks, 4 probes | 0.25 / 0.0817 | time in the guard → `now` argument only | done, run 20261008-193337 |
| **P6** | mcpserver | PM Tools, Session Tools, MCP Mount | 6 / [1,2,2,1] | in-memory transports; decoded JSON-RPC fields, never bytes | TASK, checks, 3 probes, `docs/deps/go-sdk.md` | 0.20 / 0.0986 (+ debt claude $2.40) | the SDK API → digest from the vendored source; a judge variable mix-up → one debt card (Fable), later re-run green on P20 without Fable | done, run 20261008-200807 + debt a1d948a |
| **P7** | daemon, cmd | Daemon Core, Config And Main (Pump split off at P7b) | 6 / [1,1,3,1] | fakes declared in code files; a real `sh` claude in Pump 8 | TASK, checks, 3 probes | 0.35 / 0.1401 | the glue → nothing else in the phase; survived mutants are DECISIONS risks | done, run 20261008-211009; Smoke stop 2 red (D1–D3) |

**E5–E6: smoke-2 fixes, the PM MCP harness (re-cuts with `morph plan --only`)**

| phase | Component | Functions | cards / gens | oracle and fixtures | by hand | $ est. / actual | risk → treatment | status |
|---|---|---|---|---|---|---|---|---|
| **P7b** | control, supervisor, daemon, pump, cmd (re-cut of 12 built cards) | Control Contract, Phase Loop, Runtime Guard, Daemon Core, Pump, Config And Main | 12 / [1,2,2,2,1,3,1] | the probe's own session id for UUIDv4; a real `sh` claude exiting 1 | TASK, checks, 6 probes, `_stubs` | 0.40 / ≈ 3.70 | breaking renames across generations → MorphV2 subset transaction (P21b/c); retry cascade → #16 (P24); Go self-blame → #19; outcome: group `morph accept` of run 20261010-092439 (P24b) | done, 59e3c89..a37d6d6; Smoke stop 2 re-run green |
| **P7c** | supervisor (re-cut of 2) | Phase Loop | 2 / [1,1] | Phase Loop 3, 10, 11 | TASK, checks, 1 probe, `_stubs` | 0.05 / 0.0328 | none new | done, run 20261010-165924; D4 check green |
| **P8a** | bootstrap, control, api (3 new Functions + re-cut of 4) | Project Adopt, Adopt Contract, Event View, HTTP Handlers, Router | 10 / [3,4,2,1] | probe lines as exact view JSON; scripted git for adopt | TASK, checks, 5 probes, `_stubs` (3 new files) | 0.25 / 0.1005 | new files inside a subset transaction → stubs, no placeholders | done, run 20261010-183047 |
| **P8b** | mcpserver, daemon, pump, cmd (re-cut of 9) | PM Tools, Daemon Core, Pump, Config And Main (+ mcp-mount-judge) | 9 / [2,3,3,1] | probe line 31's result for the last turn; `Tick` at the deadline | TASK, checks, 4 probes, `_stubs` (record names only, #22), scout on mcpserver and daemon + pump | 0.40 / 0.18 so far (+ scouts 0.046) | a re-cut dropping private helpers other files use (`jsonResult`, `errorResult`, `okView`) → named in the record; judge type and literal faults (`Seq int64`) → data fix 886eb54, deck re-cut 006d9a2 | **open, next**; run 20261010-190913 red 0/9 |

### P7b

`morph plan --spec contour.yaml --map morph-map.json --component control --component supervisor --component daemon
--component pump --component cmd --judge --only
control-contract,control-contract-judge,phase-loop,phase-loop-judge,runtime-guard,runtime-guard-judge,daemon-core,daemon-core-judge,pump,pump-judge,config-and-main,config-and-main-judge`
→ 12 cards, generations [1,2,2,2,1,3,1] (control-contract → phase-loop + control-contract-judge → runtime-guard +
phase-loop-judge → daemon-core + runtime-guard-judge → pump → config-and-main + daemon-core-judge + pump-judge →
config-and-main-judge); the heaviest slice pump-judge 172 869 B. The record renames `Loop.Resumes → Restarts` and widens
`Exited` with the stderr (Q19–Q22); `pump` became its own Component of package daemon (`daemon/pump.go`). `checks.json`
freezes every package but control, supervisor, daemon and `cmd/morphd`. One phase, not two: 12 cards is the cap and
every generation is a 1–3 card step.

### P7c (MorphV2 18dab5b, on main after P7b)

`morph plan --root . --spec contour.yaml --map morph-map.json --component supervisor --judge --only
phase-loop,phase-loop-judge` → 2 cards, generations [1,1], slices 27 233 B and 40 332 B. Phase Loop's start-check
"resume" branch splits on SessionID: non-empty → `--resume <id>`; empty → a fresh session (`--session-id <new id>`, the
first line, the post "<phase> resumed on a fresh session" with the check's reason). Example 3 changes, 10 and 11 are
new. `checks.json` freezes every package but supervisor; runtime-guard, daemon-core and pump are not re-cut.

### P8a / P8b — the minimal PM MCP harness

Operator items 2, 3, 5 (10.10); item 4 is the regulation data `docs/AUTONOMY_MORPHD.md`.

- **P8a**: `morph plan --root . --spec contour.yaml --map morph-map.json --component bootstrap --component control
  --component api --judge --only
  project-adopt,project-adopt-judge,adopt-contract,adopt-contract-judge,event-view,event-view-judge,http-handlers,http-handlers-judge,router,router-judge`
  → 10 cards, generations [3,4,2,1]; the heaviest slice event-view-judge 103 920 B. New: `bootstrap/adopt.go`,
  `control/adopt.go`, `control/events.go` and their tests; re-cut: `api/handlers.go` (+ `AdoptProject`),
  `api/router.go` (+ `POST /projects/adopt`) and their judges (HTTP Handlers 9 and Router 8 new). After P8a `POST
  /projects/adopt` answers 500 `internal` until P8b gives the daemon its Adopter. `checks.json` freezes every package
  but bootstrap, control and api.
- **P8b**: `morph plan … --component mcpserver --component daemon --component pump --component cmd --judge --only
  pm-tools,pm-tools-judge,mcp-mount-judge,daemon-core,daemon-core-judge,pump,pump-judge,config-and-main,config-and-main-judge`
  → 9 cards, generations [2,3,3,1] (daemon-core + pm-tools → mcp-mount-judge + pm-tools-judge + pump → config-and-main +
  daemon-core-judge + pump-judge → config-and-main-judge; mcp-mount-judge `depends_on: ["pm-tools"]` in the map); the
  heaviest slice pump-judge 186 160 B. Changed examples: PM Tools 1 and 6 and MCP Mount 1 and 6 (the tool lists gain
  `events` and `project_adopt`), Config And Main 2–5 (`DoneWait`, 11 PM tools); new: PM Tools 8–9, Daemon Core 12–14,
  Pump 10–11. Daemon Core 1–11 and Pump 1–9 keep their tests (`DoneWait` 0 = the old behaviour). `checks.json` freezes
  every package but mcpserver, daemon and `cmd/morphd`. Tests 230 → 237.
- Design choices: `AdoptProject` is not a method of `control.Control` (17 methods, every judge's fake) but the optional
  `control.Adopter` the daemon implements and `control.Adopt` resolves by a type assertion; the compact view is a pure
  Function of package control (`Views` over `eventlog.Entry` + `stream.Parse`), the HTTP `GET events` keeps raw
  entries; the last-turn wait lives in the daemon (`pendingDone` per project: the pump's `result` or exit, or `Tick` at
  the deadline, runs the held end batch); the supervisor is untouched. Two phases: P8a's leaves and servers need no
  daemon change; P8b's daemon + pump are one package and move together with the `cmd` Config.

**Total**: P0 + 11 phases (P1–P7, P7b, P7c, P8a, P8b), 60 cards in the map (30 code, 30 judge), 87 card runs (P7b
re-cuts 12, P7c 2, P8a/P8b 13), 237 record examples. Executor: estimate $2.55; spent ≈ $4.38 through P8a + P8b's
$0.23 so far, of $35. Claude for the smokes ≤ $1.45 (spent $0.38); claude for debts $2.40 (P6).

## Smoke stops

- **Smoke stop 1 (after P3, the PM on the VPS, ≤ $0.05 of claude)**: a throwaway `main` outside the repository
  (`/tmp/smoke-p3/main.go`, not committed) calls `claude.Start(ctx, "claude", claude.Args(claude.Launch{SessionID:
  <new uuid>, BudgetUSD: 0.05}), <tmp dir>, nil)`, writes `stream.Initialize("init-1")`, then `stream.User("Reply with
  exactly one word: PONG")`, parses every line with `stream.Parse` and feeds a `session.Machine`. Expected: a `system`
  `init` with the given session id; a `result` with `Subtype "success"`, `Text "PONG"`, `NumTurns 1`, `TotalCostUSD`
  between 0.001 and 0.01; the machine `State "ready"`, `Turns 1`; `Limits` non-nil with both `ResetsAt` in the future;
  the process exits 0 within 5 s of `Stop()`. Red → the protocol moved: re-probe, the record's stream examples change.
- **Smoke stop 2, final (after P7; re-run after P7b; D4 check after P7c; the PM on the VPS over HTTP and MCP, ≤ $0.50
  of claude)**: `morphd` started with a `.env` (`MORPHD_TOKEN`, `TG_*`), a throwaway project `smoke` on an empty
  private GitHub repo; from the laptop:
  - `PUT /projects/smoke/github-token` (body `{"token":…}`) → `{"access":{"reachable":true,"push":true,…},"pushed":true}`;
  - before `POST plan`: `GET status` shows `"five_hour":null,"seven_day":null`, `GET usage` all five limit keys null;
  - `POST /projects/smoke/plan` with one phase `{"id":"S1","stop_after":"operator"}` whose AUTONOMY names a one-step
    phase ("write docs/MEASURE.md row S1, commit, push, call phase_done") → `GET status` `busy`, `phase "S1"`; `GET
    events?since=0` returns the `in` first line and the `out` init within 10 s; the session id of `status` and of the
    init event is a lower-case 8-4-4-4-12 UUID with `4` at the version position; `status` holds no `last_exit`;
  - after the first `rate_limit_event` both `status` and `usage` show the percentages;
  - TG `[smoke] 🚀 S1 started`; at `phase_done` `[smoke] 🛑 S1 done: stop after operator`, `GET stop` →
    `{"stop":{"kind":"operator",…}}`; `POST order` while idle → `{"sent":true}`; `POST interrupt` during a turn → the
    next `result` has `terminal_reason "aborted_streaming"`;
  - `claude mcp add --transport http morph https://<host>/mcp/smoke --header "Authorization: Bearer $MORPH_TOKEN"`:
    the `status` tool answers the same JSON as `GET status`;
  - should the session exit on its own: `GET events` ends with `{"code":<n>,"stderr":"…"}`, TG `🐕 S1: session exited
    (code <n>), restarting` with `restart 1 of 3 this hour · <first stderr line>`, `status` shows `last_exit`, never more
    than 4 session files within an hour, the fourth exit posts `🛑 S1: session keeps exiting` and `GET stop` → kind
    `crash`;
  - D4: with a commit on main not yet pushed before `POST plan`, TG `[smoke] 🚀 S1 resumed on a fresh session` with
    `· local ahead of origin/main`, the `in` first line and no `exit` entry, exactly one session file, no `last_exit`.
  Red → a DECISIONS line and an issue labelled for the Component; the stretch stops for the operator.
- **Smoke stop 3 (after P8b, the PM over the systemd morphd on the VPS through the ssh tunnel, ≤ $0.30 of claude)**:
  over `/mcp`: `project_adopt {name: "morphstudio", dir: "/home/morph/MorphStudio", repo_url:
  "https://github.com/VasyaLutiy/MorphStudio", language: "go"}` → the ProjectView with `state "idle"`; `projects_list`
  holds it; a second `project_adopt` of the same name → `exists: project exists`; of `/home/morph/morphd` →
  `bad_input: bad input: bootstrap: refused: not a git repository: /home/morph/morphd`; of a dir with another origin →
  the `… is not …` text; `git -C /home/morph/MorphStudio config --get-all credential.helper` ends with `store
  --file=/home/morph/morphd/state/morphstudio/git-credentials` and `git log origin/main -1` is unchanged across `PUT
  /projects/morphstudio/github-token` (`pushed: false`, `access.push: true`); over `/mcp/morphstudio`: `status` idle
  with `five_hour: null`, `events {}` → `{"events":[],"last":0}`; then on `smoke` a one-phase plan S3 ("write the
  MEASURE row, commit, push, phase_done, end the turn"): TG `[smoke] 🎉 S3 done: plan complete · stretch $0.0xxx`
  non-zero, `usage` `stretch_cost_usd` > 0, `events {since: 0, limit: 50}` shows the `result` view with `cost_usd` > 0
  followed by the `exit` view (`code -1`), the session ended within `MORPHD_DONE_WAIT` (30 s) of its last `result`;
  `GET /projects/smoke/events?since=0` still returns raw entries. Red → a DECISIONS line and an issue.

## What morphd changes against the ssh/tmux flow

- `tmux capture-pane` → `GET events?since=N` / the MCP `events` view and `GET status`.
- `send-keys` orders → `order` (one stream-json user line; queued while busy, sent on `result`); Enter on a dialog →
  `answer {option}`; Escape → only an explicit `interrupt`.
- Flags `~/.morph-phase-done` / `~/.morph-wait-operator` and the cron watchdog → session tools `phase_done` /
  `wait_operator`, the queue from `plan_load`, stops enforced by morphd, `continue` from the PM.
- `tools/tg.sh` and the Stop/Notification hooks → the `milestone` tool and morphd's own 💤/🔔 from the stream (the VPS
  `tg.env` goes, Q28).
- `/usage` by eye → `GET usage` from `rate_limit_event`; blind kicks on a limit → a pause until `resetsAt`.
- Crash: invisible respawns → an `exit` entry, a 🐕 per restart, ≤ N per hour, then a `crash` stop.
- **Kept on purpose**: git inside a phase is the session's (morphd never commits, merges or pushes); the cycle of one
  phase, the failure classes, debts and the processor rules stay in the project's AUTONOMY.

## Measures per phase (`docs/MEASURE.md`, one row per phase, fix, smoke or debt)

| metric | source |
|---|---|
| builder | processor + MorphV2 binary commit |
| cards planned / accepted / burned | the run report |
| $ executor / $ orchestrator (model tokens/calls/min) | the provider bill / the Claude session |
| minutes (run, prep) | from `morph plan` to the last commit; the prep agent |
| tsc-first-red, neighbour-red | failures at the build stage only / caused by a sibling card |
| judge defects, judge stubs | judge tests red on accepted code / judges refused by the guard |
| lines by hand | map, probes, fixtures before the run |
| hazards | `deck check`, fixed by hand before the run |
| max slice, bytes | `deck check` |
| failure class, runs | AUTONOMY's classes; run ids |

The running total of the stretch is kept under the table.

## Decisions on open questions

| Q | question | decision | by |
|---|---|---|---|
| Q1 | Go toolchain: the SDK needs Go 1.25 — the repo and the VPS were on 1.22 | Go 1.25.x installed by hand at P0, `go.mod` `go 1.25`, SDK v1.8.0 | operator 08.10 |
| Q2 | the P15L skeleton (project, store, auth, api, decks/c1) | removed at P0; module path, helpers, guard, `.morph/runs` archive stay | operator 08.10 |
| Q3 | the first message of a phase session | `/morph-orchestrator <phase>` | operator 08.10 |
| Q4 | claude flags; auto-memory off ("100%") | `--dangerously-skip-permissions`, `--permission-prompt-tool stdio`; every tool but AskUserQuestion auto-allowed; the child env always carries `CLAUDE_CODE_DISABLE_AUTO_MEMORY=1` (Launch Args `Env`), never a configuration | operator 08.10 |
| Q5 | `plan_load` begins the first phase at once | yes (no separate `start` tool) | operator 08.10 |
| Q6 | the TG kind of a session question | `ask` 🔔, not 🚦 | operator 08.10 |
| Q7 | the `morph init` argv | `morph init --root <dir> --name <n> --language <l>`, plus `--module <n>` for Go; git init follows | operator 08.10 |
| Q8 | the first push of `project_create` needs the PAT that arrives later | create = init + commit; the push runs at `PUT github-token` (`FirstPush`), result `pushed` | operator 08.10 |
| Q9 | `restart` semantics | kill + the start check; no reset, morphd never rewrites git | operator 08.10 |
| Q10 | git identity for the create commit | `user.name morphd`, `user.email morphd@localhost` in the project's config | operator 08.10 |
| Q11 | `.env` location and keys | the working directory of morphd; keys in Config And Main | operator 08.10 |
| Q12 | the executor $ per phase | not enforced by morphd in v1; the session posts it in `milestone run`; the stretch cap counts claude list price only | operator 08.10 |
| Q13 | the stall rule | one nudge per phase at 30 min of no events; then only the wall-clock cap | operator 08.10 |
| Q14 | not-pushed stop | the process stays alive; `continue` re-runs the end check | operator 08.10 |
| Q15 | when the repo access is re-checked | at the token PUT only; status reports the last check | operator 08.10 |
| Q16 | event log retention | one file per session under the state dir, never rotated in v1 | operator 08.10 |
| Q17 | the rate-limit pause rule | `utilization >= 1` of a window → pause until its `resetsAt` | operator 08.10 |
| Q18 | queued projects under max-parallel | the oldest queued project starts at the next Tick when a slot frees | operator 08.10 |
| Q19 | what resets the per-hour restart count | every exit the loop still wanted is a crash; fresh and `--resume` restarts both count; `plan_load`, `continue` and `restart` reset it | operator, cc46ed1 |
| Q20 | which ids must be UUIDs | one `newID` (RFC 4122 v4, lower-case 8-4-4-4-12) for the session id, the session token and the interrupt request id | operator, cc46ed1 |
| Q21 | the name `MORPHD_RESUMES_PER_HOUR` | kept for the env key and the Config fields; the loop's list is `restarts` | operator, cc46ed1 |
| Q22 | how a crash shows in `status` | no new `state`; `last_exit {code, at, stderr, restarts}`, `stop.kind "crash"` with the code and the first stderr line, an `exit` entry with the last 20 stderr lines, a 🐕 per restart | operator, cc46ed1 |
| Q23 | what adopt writes into the existing repository | only `git config credential.helper "store --file=<state>/<p>/git-credentials"`; no identity, commit, push or `morph init` | operator, 50a83fd |
| Q24 | where adopt lives in the API and the Control | `POST /projects/adopt`; the Control stays at 17 methods, `AdoptProject` is the optional `control.Adopter` | operator, 50a83fd |
| Q25 | how long the daemon waits for the last turn after `phase_done` | `MORPHD_DONE_WAIT` 30 s, then the end batch runs anyway; a daemon restart inside the wait loses it | operator, 50a83fd |
| Q26 | the `events` tool's bounds | text and stderr cut at 200 runes + "…"; `limit` absent → 50; HTTP `GET events` keeps raw lines | operator, 50a83fd |
| Q27 | caps of an adopted project | `Config.Defaults` ($30, 3 h, $5) | operator, 50a83fd |
| Q28 | Claude Code hooks (`tg.sh idle/ask`) under morphd | the daemon posts 💤 and 🔔 itself; the PM removes the morph user's `~/.config/morph/tg.env` after P8b | operator, 50a83fd |
| Q29 | the regulation a morphd session reads | `docs/AUTONOMY_MORPHD.md`; `docs/AUTONOMY.md` carries a pointer line | operator, 50a83fd |

## Out of scope

| item | owner later |
|---|---|
| Web wizard U0–U4, any browser UI | the Studio web product (SKETCH), after v1 |
| Hetzner VPS / snapshot provisioning | the provisioning product (SKETCH §7.1), after v1 |
| Multi-user, billing | the SaaS layer; one morphd = one user = one VPS in v1 |
| TLS, the public hostname, systemd unit, log rotation | Caddy and the VPS runbook (the PM's data, not cards) |
| Executor $ enforcement from `morph run` reports | a later morphd phase once the session reports it (Q12) |
| Parallel sessions (max-parallel > 1 with a scheduler) | a later phase; the setting and the per-project state exist |
| Refreshing the GitHub access check periodically; creating repositories (a fine-grained PAT cannot) | later; the user creates the empty repo |
| HTTP routes for the session tools (phase_done over curl) | not needed: the session reports over MCP |
| Hook events (`--include-hook-events` is passed; nothing reads them) | later if a hook kind proves useful |
| `--model` policy, effort | config only (`MORPHD_MODEL`, `MORPHD_CLAUDE_EXTRA_ARGS`); auto-memory is NOT config (Q4) |

## Scaffold (P0, hand data)

Nothing here is written by a card.

- `go.mod`: `module morphstudio`, `go 1.25`, `require github.com/modelcontextprotocol/go-sdk v1.8.0` (+ its
  requirements as `go mod tidy` writes them); `go.sum`; `internal/deps/tools.go` (`//go:build tools`, blank import of
  `github.com/modelcontextprotocol/go-sdk/mcp`); `go mod vendor` → `vendor/` committed.
- Removed: `project/`, `store/`, `auth/`, `api/`, `decks/c1/` (Q2); kept: `.morph/runs/`, `.morph/primer.md`, `docs/`,
  `.gitignore` (+ `*.env`, `/tmp/`).
- `internal/testhelp/testhelp.go` (`Equal`, `WriteFile`) and `internal/testhelp/probe.go`: `func ProbeLine(t
  testing.TB, path string, n int) (dir string, msg []byte)` — line n (1-based) of a JSONL file, `{"t":…,"dir":…,"msg":…}`
  with `msg` raw; `func ProbeOut(t, path string, from, to int) [][]byte` — the msg bytes of every `out` line in
  `from..to`.
- `decks/tools/guard.mjs`, `decks/tools/firstdiff.mjs`, `decks/tools/stubcheck.mjs`, `decks/tools/scale_tokens.py`:
  copies of MorphV2 `templates/go/decks/tools/*` and `templates/common`.
- `decks/tools/layers.json` — the layer table (direct imports only; `pure: []` because the machines carry `time.Time`):

  | package | may import (module packages) |
  |---|---|
  | stream, runner, queue, eventlog, telegram, github, claude, internal/testhelp | — |
  | session | stream |
  | bootstrap | runner, github |
  | registry | queue |
  | gitrules | runner |
  | control | queue, github, eventlog, session, stream |
  | supervisor | queue, gitrules, control, stream |
  | api | control, queue, github, eventlog, session |
  | mcpserver | control, queue, github, eventlog, session, supervisor |
  | daemon | stream, session, eventlog, telegram, runner, queue, claude, github, bootstrap, registry, gitrules, control, supervisor |
  | cmd/morphd | every package |

  Per-card stdlib rules for `checks.json` `extra`: `os/exec` only in runner and claude; `time.Now`, `math/rand`,
  `crypto/rand` only in `cmd/morphd`; `net.Dial`, `http.Get`, `http.Post`, `http.DefaultClient` only in `cmd/morphd`
  (every other package takes `Do`); `syscall`, `unsafe` nowhere but claude (`syscall.SIGKILL`).
- Fixtures: `tests/fixtures/stream/probe.jsonl` = `docs/probe/stream-json-2.1.294.jsonl` byte for byte (37 lines,
  81 752 B); `tests/fixtures/plan/queue-3.json` = exactly
  `{"approved":"7b31dfe","phases":[{"id":"P17","stop_after":"none"},{"id":"P18","stop_after":"smoke","caps":{"claude_usd":12}},{"id":"P19","stop_after":"none"}]}`;
  `tests/fixtures/git/measure.md` = a MEASURE table whose rows begin `| P16 primer-go |` and `| P17 debt |`.
- `docs/deps/go-sdk.md` (from the vendored v1.8.0 source, every value run on it): the import path, `Implementation`,
  `NewServer`, `AddTool[In, Out]` with the `json`/`jsonschema` tags and the empty-input idiom, `CallToolRequest`,
  `CallToolResult`, `NewStreamableHTTPHandler` with the stateless options and headers, `NewInMemoryTransports`,
  `Server.Connect`, `NewClient`, `Client.Connect`, `ClientSession.ListTools / CallTool / Close`, one measured
  `tools/call` request and response.
- Regulation files from MorphV2 `templates/common` instantiated for morphd: `docs/AUTONOMY.md`, `docs/MEASURE.md`,
  `docs/TASK_TEMPLATE.md`, `CLAUDE.md`, `tools/`, `.claude/settings.json`; `README.md` for morphd;
  `docs/AUTONOMY_MORPHD.md` (P8, data).

## Zero Contour

Principles: one System `morphd`; a Component = one Go package (or, for `pump`, one file of package daemon), every
Component `language: go`; `calls` names a Function of the same Component, `uses` another Component; Data Objects with
a `schema` for what the executor must construct (Event, Action, Plan, Status, Loop Action); every Function has
`behavior` with the exact Go declarations and `examples` with full literals; large literals live in fixtures and
examples cite them by `ref`; the record is English (the executor reads it verbatim). The record is committed
(`contour.yaml`, 233 KB); sizes below are the YAML dump of each Component (10.10).

| Component | KB | Functions | examples |
|---|---|---|---|
| stream | 10.2 | Parse Event, Encode Lines | 16 |
| session | 10.0 | Session Machine, Order Queue | 14 |
| eventlog | 4.5 | Event Log | 10 |
| telegram | 4.3 | Milestone Post | 8 |
| runner | 3.4 | Exec Runner | 7 |
| queue | 5.5 | Phase Queue | 8 |
| claude | 8.4 | Launch Args, Process | 14 |
| github | 3.9 | Repo Access | 8 |
| bootstrap | 13.6 | Project Create, Project Adopt | 14 |
| registry | 7.1 | Project Registry, Secret Files | 13 |
| gitrules | 7.9 | Phase Start Check, Phase End Check | 15 |
| control | 20.0 | Control Contract, Adopt Contract, Event View | 19 |
| supervisor | 24.5 | Phase Loop, Runtime Guard | 20 |
| daemon | 27.7 | Daemon Core | 14 |
| pump | 16.8 | Pump | 11 |
| api | 14.5 | HTTP Handlers, Router | 17 |
| mcpserver | 20.6 | PM Tools, Session Tools, MCP Mount | 22 |
| cmd | 8.0 | Config And Main | 7 |
| **18** | | **30** | **237** |

Requirements: Claude Behind An Interface · Bearer Token On Every Route · The Session Reports Only Itself.
Guardrails: Injected Clock And Ids · No Network In Tests · No Shell Outside Runner · Secrets Never Logged.
Dependency: `github.com/modelcontextprotocol/go-sdk` v1.8.0 (go), doc `docs/deps/go-sdk.md`, used by mcpserver only.
Limit: one Component ≤ 30 KB — daemon is the nearest (27.7 KB); the next daemon change splits a third Component off
package daemon (a plan change, the architect).

## morph-map.json

Committed with the record: `language: go`, `docs: ["go.mod"]`, 60 card entries (30 code, 30 judge) with `targets`,
`context_slice`, `depends_on`, `instruction` and `max_tokens` the record does not carry; the slice names the files a
card reads (the record never goes in a slice). Budgets are before the processor's ×3 (`scale_tokens.py … 3` on `ds`).
The cards of P1:

| card | target | slice | depends on | max_tokens |
|---|---|---|---|---|
| parse-event | stream/parse.go | go.mod | — | 12 000 |
| encode-lines | stream/encode.go | go.mod, stream/parse.go | parse-event | 8 000 |
| exec-runner | runner/runner.go | go.mod | — | 10 000 |
| phase-queue | queue/queue.go | go.mod | — | 12 000 |
| parse-event-judge | stream/parse_examples_test.go | go.mod, parse.go, testhelp.go, probe.go, fixtures/stream/probe.jsonl | parse-event | 20 000 |
| encode-lines-judge | stream/encode_examples_test.go | go.mod, encode.go, parse.go, testhelp.go, probe.go, probe.jsonl | encode-lines | 16 000 |
| exec-runner-judge | runner/runner_examples_test.go | go.mod, runner.go, testhelp.go | exec-runner | 14 000 |
| phase-queue-judge | queue/queue_examples_test.go | go.mod, queue.go, testhelp.go, fixtures/plan/queue-3.json | phase-queue | 18 000 |

The heaviest budgets now: daemon-core-judge 38 000, pump-judge 34 000, pm-tools-judge 32 000, phase-loop-judge 30 000,
http-handlers-judge 30 000, daemon-core 28 000, runtime-guard-judge 26 000, session-machine-judge 24 000, router-judge
24 000.

## Verification

Before every run (the gate; `docs/AUTONOMY.md` "The cycle of one phase"):

1. `morph plan --spec contour.yaml --map morph-map.json --component <C>… --judge [--only …] --checks
   decks/<phase>/checks.json --out decks/<phase>/deck.json` → exit 0; cards and generations as in the phase row.
2. `morph deck check` → 0 errors; every slice under 200 KB.
3. `morph gate` (or the probes by hand in a scratch worktree): every acceptance red per example on stubs, green on a
   reference; mutants ≤ 30.

After the run, on `morph/<run-id>`: `git status --short` empty but the cards' targets; `GOFLAGS=-mod=vendor
GOPROXY=off go build ./... && go vet ./...`, `gofmt -l` empty, `go test -count=1 ./...` green, `-race` on supervisor
and daemon; the MEASURE row; main moves only by fast-forward.

Done-criteria: **the framework** — Smoke stop 2, final, green (held 10.10, with the D4 check); **the harness** — Smoke
stop 3 green after P8b, with `GOFLAGS=-mod=vendor GOPROXY=off go test -count=1 ./...` green at 237 example tests.

## Record rules for this project

1. The record holds only the current contract: no history.
2. Large literals live in fixtures; one Component ≤ 30 KB, else two. Every stream or session example names its probe
   line; a new protocol fact is a new line in a new probe fixture, never an edited old one.
3. Cross-Component dependencies are `uses: <component>` steps; `calls:` names a Function of the same Component; the
   file-level dependencies are the map's `context_slice` and `depends_on`.
4. Every time is a `now` argument or an injected `Now`; every id an injected `NewID`; every external process behind
   `runner.Runner` or `claude.Process`; every HTTP call behind an injected `Do`. The fakes live in the code files
   (`runner.Fake`, `claude.Fake`) so other packages' judges can use them.
5. Error texts: a package prefix (`stream: `, `eventlog: `, `queue: `, `bootstrap: `, `registry: `, `telegram: `,
   `github: `, `claude: `, `config: `), sentinels as package variables compared with `errors.Is`, API bodies
   `{"error":{"code":…,"message":…}}` + `"\n"`, every body written by `json.Encoder`.
6. Secrets: files of mode 0600 under the state dir; never in a response, a log line, a milestone, an error, or an argv.
7. Layers: `decks/tools/layers.json` is the only import table; a new package is a plan change.
8. A re-cut names in the record every symbol of its files that a file outside the subset uses (private helpers too).
