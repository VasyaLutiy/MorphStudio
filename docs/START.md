# Brief — Morph Studio, first goal (draft, PM 08.10)

## Goal (operator 08.10)
"Убрать ssh flow для взаимодействия с claude внутри VPS … сделать это профессионально на Golang."
A Go backend on the VPS that drives the claude CLI session; base: the Morph-written Go skeleton
(github.com/VasyaLutiy/MorphStudio, P15L gocrud).

## Who uses it
The PM (Claude Code on the operator's laptop) and the operator. One user = one Hetzner VPS (DECISIONS 08.10).

## Today's ssh flow → backend (filled as the operator describes each step)
| # | Today | Backend |
|---|---|---|
| 1 | look: `tmux capture-pane -pt morph \| tail -20` (`-S -60` scrollback) | GET events?since=N; GET status |
| 3 | order when waiting at ❯: `send-keys -l "<text>"`, sleep 2, `send-keys C-m`; only when idle, else it queues or derails | POST messages {text} → one stream-json user line; while busy → queued by the backend, sent on `result` |
| 4 | a key: Enter = option 1 in a dialog ("Gate waiver"), Escape = close /usage; Escape in a working session interrupts it | dialog → POST answer {option}; no /usage panel in headless; interrupt only as an explicit POST interrupt |
| 5 | restart from scratch (new flow): `git pull --ff-only`, `tmux kill-session`, sleep 2, `tools/vps-start.sh start` → tmux + `claude --dangerously-skip-permissions "$(cat tools/vps-prompt.txt)"`, screen → ~/morph-logs/session-<date>.log, TG 🚀 | POST sessions/restart: refuse while busy unless force (interrupt, then stop); git pull ff-only; new process, new session id, first user line = vps-prompt.txt; events → log file; TG 🚀 |
| 7 | phase handoff + watchdog: the session writes MEASURE/DECISIONS/TASK §11, merges, pushes, rewrites "State at handoff", TG 🔀, then `touch ~/.morph-phase-done` (or `~/.morph-wait-operator` before a stop); cron */10 kills tmux and runs vps-start.sh; otherwise kicks on a frozen limit screen (≤12) or 30 min of no change. Known gaps (operator): ≤10 min delay, two 🚀, kills without checking the session finished, cron PATH/TERM untested (first live switch P17→P18 worked: session 12:30 took P18) | the session tells the daemon itself (`phase_done {phase, next}` / `wait_operator {reason}`); the daemon acts only after the turn's `result` and after it checks the push (HEAD == origin/main, clean tree), then starts the fresh process at once, one 🚀; wait → state waiting, TG, no kicks; process exit seen at once (no polling): unexpected exit → `--resume`, ≤ N per hour, then TG; limit → `rate_limit_event` status + resetsAt → resume at the reset, not blind kicks; stall = busy with no events for 30 min → TG, one nudge |
| 6 | /usage: only by eye — ssh, tmux, open /usage, read, Escape (operator: "считаю правильным сделать это внутри Go backend") | GET usage: five_hour / seven_day used % + resets_at from the stream's `rate_limit_event` (in CLI 2.1.280, fields to confirm by probe) + the session's own sum of total_cost_usd (list price); TG alert at a threshold (50% weekly, skill §5) |

## v1 scope (operator approved 08.10)
Build `morphd`: session lifecycle, events, orders, answers, interrupt, usage, TG milestones, HTTP API with a token, remote MCP over it (operator 08.10).

## Not built (operator approved 08.10)
Web wizard U0–U4, Hetzner VPS/snapshot provisioning, multi-user, billing, any browser UI.
## Rules (to ask)
## Scenarios with values (to ask)
## Stack
Go, stdlib first; external modules only declared in the record and vendored by P0 (MorphV2 P19). The MCP server on the official Go MCP SDK (operator 08.10). Tests never reach the network or a real claude.

## Protocol facts (probe on the VPS 08.10, CLI 2.1.294, haiku, log `docs/probe/stream-json-2.1.294.jsonl`)
- Start: `claude -p --input-format stream-json --output-format stream-json --verbose --include-hook-events --permission-prompt-tool stdio --session-id <uuid>`; first line in: `control_request {subtype: initialize}` → `control_response` (commands, …).
- Order: `{"type":"user","message":{"role":"user","content":"…"},"parent_tool_use_id":null}`; each turn ends with one `result`.
- Interrupt: `control_request {subtype: interrupt}` → `control_response {still_queued: []}`, then `result` subtype `error_during_execution`, `terminal_reason: aborted_streaming`.
- Question (AskUserQuestion): out `control_request {subtype: can_use_tool, tool_name: AskUserQuestion, input.questions[], requires_user_interaction: true}`; answer `control_response {behavior: allow, updatedInput: input + answers{question: label}}` → the session got "Option B".
- Limits: `rate_limit_event.rate_limit_info.unifiedWindows.{five_hour,seven_day}.{utilization 0–1, resetsAt epoch s}` (probe saw 0.22 / 0.60).
- result: subtype, is_error, num_turns, total_cost_usd (cumulative per session), usage, modelUsage, session_id, stop_reason, terminal_reason, duration_ms, permission_denials, queued_turn_count.
- Probe cost $0.0064 list price, 4 turns. No hook events appeared (no hooks configured for that run).

## Session per phase (discussion 08.10, operator: "на каждую фазу стартует чистая сессия /morph-orchestrator без контекста … после этого сброс")
- Input: the approved plan. The PM drives (operator 08.10): after the approval the PM loads the phase queue into morphd through MCP (`plan_load {approved hash, phases: id, stop after none/smoke/operator, caps}`), transcribed from the architect's PLAN table; morphd runs it on its own between stops; at a stop it waits for the PM (`stop_check`, `continue`). PLAN changes → new approval → new `plan_load`.
- Start: new process, new session id, never `--resume`/`--continue` across phases; first message `/morph-orchestrator <phase>` by AUTONOMY.
- Inside a phase: the session reports steps to morphd (prepare, gate, run, verify, merge) through morphd's own tools; morphd forwards to TG (replaces tg.sh).
- End: `phase_done` → morphd checks result event, pushed main, clean tree, the phase's MEASURE row → kills the process (the reset) → next phase, or waits when the plan marks a stop. Stops are enforced by morphd from the plan, not by the session.
- Crash inside a phase: `--resume` the same session id, ≤ N; limit: pause until resetsAt, resume the same session.
- Caps per phase: Claude `--max-budget-usd`, wall clock; executor $ from the morph run report.
- Claude auto-memory in phase sessions: off (operator 08.10). Caps default: Claude $30 list, 3 h (operator 08.10).

## Rules (operator 08.10)
- API: token header, Caddy TLS in front, morphd on localhost only.
- Telegram: morphd sends milestones itself (Bot API, token from .env); no tg.sh.

## Scenarios with values (operator approved 08.10; more may come later)
| # | Action | morphd answer |
|---|---|---|
| 1 | PM calls `status` | `{state: "busy", phase: "P18", minutes: 37, cost_usd: 1.92, five_hour: 22, seven_day: 60}` |
| 2 | `order "smoke checked green. Resume with P12b"` while the session waits; the same while busy | `{sent: true}`; `{queued: 1}`, sent after `result` |
| 3 | the session asks "Gate waiver" with 2 options | TG 🚦; PM `pending` shows the question; `answer {option: 1}` → the session continues |
| 4 | the session calls `phase_done {phase: "P17", next: "P18"}`, push checked | a fresh session at once, TG "🚀 P18" |
| 5 | `phase_done`, main not pushed | no new session, TG "🛑 P17 not pushed", state waiting for the operator |

## Git of one phase (operator 08.10, from MorphV2 AUTONOMY "The cycle of one phase"; Morph's mandatory flow, morphd wraps it, does not change it)
| Step | Who | Git |
|---|---|---|
| before start | today the operator/PM by hand → **morphd** | `git pull --ff-only` on main, then the session |
| 1 prepare | prep agent | phase data committed locally on main, no push |
| 2 gate | main session | no commit; a failed gate → MEASURE row + stop |
| 3 run | `morph run` | branch `morph/<runId>`, one commit per card (Morph-Card/Morph-Model trailers), archive `.morph/runs/<runId>/` |
| 4 verify | main session | on the run branch: clean status, the profile's checks, code read |
| 5 record | main session | one commit: TASK §11, MEASURE, DECISIONS, "State at handoff" |
| 6 merge | main session | `checkout main && merge --ff-only morph/<runId> && push origin main`; no ff → stop with a report |
morphd rules from it:
- morphd never commits, merges or pushes; git inside a phase is the session's.
- Phase start: fetch; local main == origin/main → fresh session; local main ahead (unpushed prep commits = an interrupted phase) → `--resume` that phase, never pull or reset; diverged → stop, TG 🛑.
- Phase end (`phase_done`): main == origin/main, clean tree, the phase's MEASURE row present, no `morph/<runId>` left unmerged → reset, next phase.

## What morphd takes over from a Morph project's AUTONOMY (read MorphV2 docs/AUTONOMY.md 08.10, operator's permission)
The regulation stays the project's (AUTONOMY from `morph init`, P18); morphd replaces only its machine parts:
| AUTONOMY today | morphd |
|---|---|
| "State at handoff" names the next phase; flags `~/.morph-phase-done` / `~/.morph-wait-operator` | the queue from `plan_load`; tools `phase_done` / `wait_operator {kind: smoke/gate/emergency/no-next, reason, issue url}` |
| smoke stops listed by phase inside AUTONOMY | `stop after` per phase in the queue; the session never resumes itself, morphd waits for the PM's `continue` |
| `tools/tg.sh <kind> "<headline>" "<numbers>"`, kinds start 🚀 gate 🚦 run 🏁 fail ❌ merge 🔀 smoke 🧪 stop 🛑 debt 💸 end 🎉 info 💬, watchdog 🐕, hooks idle 💤 ask 🔔 | tool `milestone {kind, headline, numbers}` with the same kinds and icons → TG and the event log; idle/ask from the stream (result, can_use_tool) |
| tmux `morph`, `vps-start.sh attach/log` | the event log + `GET events` |
| hooks Stop/Notification in .claude/settings.json | not needed: the stream carries both |
| money: $5 per phase, $30 the stretch, the running total in MEASURE | executor $ per phase from the run report; the stretch total enforced by morphd (stop at the cap); Claude $ per phase via `--max-budget-usd` |
Not morphd's: the cycle steps 1–6, the failure classes, the debt payment, the processor and binary-copy rules — all inside the session.

## Projects (operator 08.10)
- Many projects per user, one folder each (`~/projects/<name>/`, its own git repo on the user's GitHub, fine-grained PAT for that repo only).
- morphd: a registry; per project its queue, state, event log, caps, sessions; `project` in every call; TG `[<project>]`.
- One active session per VPS, a global queue across projects; max-parallel setting (default 1) so parallel runs come later without a redesign.
- `project_create {name, language, repo_url}`: clone the empty repo, `morph init --name --language`, first push (the PAT cannot create repos).

## How `project` reaches morphd (operator approved 08.10)
- The agent never types `project`: the MCP endpoint is bound to it. Per project folder on the laptop `.mcp.json` (written by `claude mcp add --scope project --transport http morph https://<host>/mcp/<project> --header "Authorization: Bearer ${MORPH_TOKEN}"`) → tools status/order/answer/interrupt/usage/plan_load/continue have no `project` argument.
- One user-level endpoint `https://<host>/mcp` (scope user) with only registry tools: `projects_list`, `project_create` — for a Start before the folder exists.
- The VPS session gets its own endpoint from morphd at launch: `--mcp-config` → `http://127.0.0.1:<port>/mcp/<project>/session` with a per-session token; tools phase_done/wait_operator/milestone; it cannot report for another project or a past session.
- Inside, the HTTP API keeps it in the path: `/projects/{project}/…`.
- GitHub token (operator 08.10): `PUT /projects/{project}/github-token` (HTTP only, not an MCP tool, so the token never enters a model's context); morphd stores it per project (mode 600), never returns it; `status` reports the repo reachable + push access (401/403/404 → a plain reason).
- Probe 2 (08.10, skip-permissions): AskUserQuestion still comes as can_use_tool; other tools need no answer. Log `docs/probe/stream-json-2.1.294-skip.jsonl`.
