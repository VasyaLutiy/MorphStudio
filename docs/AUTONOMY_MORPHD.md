# Autonomous mode under morphd

The morphd variant of the sections of `docs/AUTONOMY.md` a phase session reads. **Rule**: when the MCP server `morphd`
is among the session's tools (the tools `milestone`, `phase_done`, `wait_operator`), this file overrides the sections
it names below; every other section of `docs/AUTONOMY.md` (Shell hygiene, The cycle of one phase, Dependencies,
Decisions the session makes alone, Money, What never happens, Order of the phases) applies unchanged. Without that
server the session is the laptop/tmux flow and reads `docs/AUTONOMY.md` alone. Data, not code: no card writes this
file; the operator can change any line.

## Machine (morphd)

- User `morph`, the repository `/home/morph/MorphStudio` (origin `github.com/VasyaLutiy/MorphStudio`, `main`, pushed
  only by fast-forward). Go 1.25 at `/usr/local/bin/go`; every Go command offline with `GOFLAGS=-mod=vendor GOPROXY=off`.
- The session is a child of morphd (systemd unit `morphd.service`, `/home/morph/morphd`): `claude -p` in stream-json
  mode, `--dangerously-skip-permissions`, `--max-budget-usd <cap>`, `CLAUDE_CODE_DISABLE_AUTO_MEMORY=1`, a fresh
  session id per phase, the MCP server `morphd` from `--mcp-config` (the per-session endpoint with the token minted for
  this spawn — it reports this session only). There is no tmux and no screen log: every line in and out is the
  daemon's event log (`<state>/<project>/sessions/<session id>.jsonl`), which the PM reads with `events`.
- The `morph` binary: MorphV2 at `/home/morph/MorphV2` is read-only for this project (never edit, commit, build in
  place, or touch its tmux `morph`). The run uses the COPY `/home/morph/morph-bin/dist` (what `MORPHD_MORPH_BIN`
  points at; refreshed by hand by the PM when MorphV2 moves); the recipe of the binary copy per phase in
  `docs/AUTONOMY.md` "State at handoff" still works for a phase-local copy.
- Processor environment: as in `docs/AUTONOMY.md` "Machine" — exported by indirection in a subshell from
  `/home/morph/MorphProject/morph-lab/.env`, never printed. morphd's own `.env` holds no processor key.
- Anything longer than a minute runs under `nohup` with a log file; the daemon never kills a session for being busy,
  only at `phase_done`, `restart`, a cap or a crash storm.

## One phase, one session (morphd)

Every phase gets a fresh session; the memory of a phase is in the repository (record, TASK §11, MEASURE, DECISIONS,
`docs/AUTONOMY.md` "State at handoff"), never in the session's context. The first user line is
`/morph-orchestrator <phase>`; the phase is the one the daemon's queue names (the PM loaded it from PLAN's table with
`plan_load`, which also carries the caps and the stops) — it must equal the phase "State at handoff" names; a
mismatch is a stop (`wait_operator {kind: "no-next", reason: "handoff names <x>, queue runs <y>"}`).

The session works the phase, records it, merges it, rewrites "State at handoff" for the next phase and pushes. Then
it ends in one of two ways, and in both the LAST tool call of the turn is the report, followed by one line of text and
the end of the turn — nothing else runs after it:

- **the phase is done** (whatever the plan says comes next — the daemon knows the stops): `phase_done {phase: "<this
  phase>", next: "<the next phase of State at handoff, or "" when none>"}`. The daemon answers `{"ok":true}` at once
  and then, when this turn's `result` arrives (or after `MORPHD_DONE_WAIT` seconds, default 30, if it does not), runs
  the end check — `main` pushed (`HEAD == origin/main`), tree clean, no `morph/*` branch left unmerged, the phase's
  row in `docs/MEASURE.md` — kills the process (the reset) and either starts the next phase's fresh session itself
  (one 🚀, no watchdog, no delay) or stops where the plan says (🛑 "done: stop after smoke/operator", 🎉 "plan
  complete") and waits for the PM's `continue`. A red end check posts 🛑 "<phase> not pushed" with the problems and
  waits; the PM's `continue` re-runs the check — the session is still alive then and may be ordered to fix the push.
  Never call `phase_done` twice, never keep working after it: the turn's cost after the call is counted only until
  the `result`.
- **a stop the plan does not know** (a gate failure, an emergency stop, an environment red, the handoff mismatch
  above, a smoke the session must run itself): `wait_operator {kind, reason, issue_url}` with kind `gate`, `emergency`,
  `smoke` or `no-next` and a one-line reason (the issue URL when one was opened). The daemon posts 🛑 "<phase>:
  wait_operator <kind>", keeps the process alive and the state `waiting`; nothing nudges it; the PM's `continue` kills
  the process and begins the phase again on a fresh session (the phase re-reads "State at handoff": write there what
  the next session must do). Smoke stops that PLAN marks are the plan's `stop_after: smoke`: the session calls
  `phase_done` normally and the PM runs the smoke at the stop.

There are no `~/.morph-phase-done` / `~/.morph-wait-operator` markers and no `tools/vps-start.sh` under morphd; the
watchdog is the daemon's Runtime Guard (below). `tools/tg.sh` is not called by the session (see Observability).

## What the daemon does on its own (replaces the watchdog)

- **An exit before `phase_done` / `wait_operator` is a crash**: the daemon logs the exit code and stderr (an `exit`
  entry in the event log, `last_exit` in `status`), posts 🐕 "restart n of 3 this hour · <first stderr line>" and
  restarts within the hour's cap — `--resume <the same session id>` when the tree is dirty, on a branch or ahead of
  `origin/main`, a fresh session otherwise. A resumed session gets NO first line: it continues from its own context by
  this regulation (post the state first, then go on). Past the cap: 🛑 "<phase>: session keeps exiting", state
  `waiting`, the PM's `continue` gives a new round.
- **Usage limit**: a `rate_limit_event` with a window at 100% pauses the project until the window resets; the session
  then receives the user line "Continue by docs/AUTONOMY.md from where you stopped; the usage window has reset." —
  treat it as an order: post the state, continue. 50% of the weekly window → one 💬 line, nothing else.
- **Stall**: 30 minutes without an event from the session → one 🐕 line and one user line "Nothing has happened for
  30 minutes. Check the state of your agents and the run branch, then continue by docs/AUTONOMY.md; post the current
  state first." — once per phase; after it only the wall-clock cap acts.
- **Caps**: the Claude cap per phase (`--max-budget-usd`, default $30 list price) is claude's own; the wall clock (default
  3 h per phase) and the stretch total (default $30 across phases) are the daemon's: 🛑 "<phase>: wall clock cap" /
  "stretch cap", the process killed, state `waiting`. The executor cap ($5 per phase, $35 for the stretch, in
  `docs/MEASURE.md`) stays the session's own rule (`docs/AUTONOMY.md` "Money"): reaching it is `wait_operator
  {kind: "emergency", reason: "executor stretch $35 reached"}`.
- **Orders and answers from the PM** arrive as user lines (`order`) and as the chosen option of an `AskUserQuestion`
  (`answer`); the PM may `interrupt` a turn (the `result` then says `aborted_streaming`) and `restart` the session
  (kill + the start check: a dirty or ahead tree resumes the same conversation, an equal tree gets a fresh session and
  its first line again).

## Failure (morphd)

The classes, the one fix per class, the one paid debt per phase and the emergency stop are those of `docs/AUTONOMY.md`
"Failure", with the reports moved to the daemon:

- a gate that does not pass → the MEASURE row "stopped at gate: <reason>", then `wait_operator {kind: "gate", reason:
  "<reason>"}` (replaces `touch ~/.morph-wait-operator`);
- an emergency stop (a card still red after its one fix and the debt red, or an environment red after its re-run) →
  the issue (`gh issue create --label debt` / the Component's label), then `wait_operator {kind: "emergency", reason:
  "<card> <class>", issue_url: "<the issue>"}` (replaces the "EMERGENCY STOP" post: the daemon posts 🛑 with the
  reason and the URL);
- a debt paid green → `milestone {kind: "debt", headline: "<card> paid by <model>", numbers: "$… · <min> min"}` and the
  loop goes on;
- a smoke the session runs itself (PLAN names one inside a phase) → `milestone {kind: "smoke", …}` with the numbers,
  then `wait_operator {kind: "smoke", reason: "<what the PM must check>"}`; a red smoke is an emergency stop;
- a non-ff `main`, a push refused, a diverged tree → not the session's to fix: `wait_operator {kind: "emergency",
  reason: "git: <what git said>"}`. The daemon's own start check (fetch; equal → fresh, ahead → resume, diverged →
  🛑 "not started: git") and end check never commit, merge or push.

## Observability (morphd)

- The session reports through the MCP tool `milestone {kind, headline, numbers}`; the daemon prefixes `[<project>]` and
  the icon and posts to the operator's Telegram (the token is morphd's `.env`, never the session's). Kinds the
  session posts: `gate` 🚦 gate passed (with the forecast and the card count), `run` 🏁 run green (written, $, minutes,
  burned variants), `fail` ❌ run with red cards, `merge` 🔀 merge and push done, `debt` 💸, `smoke` 🧪 (a smoke it ran
  itself), `info` 💬 anything else. Example: `milestone {kind: "run", headline: "P8a bootstrap+control+api: run green",
  numbers: "10/10 written · $0.21 · 14 min · 0 burned"}`.
- Kinds the session never posts, because the daemon does: `start` 🚀 (the spawn), `stop` 🛑 (every stop: end check
  red, `wait_operator`, crash storm, caps, git), `end` 🎉 (plan complete), `watchdog` 🐕 (restarts, limit pause and
  resume, the stall nudge), `idle` 💤 (every turn's end, with the turn's text and cost), `ask` 🔔 (a question of the
  session with its options). `tools/tg.sh` and the Claude Code hooks of `.claude/settings.json` are the laptop flow's;
  under morphd they would double 💤 and 🔔 (Q28: the operator decides the `tg.env` on the VPS).
- The PM reads the session without ssh: `status` (state, phase, minutes, cost, limits, queue, repo, stop, last_exit),
  `events {since, limit}` (the compact view: type, short text, tool names, result cost, exit), `usage`, `pending` /
  `answer`, `stop_check` / `continue`. Nothing the session prints to stdout outside the stream is seen: a long
  command's output goes to a log file under /tmp and its summary into a `milestone` or the MEASURE row.

## State at handoff conventions (morphd)

The section "State at handoff" of `docs/AUTONOMY.md` is still rewritten by the session at the end of every phase, with
these conventions:

- its first line names the phase just done and its merge commit; its second the next phase exactly as PLAN's table
  names it (the id the PM loads into the queue; the daemon's `phase_done` closes the current one and begins the queue's
  next — `next` in the call is informational and must match), or "no next phase: <why>";
- the running totals (executor $ of the stretch, claude $ of the smokes, debts) as today; the daemon's own stretch
  counter (`usage.stretch_cost_usd`, list price of the claude sessions) is a second number, not a replacement;
- no marker files, no watchdog notes, no tmux recipe; what the next session must do before anything else (an unpushed
  commit, a leftover run branch, a smoke the PM must run) is written as the first bullet, because a resumed session
  reads only this and its own context;
- the "Lessons for the next preparations" list stays; a lesson about the daemon (a stop that should have been a
  milestone, a wait that was too short) goes to `docs/DECISIONS.md` with the phase and the tool call verbatim.
