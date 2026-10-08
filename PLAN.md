# PLAN.md — morphd (Morph Studio, first goal)

> Written by `morph-architect` from `docs/START.md` (operator-approved 08.10) and `docs/DECISIONS.md`; approved by the
> operator by commit hash. After approval only the phase data changes without a new approval. The record is
> `contour.yaml`, the map `morph-map.json`, both committed with this plan. Open questions: `docs/QUESTIONS.md`.

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
  cap $5 per phase; phases of 8–12 cards.
- What the operator gets at the end: a daemon binary `morphd` with 17 packages, an HTTP API of 15 routes under a
  bearer token, three MCP endpoints with 15 tools, a JSONL event log per session, Telegram milestones, and about 200
  example tests (one per record example). **Not production**: no TLS, no deploy unit, no browser UI, no multi-user,
  no provisioning — the framework the PM and the operator drive from outside.

## Brief → design (traceability)

| scenario of the brief | phase | Component · Function | examples carrying its values |
|---|---|---|---|
| S1 `status` → `{state: "busy", phase: "P18", minutes: 37, cost_usd: 1.92, five_hour: 22, seven_day: 60}` | P4, P7 | control · Control Contract; daemon · Daemon Core | Control Contract 1 (the exact JSON); Daemon Core 2 (busy, P17); Pump 1 (22 / 60 from the probe's rate_limit_event) |
| S2 `order` while the session waits → `{sent: true}`; while busy → `{queued: 1}`, sent after `result` | P2, P7 | session · Order Queue, Session Machine; daemon · Daemon Core | Order Queue 1–2 (`{"sent":true}`, `{"queued":1}`); Session Machine 5 (the queue drains on the result); Daemon Core 3 |
| S3 the session asks "Gate waiver" with 2 options → TG, `pending` shows it, `answer {option: 1}` → the session continues | P1, P2, P7 | stream · Parse Event, Encode Lines; session · Session Machine, Order Queue; daemon · Pump | Parse Event 6 (probe line 27); Encode Lines 6–7 (the answer line equals probe line 28); Session Machine 3; Order Queue 4–5; Pump 2 (TG `ask`, `answer` 2 → "Option B") |
| S4 `phase_done {phase: "P17", next: "P18"}`, push checked → a fresh session at once, TG "🚀 P18" | P4, P5, P7 | gitrules · Phase End Check; supervisor · Phase Loop; daemon · Daemon Core | Phase End Check 1; Phase Loop 5 (`kill`, `start_check P18`); Daemon Core 5 (the second spawn, "P18 started") |
| S5 `phase_done`, main not pushed → no new session, TG "🛑 P17 not pushed", state waiting | P4, P5, P7 | gitrules · Phase End Check; supervisor · Phase Loop; daemon · Daemon Core | Phase End Check 2; Phase Loop 7; Daemon Core 6 (no spawn, the stop post, `continue` re-checks) |
| flow 1 look: `tmux capture-pane` → `GET events?since=N`, `GET status` | P2, P5 | eventlog · Event Log; api · HTTP Handlers | Event Log 2–5 (seq, since); HTTP Handlers 1, 5 |
| flow 3 order only when idle, else queue | P2 | session · Order Queue | Order Queue 1–2 |
| flow 4 dialog answer by option; interrupt only explicit; every other tool allowed without a human | P1, P2 | stream · Encode Lines; session · Session Machine, Order Queue | Encode Lines 5 (Allow); Session Machine 4 (auto-allow Bash); Order Queue 6–7 (Interrupt) |
| flow 5 restart: refuse while busy unless force; git pull ff-only; new process, new session id; events → log file | P4, P5, P7 | gitrules · Phase Start Check; supervisor · Phase Loop; daemon · Daemon Core | Phase Start Check 3 (pull ff-only); Phase Loop 9 (Restart); Daemon Core 4 (ErrBusy, then force) |
| flow 7 the session tells the daemon (`phase_done` / `wait_operator`); the daemon acts after the push check; exit seen at once → `--resume` ≤ N per hour; limit → pause until resetsAt; stall 30 min → TG, one nudge | P3, P5, P6 | claude · Launch Args, Process; supervisor · Phase Loop, Runtime Guard; mcpserver · Session Tools | Launch Args 2 (`--resume`); Process 1–3 (exit seen on a channel); Runtime Guard 1–2 (3 per hour), 5–6 (pause, resume at reset), 7 (one nudge, wall clock cap); Session Tools 2–5 |
| flow 6 `GET usage`: five_hour / seven_day + resets + the session's cost sum; TG alert at 50% weekly | P1, P4, P5 | stream · Parse Event; control · Control Contract; supervisor · Runtime Guard | Parse Event 3 (0.22 / 0.6, resetsAt); Control Contract 7 (the Usage JSON); Runtime Guard 4 (the 60% alert, once) |
| protocol: start argv, initialize, user line, interrupt, AskUserQuestion answer, rate_limit_event, result fields | P1, P3 | stream · Parse Event, Encode Lines; claude · Launch Args | every Parse Event / Encode Lines example cites a probe line; Launch Args 1 (the argv of START "Protocol facts" plus `--dangerously-skip-permissions`) |
| session per phase: fresh process, first message `/morph-orchestrator <phase>`, `plan_load {approved, phases: id, stop after, caps}`, stops enforced by morphd, `stop_check` / `continue` | P1, P5, P6 | queue · Phase Queue; supervisor · Phase Loop; mcpserver · PM Tools | Phase Queue 1, 4–5 (stop after smoke, end); Phase Loop 2 (`first_line`), 8 (stop, continue); PM Tools 1, 3, 5 |
| caps: Claude `--max-budget-usd`, wall clock; the stretch total | P3, P5 | claude · Launch Args; supervisor · Runtime Guard | Launch Args 1–2 (`--max-budget-usd 30`, `12.5`); Runtime Guard 3 (stretch $30), 7 (3 h) |
| Claude auto-memory in phase sessions: off, 100% (operator 08.10) | P3 | claude · Launch Args | Launch Args 4 (`Env` is exactly `["CLAUDE_CODE_DISABLE_AUTO_MEMORY=1"]`, whatever the Launch) |
| git of a phase: morphd never commits/merges/pushes; start: fetch; equal → fresh; ahead → resume; diverged → stop; end: pushed, clean, MEASURE row, no `morph/*` unmerged | P4 | gitrules · Phase Start Check, Phase End Check | Phase Start Check 1–8; Phase End Check 1–7 (every git call scripted, none run) |
| rule: token header, Caddy TLS, localhost only | P5, P7 | api · Router; cmd · Config And Main | Router 2–4 (401 bodies, an empty token refuses); Config And Main (`127.0.0.1:<port>` in the behaviour; Requirement "Bearer Token On Every Route") |
| rule: Telegram by morphd, `[<project>]` prefix, kinds and icons of AUTONOMY | P2 | telegram · Milestone Post | Milestone Post 1–5 (the exact text and the urlencoded body) |
| projects: registry, one folder each, `project` in every call, per-project queue/state/log/caps/sessions; max-parallel default 1 | P4, P7 | registry · Project Registry, Secret Files; daemon · Daemon Core | Project Registry 1–7; Daemon Core 9 (the second project waits) |
| `project_create {name, language, repo_url}`: morph init, first push | P3, P7 | bootstrap · Project Create; daemon · Daemon Core | Project Create 1–7 (the eight commands, `--module` for Go; FirstPush at the token PUT — Q8) |
| MCP binding: `/mcp/<project>` tools without `project`; user-level `/mcp` with projects_list / project_create; the session's `/mcp/<project>/session` with its own token via `--mcp-config` | P3, P6 | claude · Launch Args; mcpserver · PM Tools, Session Tools, MCP Mount | Launch Args 5–6 (the mcp-config file); PM Tools 1, 6; Session Tools 1; MCP Mount 1–6 (tokens per path) |
| GitHub token: HTTP only, stored mode 600, never returned; status reports reachable + push access with a plain reason | P3, P4, P5 | github · Repo Access; registry · Secret Files; api · HTTP Handlers | Repo Access 4–8 (401/403/404 reasons); Secret Files 1, 4 (0600); HTTP Handlers 8 (the token appears nowhere in the body) |

Every rule of the brief is an example or a Guardrail: Guardrails "Injected Clock And Ids", "No Network In Tests",
"No Shell Outside Runner", "Secrets Never Logged"; Requirements "Claude Behind An Interface", "Bearer Token On Every
Route", "The Session Reports Only Itself". Every not-build item is in "Out of scope".

## What is known

- **Language profile**: MorphV2 `go` (P15): code `{component}/{name}.go`, judges `<name>_examples_test.go` next to the
  code, `go build ./...` + `gofmt -l`, `go vet ./...`, the Go guard (`decks/tools/guard.mjs` = V2's goguard, layers from
  `decks/tools/layers.json`, direct imports by `go list`, the declared module allowed per card since P19b),
  `go test -count=1 ./<package>/` then `./...`; helpers `internal/testhelp` (`Equal`, `WriteFile`, and `ProbeLine`
  added at P0). Dependencies: declared in the record (`System.dependencies`, `Component.uses`), vendored at P0,
  `GOFLAGS=-mod=vendor GOPROXY=off` in every acceptance (P19b). Each code card gets per-package import rules as
  `extra` lines of the phase's `checks.json` (the P15L pattern; the table is in "Scaffold").
- **Comparable project**: P15L gocrud in this very repository: 14 cards, 5 generations, 14/14 written, 12 at the
  first attempt, $0.0852, 7.1 min on `deepseek/deepseek-v4.1-flash`; MorphV2's own P19a/P19b: 8 cards each, 8/8,
  $0.0935 / $0.0934. morphd's cards are larger (daemon-core ≈ 20 KB of answer), so the estimates below are 2–4× P15L.
- **The claude protocol** is measured, not assumed: `docs/probe/stream-json-2.1.294.jsonl` (CLI 2.1.294, 37 lines,
  4 turns, $0.0064) is copied at P0 to `tests/fixtures/stream/probe.jsonl` and every stream/session example names
  the line it uses (`docs/probe/` is gitignored by the `probe/` rule; the fixture copy is tracked).
- **The SDK**: `github.com/modelcontextprotocol/go-sdk` v1.8.0 (module proxy 08.10: the latest tag, 2026-09-04; `go
  1.25.0`; direct requirements jwt/v5, go-cmp, jsonschema-go, segmentio/encoding, uritemplate/v3, x/oauth2, x/time,
  x/tools; the `mcp` package imports jsonschema-go, uritemplate, x/oauth2, x/sync, x/time and the SDK's `auth`,
  `jsonrpc`, `internal/*`). Verified in its source: `mcp.NewServer`, generic `mcp.AddTool[In, Out]`, `ToolHandlerFor`
  = `func(ctx, *CallToolRequest, In) (*CallToolResult, Out, error)`, `mcp.NewStreamableHTTPHandler(getServer func(*http.Request) *Server, *StreamableHTTPOptions{Stateless, JSONResponse})`,
  `mcp.NewInMemoryTransports()`, `Server.Connect`, `Client.Connect`, `ClientSession.ListTools / CallTool`; a stateless
  POST needs `Accept: application/json, text/event-stream` (else 400). **Every SDK version needs Go ≥ 1.23 and v1.6.1+
  needs Go 1.25** — the repository is on Go 1.22: Q1.
- **Risks of this design for autonomous building and how the cut handles each**:
  1. *The daemon glue* (daemon-core, pump): goroutines, a mutex, fakes of five kinds. Kept pure everywhere else
     (session and supervisor are state machines with literal examples); the daemon examples use only fakes declared
     in non-test files (`runner.Fake`, `claude.Fake`, recorder funcs); daemon-core-judge sized 32 000 before ×3; P7
     holds nothing else but `cmd`.
  2. *The MCP SDK API at v1.8.0*: the mount examples compare decoded JSON-RPC fields, never bytes; the digest
     `docs/deps/go-sdk.md` is written at P0 from the vendored source with every value run on v1.8.0; P6 is the SDK
     alone (6 cards) so a red there is isolated.
  3. *A real claude at a rate limit was never observed* (the probe saw `status: "allowed"`): the pause rule is pinned
     on `utilization >= 1` (Runtime Guard 5); the first real limit is a DECISIONS line, not a redesign.
  4. *Process tests* (claude · Process) run stub shell scripts: Linux only, no network; `sh` and `/dev/zero` are
     on every VPS.
  5. *Two Functions of one package in one phase* serialize generations (Go builds a package whole): every package
     here has ≤ 3 Functions and every phase stays at 3–4 generations (measured by the dry cut below).

## Epics and phases

| phase | Component(s) | Functions | cards | gens | depends on | $ est. | stop after |
|---|---|---|---|---|---|---|---|
| P0 | scaffold (hand data) | — | 0 | — | — | 0 | — |
| P1 | stream, runner, queue | Parse Event, Encode Lines, Exec Runner, Phase Queue | 8 | 3 | P0 | 0.15 | — |
| P2 | session, eventlog, telegram | Session Machine, Order Queue, Event Log, Milestone Post | 8 | 3 | P1 | 0.15 | — |
| P3 | claude, github, bootstrap | Launch Args, Process, Repo Access, Project Create | 8 | 3 | P1 | 0.15 | **smoke 1** |
| P4 | registry, gitrules, control | Project Registry, Secret Files, Phase Start Check, Phase End Check, Control Contract | 10 | 3 | P1, P2, P3 | 0.20 | — |
| P5 | supervisor, api | Phase Loop, Runtime Guard, HTTP Handlers, Router | 8 | 3 | P4 | 0.25 | — |
| P6 | mcpserver | PM Tools, Session Tools, MCP Mount | 6 | 4 | P4 | 0.20 | — |
| P7 | daemon, cmd | Daemon Core, Pump, Config And Main | 6 | 4 | P2–P6 | 0.35 | **smoke 2 (final)** |

Measured by the dry cut (V2 binary built from origin/main bc311aa, 08.10, on a scratch copy with the P0 placeholders):
every phase `morph plan` exit 0 and `morph deck check` 0 errors; generations P1 [3,4,1], P2 [3,4,1], P3 [2,4,2],
P4 [3,5,2], P5 [2,4,2], P6 [1,2,2,1], P7 [1,1,3,1]; the largest slice 19 files (daemon-core), the heaviest judge slice
≈ 83 KB (the probe log), all under 200 KB.

- **Smoke stop 1 (after P3, the PM on the VPS, ≤ $0.05 of claude)**: a throwaway `main` outside the repository
  (`/tmp/smoke-p3/main.go`, not committed) that calls `claude.Start(ctx, "claude", claude.Args(claude.Launch{SessionID:
  <new uuid>, BudgetUSD: 0.05}), <tmp dir>, nil)`, writes `stream.Initialize("init-1")`, then `stream.User("Reply with
  exactly one word: PONG")`, parses every line with `stream.Parse` and feeds a `session.Machine`. Expected: a `system`
  `init` with the given session id; a `result` with `Subtype "success"`, `Text "PONG"`, `NumTurns 1`, `TotalCostUSD`
  between 0.001 and 0.01; the machine `State "ready"`, `Turns 1`; `Limits` non-nil with both windows' `ResetsAt` in the
  future; the process exits 0 within 5 s of `Stop()`. Red → the protocol moved since 2.1.294: the PM re-probes and the
  record's stream examples change (a DECISIONS line each).
- **Smoke stop 2, final (after P7, the PM on the VPS over HTTP and MCP, ≤ $0.50 of claude)**: `morphd` started by hand
  with a `.env` (`MORPHD_TOKEN`, `TG_*`), a throwaway project `smoke` whose repository is an empty GitHub repo created
  for it; then from the laptop: `PUT /projects/smoke/github-token` → `{"access":{"reachable":true,"push":true,…},"pushed":true}`;
  `POST /projects/smoke/plan` with one phase `{"id":"S1","stop_after":"operator"}` whose `docs/AUTONOMY.md` names a
  one-step phase ("write docs/MEASURE.md row S1, commit, push, call phase_done") → `GET status` shows `busy`, `phase
  "S1"`, `five_hour`/`seven_day` within 0..100; `GET events?since=0` returns the `in` first line and the `out` init
  within 10 s; TG shows `[smoke] 🚀 P… S1 started`; when the session calls `phase_done` the daemon posts `[smoke] 🛑 S1
  done: stop after operator`, `GET stop` → `{"stop":{"kind":"operator",…}}`; `POST order` while idle → `{"sent":true}`;
  `POST interrupt` during a turn → the next `result` has `terminal_reason "aborted_streaming"`; from the laptop
  `claude mcp add --transport http morph https://<host>/mcp/smoke --header "Authorization: Bearer $MORPH_TOKEN"` and
  the `status` tool answers the same JSON as `GET status`. Red → a DECISIONS line and an issue labelled for the
  Component; the stretch stops for the operator.
- **Total**: 7 phases, 54 cards (27 code, 27 judge), 199 record examples; executor estimate $1.45 on `ds` (cap $5 per
  phase, $35 for the stretch); claude for the two smokes ≤ $0.55.

### Measures per phase

The `docs/MEASURE.md` row every phase fills: written / planned, first attempt, retries, fixes, $ executor, $ orchestrator
(model tokens/calls/min), run minutes, preparation minutes, mutants (count / killed / minutes), the largest slice in
bytes, the run id.

### Decisions on open questions

| Q | question | decision | by |
|---|---|---|---|
| Q1 | Go toolchain: the SDK needs Go 1.25 (v1.8.0) — the repo and the VPS are on 1.22 | Go 1.25.x installed by hand at P0, `go.mod` `go 1.25`, SDK v1.8.0 | operator 08.10 |
| Q2 | the P15L skeleton (packages project, store, auth, api, 87 tests, decks/c1) | removed at P0; module path, helpers, guard, `.morph/runs` archive stay | operator 08.10 |
| Q3 | the first message of a phase session | `/morph-orchestrator <phase>` (START "Session per phase"), not `vps-prompt.txt` | operator 08.10 |
| Q4 | claude flags beyond the probe's; auto-memory off (operator's hard rule, "100%") | `--dangerously-skip-permissions` and `--permission-prompt-tool stdio`; every tool but AskUserQuestion auto-allowed; the child env always carries `CLAUDE_CODE_DISABLE_AUTO_MEMORY=1` (claude · Launch Args `Env`, example 4), never a configuration; whether AskUserQuestion still arrives as `can_use_tool` under skip-permissions is the PM's check before approval | operator 08.10 |
| Q5 | `plan_load` begins the first phase at once | yes (no separate `start` tool) | operator 08.10 |
| Q6 | the TG kind of a session question | `ask` 🔔 (AUTONOMY's hook kind), not 🚦 | operator 08.10 |
| Q7 | the `morph init` argv | `morph init <dir> --name <n> --language <l>`, plus `--module <n>` for Go (PM fact from V2 `src/cli/parse.ts`: one positional dir, flags --name --language --module --templates) | operator 08.10 |
| Q8 | the first push of `project_create` needs the PAT that arrives later | create = init + commit; the push runs at `PUT github-token` (`FirstPush`), result `pushed` | operator 08.10 |
| Q9 | `restart` semantics | kill + the start check (a resume state yields `--resume`); no reset, morphd never rewrites git | operator 08.10 |
| Q10 | git identity for the create commit | `user.name morphd`, `user.email morphd@localhost` in the project's config | operator 08.10 |
| Q11 | `.env` location and keys | the working directory of morphd; keys in Config And Main | operator 08.10 |
| Q12 | the executor $ per phase (from the run report) | not enforced by morphd in v1; the session posts it in `milestone run` numbers; the stretch cap counts claude list price only | operator 08.10 |
| Q13 | the stall rule | one nudge per phase at 30 min of no events; then only the wall-clock cap | operator 08.10 |
| Q14 | not-pushed stop | the process stays alive; `continue` re-runs the end check | operator 08.10 |
| Q15 | when the repo access is re-checked | at the token PUT only; status reports the last check | operator 08.10 |
| Q16 | event log retention | one file per session under the state dir, never rotated in v1 | operator 08.10 |
| Q17 | the rate-limit pause rule | `utilization >= 1` of a window → pause until its `resetsAt`; revisit at the first real limit | operator 08.10 |
| Q18 | queued projects under max-parallel | the oldest queued project starts at the next Tick when a slot frees | operator 08.10 |

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
| HTTP routes for the session tools (phase_done over curl) | not needed: the session reports over MCP; the final smoke drives a real session |
| Hook events (`--include-hook-events` is passed; nothing reads them) | later if a hook kind proves useful; the stream carries result and can_use_tool |
| `--model` policy, effort | config only (`MORPHD_MODEL`, `MORPHD_CLAUDE_EXTRA_ARGS`); no logic. Auto-memory is NOT config: always off by `CLAUDE_CODE_DISABLE_AUTO_MEMORY=1` in the child env (Launch Args `Env`) |

## Scaffold (P0, hand data on the VPS side)

One line each; nothing here is written by a card.

- `go.mod`: `module morphstudio`, `go 1.25`, `require github.com/modelcontextprotocol/go-sdk v1.8.0` (+ its requirements as
  `go mod tidy` writes them); `go.sum`; `internal/deps/tools.go` (`//go:build tools`, blank import of
  `github.com/modelcontextprotocol/go-sdk/mcp`); `go mod vendor` → `vendor/` committed (≈ 3–6 MB, measured at P0);
  `GOFLAGS=-mod=vendor GOPROXY=off go build ./...` green on the empty tree. Go 1.25.x on the VPS (Q1).
- Removed: `project/`, `store/`, `auth/`, `api/`, `decks/c1/` (Q2); kept: `.morph/runs/`, `.morph/primer.md` (re-generated
  after P1), `docs/`, `.gitignore` (+ `*.env`, `/tmp/`).
- `internal/testhelp/testhelp.go` (as is: `Equal`, `WriteFile`) and new `internal/testhelp/probe.go`:
  `func ProbeLine(t testing.TB, path string, n int) (dir string, msg []byte)` — reads line n (1-based) of a JSONL file,
  decodes `{"t":…,"dir":…,"msg":…}` with `msg` as `json.RawMessage`, returns dir and the raw bytes; `func ProbeOut(t, path
  string, from, to int) [][]byte` — the msg bytes of every `out` line in `from..to`.
- `decks/tools/guard.mjs`, `decks/tools/firstdiff.mjs`: copies of MorphV2 `templates/go/decks/tools/*` at origin/main (the
  P19b guard that reads the allowed module list); `decks/tools/scale_tokens.py` from `templates/common`.
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

  Per-card stdlib rules for the phase's `checks.json` `extra` (the P15L pattern, `go list -e -overlay` + `gofmt -r`):
  `os/exec` only in runner and claude; `time.Now`, `math/rand`, `crypto/rand` only in cmd/morphd; `net.Dial`,
  `http.Get`, `http.Post`, `http.DefaultClient` only in cmd/morphd (every other package takes `Do`); `syscall`, `unsafe`
  nowhere but claude (`syscall.SIGKILL`).
- Fixtures: `tests/fixtures/stream/probe.jsonl` = `docs/probe/stream-json-2.1.294.jsonl` byte for byte (37 lines, 81 752 B);
  `tests/fixtures/plan/queue-3.json` = exactly
  `{"approved":"7b31dfe","phases":[{"id":"P17","stop_after":"none"},{"id":"P18","stop_after":"smoke","caps":{"claude_usd":12}},{"id":"P19","stop_after":"none"}]}`;
  `tests/fixtures/git/measure.md` = a MEASURE table whose rows begin `| P16 primer-go |` and `| P17 debt |` (no other
  phase ids).
- `docs/deps/go-sdk.md` (2–5 KB, from the vendored v1.8.0 source, every value run on it): the import path
  `github.com/modelcontextprotocol/go-sdk/mcp`; `Implementation{Name, Version}`; `NewServer(impl, *ServerOptions)`;
  `AddTool[In, Out](s, &Tool{Name, Description}, ToolHandlerFor[In, Out])` with the `json`/`jsonschema` struct tags and the
  empty-input idiom; `CallToolRequest`, `CallToolResult{Content: []Content{&TextContent{Text}}, IsError}`;
  `NewStreamableHTTPHandler(getServer, &StreamableHTTPOptions{Stateless: true, JSONResponse: true})` and the request headers
  a stateless POST needs; `NewInMemoryTransports`, `Server.Connect`, `NewClient`, `Client.Connect`, `ClientSession.ListTools /
  CallTool / Close`; one JSON-RPC `tools/call` request and its response as the SDK writes them (measured).
- Regulation files from MorphV2 `templates/common` instantiated for morphd: `docs/AUTONOMY.md` (its "State at handoff"
  names P1; smoke stops after P3 and P7), `docs/MEASURE.md`, `docs/TASK_TEMPLATE.md`, `CLAUDE.md`, `tools/` (the tmux
  flow still builds morphd itself: morphd replaces it only once it runs), `.claude/settings.json`. `README.md` rewritten
  for morphd (data).

## Zero Contour

`contour.yaml` (≈ 158 KB, 17 Components, 27 Functions, 199 examples; sizes by YAML dump of each Component):

| Component | KB | Functions | examples |
|---|---|---|---|
| stream | 10.6 | 2 | 16 |
| session | 9.9 | 2 | 14 |
| eventlog | 4.1 | 1 | 9 |
| telegram | 4.5 | 1 | 8 |
| runner | 3.3 | 1 | 7 |
| queue | 5.8 | 1 | 8 |
| claude | 7.7 | 2 | 14 |
| github | 4.0 | 1 | 8 |
| bootstrap | 6.0 | 1 | 7 |
| registry | 7.1 | 2 | 13 |
| gitrules | 8.0 | 2 | 15 |
| control | 7.8 | 1 | 7 |
| supervisor | 17.6 | 2 | 17 |
| daemon | 17.2 | 2 | 16 |
| api | 10.7 | 2 | 15 |
| mcpserver | 12.4 | 3 | 20 |
| cmd | 5.0 | 1 | 5 |

Requirements: Claude Behind An Interface · Bearer Token On Every Route · The Session Reports Only Itself.
Guardrails: Injected Clock And Ids · No Network In Tests · No Shell Outside Runner · Secrets Never Logged.
Dependency: `github.com/modelcontextprotocol/go-sdk` v1.8.0 (go), doc `docs/deps/go-sdk.md`, used by mcpserver only.

## morph-map.json

Committed with this plan (54 entries). The cards of P1:

| card | target | slice | depends on | max_tokens |
|---|---|---|---|---|
| parse-event | stream/parse.go | go.mod | — | 12 000 |
| encode-lines | stream/encode.go | go.mod, stream/parse.go | parse-event | 8 000 |
| exec-runner | runner/runner.go | go.mod | — | 10 000 |
| phase-queue | queue/queue.go | go.mod | — | 12 000 |
| parse-event-judge | stream/parse_examples_test.go (derived) | go.mod, stream/parse.go, testhelp.go, probe.go, fixtures/stream/probe.jsonl | parse-event | 20 000 |
| encode-lines-judge | stream/encode_examples_test.go | go.mod, encode.go, parse.go, testhelp.go, probe.go, probe.jsonl | encode-lines | 16 000 |
| exec-runner-judge | runner/runner_examples_test.go | go.mod, runner.go, testhelp.go | exec-runner | 14 000 |
| phase-queue-judge | queue/queue_examples_test.go | go.mod, queue.go, testhelp.go, fixtures/plan/queue-3.json | phase-queue | 18 000 |

Budgets are before the processor's ×3 (`scale_tokens.py … 3` on `ds`); the heaviest judges: daemon-core-judge 32 000,
http-handlers-judge 28 000, pump-judge 28 000, pm-tools-judge 28 000.

## Verification

The operator knows the framework is done when smoke stop 2 is green as written above: `morphd` on the VPS answers
`GET /healthz` `{"status":"ok"}`, refuses `GET /projects` without the token (401), runs the one-phase plan of the
`smoke` project through a real claude session to `phase_done` and its stop, serves `status`, `events`, `order`,
`interrupt`, `usage` over HTTP and the same `status` over the remote MCP from the laptop, posts the start and stop
lines to Telegram with the `[smoke]` prefix, and `GOFLAGS=-mod=vendor GOPROXY=off go test -count=1 ./...` is green
with ≈ 199 example tests (one per record example; the count is the sum of the examples of the Functions merged).

## Record rules for this project

1. The record holds only the current contract: no history.
2. Large literals live in fixtures; one Component ≤ 30 KB, else two. Every stream or session example names its probe
   line; a new protocol fact is a new line in a new probe fixture, never an edited old one.
3. Cross-Component dependencies are `uses: <component>` steps; `calls:` names a Function of the same Component (V2
   `functionLinks`); the file-level dependencies are the map's `context_slice` and `depends_on`.
4. Every time is a `now` argument or an injected `Now`; every id an injected `NewID`; every external process behind
   `runner.Runner` or `claude.Process`; every HTTP call behind an injected `Do`. The fakes live in the code files
   (`runner.Fake`, `claude.Fake`) so other packages' judges can use them.
5. Error texts: a package prefix (`stream: `, `eventlog: `, `queue: `, `bootstrap: `, `registry: `, `telegram: `,
   `github: `, `claude: `, `config: `), sentinels as package variables compared with `errors.Is`, API bodies
   `{"error":{"code":…,"message":…}}` + `"\n"`, every body written by `json.Encoder`.
6. Secrets: files of mode 0600 under the state dir; never in a response, a log line, a milestone, an error, or an argv.
7. Layers: `decks/tools/layers.json` is the only import table; a new package is a plan change.
