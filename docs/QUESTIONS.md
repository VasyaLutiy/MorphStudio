# Open questions — morphd design (architect, 08.10)

**Answered by the operator 08.10** (relayed by the PM): every proposal accepted as written, except Q4 (changed:
auto-memory off is a hard rule — Launch Args `Env` always sets `CLAUDE_CODE_DISABLE_AUTO_MEMORY=1`, never a
configuration; the AskUserQuestion-under-skip-permissions check is the PM's before approval) and Q7 (PM fact, corrected 08.10: `morph
init --root <dir>` with flags `--name --language --module --templates`; Go gets `--module <name>`; the target must be
empty, a `.git` entry too). The
decisions table in `PLAN.md` carries the status; the text below is kept as the record of what was asked.

Each question: what the brief does not answer, the architect's proposal (the record uses it, marked "(proposed, Q<n>)"
in PLAN's decisions table), and the Functions that depend on the answer. The PM asks; the operator's answer becomes a
line in `docs/DECISIONS.md` and, where it differs from the proposal, a record change before approval.

1. **Go toolchain.** The official Go MCP SDK needs Go ≥ 1.23 at every version and Go 1.25 from v1.6.1 on (v1.8.0,
   the latest, 2026-09-04); the repository's `go.mod` and the VPS are on Go 1.22 (the Morph go profile measured 1.22.2).
   Proposal: install Go 1.25.x on the VPS by hand at P0 (`GOTOOLCHAIN=local` stays), set `go 1.25` in `go.mod`, declare
   v1.8.0. Alternative: v1.2.0 on Go 1.23 (older protocol support, no `http.CrossOriginProtection` need). Depends: the
   whole scaffold, mcpserver (PM Tools, Session Tools, MCP Mount).
2. **The P15L skeleton.** Packages `project`, `store`, `auth`, `api` (87 tests, Basic auth, a CRUD of slugs) serve no
   Function of morphd; keeping them keeps dead layers in the guard and 27 KB of contract nobody reads. Proposal: remove
   them at P0 (a data commit); keep the module path `morphstudio`, `internal/testhelp`, the guard files, `.morph/runs`
   (the P15L history) and the README's "how it was built" paragraph rewritten for morphd. Depends: P0, the layer table.
3. **The first user line of a phase session.** START row 5 says `vps-prompt.txt`, START "Session per phase" says
   `/morph-orchestrator <phase>` by AUTONOMY. Proposal: `/morph-orchestrator <phase>` (the later, more specific
   statement); `vps-prompt.txt` belongs to the tmux flow morphd replaces. Depends: Phase Loop (`first_line`).
4. **claude launch flags.** The probe ran with `--permission-mode default --permission-prompt-tool stdio`; the VPS flow
   runs `--dangerously-skip-permissions`. Does an `AskUserQuestion` still arrive as a `can_use_tool` control request
   under skip-permissions? And how is auto-memory turned off (operator: off, 100%)? Proposal: launch with both
   `--dangerously-skip-permissions` and `--permission-prompt-tool stdio`; morphd auto-allows every `can_use_tool` but
   `AskUserQuestion`; the memory switch goes through `MORPHD_CLAUDE_EXTRA_ARGS` once the PM confirms the flag on 2.1.294
   (smoke stop 1 checks the question path). Depends: Launch Args, Session Machine.
5. **`plan_load` starts at once.** The brief: "the PM loads the queue … morphd runs it on its own between stops".
   Proposal: `plan_load` begins the first phase immediately (no `start` tool); a load while a phase runs is refused
   (`busy`). Depends: Daemon Core (PlanLoad), PM Tools.
6. **The Telegram kind of a session question.** Scenario 3 says "TG 🚦"; AUTONOMY's hook kind for a question is `ask`
   🔔 and 🚦 is the gate. Proposal: `ask` 🔔, headline = the question text, numbers = the numbered options. Depends:
   Session Machine (the `ask` action), Pump.
7. **The `morph init` argv.** START says `morph init --name --language`; P18 takes a target directory. Proposal:
   `morph init <dir> --name <name> --language <language>` run in the projects directory; the PM confirms against the
   P18 CLI before P3. Depends: Project Create (step 1).
8. **The first push vs. the token.** `project_create {name, language, repo_url}` must push, but the PAT reaches morphd
   only by `PUT /projects/{p}/github-token`, which needs the project to exist. Proposal: create = `morph init` + git
   init + credential helper (`store --file=<state>/<p>/git-credentials`) + remote + first commit, no push; the push
   runs at the token PUT when the access check reports push (`FirstPush`), and the PUT answers
   `{"access": …, "pushed": true|false, "push_error": …}`. Depends: Project Create, Daemon Core (PutGithubToken),
   Control Contract (TokenResult), HTTP Handlers.
9. **`restart` semantics.** Today's flow restarts "from scratch" (kill, pull, new session). morphd never resets git.
   Proposal: restart = kill the process, run the start check; a tree that is ahead or dirty yields `--resume` of the
   same session id, an equal tree a fresh session id; refused while a turn is in flight unless `force`. Depends:
   Phase Loop (Restart), Daemon Core.
10. **Git identity of the create commit.** Proposal: `user.name morphd`, `user.email morphd@localhost` in the project's
    local config; the phase sessions commit with the VPS user's own git config as today. Depends: Project Create.
11. **`.env` location and keys.** Proposal: `.env` in morphd's working directory (the systemd `WorkingDirectory`, e.g.
    `/home/morph`), environment variables win; keys `MORPHD_TOKEN` (required), `MORPHD_PORT` 7080, `MORPHD_STATE_DIR`
    `~/.local/state/morphd`, `MORPHD_PROJECTS_DIR` `~/projects`, `MORPHD_CLAUDE_BIN`, `MORPHD_MORPH_BIN`, `TG_BOT_TOKEN`,
    `TG_CHAT_ID`, `MORPHD_MODEL`, `MORPHD_CLAUDE_EXTRA_ARGS`, `MORPHD_CLAUDE_USD` 30, `MORPHD_HOURS` 3,
    `MORPHD_EXECUTOR_USD` 5, `MORPHD_STRETCH_USD` 30, `MORPHD_MAX_PARALLEL` 1, `MORPHD_RESUMES_PER_HOUR` 3,
    `MORPHD_STALL_MINUTES` 30, `MORPHD_USAGE_ALERT` 50. Depends: Config And Main.
12. **The executor $ per phase.** The brief: "executor $ per phase from the run report; the stretch total enforced by
    morphd". morphd cannot read `.morph/runs/<id>/report.json` without the session telling it which run. Proposal: in
    v1 morphd enforces the stretch cap on the claude list-price total only (`total_cost_usd` summed over phases); the
    session posts the executor $ in its `milestone run` numbers as today; executor enforcement is a later phase.
    Depends: Runtime Guard (Turn), the stretch cap default.
13. **The stall rule.** Proposal: a phase gets one nudge (a user line) at 30 min without events and a 🐕 line; no
    second nudge; the wall-clock cap (3 h) ends a frozen phase. Depends: Runtime Guard (Tick).
14. **The not-pushed stop.** Proposal: the process stays alive (the session may still be finishing), state `waiting`,
    🛑; `continue` re-runs the end check instead of restarting the phase. Depends: Phase Loop (Continue), Daemon Core.
15. **When the repo access is checked.** Proposal: at the token PUT only; `status.repo` is the last check with its time;
    a periodic re-check is later. Depends: Daemon Core (Status), Repo Access.
16. **Event log retention.** Proposal: one JSONL file per session id under `<state>/<project>/sessions/`, appended on
    resume, never rotated in v1; `GET events` serves the last 2000 entries from memory. Depends: Event Log, Pump.
17. **The rate-limit pause rule.** The probe saw only `status: "allowed"` with utilization 0.22 / 0.6; a limited
    session was never observed. Proposal: pause when a window's `utilization >= 1`, until its `resetsAt`, then one
    "the usage window has reset" line; the first real limit is recorded and the rule adjusted by a DECISIONS line.
    Depends: Runtime Guard (Limits, Tick), Parse Event.
18. **Queued projects under max-parallel 1.** Proposal: a second project's `plan_load` answers `queued`; it starts at
    the next daemon Tick (30 s) once no project is running/starting/paused. Depends: Daemon Core (PlanLoad, Tick).

**Added 09.10 (P7b, after smoke stop 2 RED — D1 session id, D2 invisible crash / restart storm, D3 limits before any
event).** Proposals used in the record marked "(proposed, Q<n>)" in PLAN's decisions table until the operator answers.

19. **What resets the per-hour restart count.** The guard restarts a session that exits on its own (fresh or `--resume`,
    both count) at most `MORPHD_RESUMES_PER_HOUR` times per hour, then stops the project with kind `crash`. Does the PM's
    explicit `restart` and the operator's `continue` after such a stop reset the count? Proposal: yes — `plan_load`,
    `continue` (Begin) and `restart` set the list to nil (a human acted; the next storm gets its own 3 attempts); a
    fresh start check never resets it. Depends: Phase Loop (Begin, Restart, StartChecked), Runtime Guard (Exited).
20. **One id source.** Only the claude session id must be a version 4 UUID; the session token and the interrupt
    request id could stay 32 hex chars. Proposal: one `newID` for all three (a UUID is 122 random bits, enough for a
    bearer token; one function, one example set, nothing to confuse). Depends: Config And Main (newID), Daemon Core.
21. **The name `MORPHD_RESUMES_PER_HOUR`.** The cap now counts every restart, not resumes only. Proposal: keep the env
    key and `Config.ResumesPerHour` / `supervisor.Config.MaxResumesPerHour` (the VPS `.env` and the morph-pm skill use the
    name; renaming buys nothing but a config break); the loop's own list is renamed `Restarts` (`"restarts"` in
    `state.json`). Depends: Config And Main, Runtime Guard.
22. **Status after a crash.** The brief asks for "a crashed/restarting state with the exit code". Proposal: no new value
    of `state` (the loop is `starting` for the instant of the restart and `busy` again at once); instead `status` gains
    `last_exit` `{code, at, stderr, restarts}` (absent until an exit), the stop past the cap is `{kind: "crash", reason:
    "exited 4 times within an hour (last code 1): <first stderr line>"}`, every exit is an `exit` entry of the session's
    event log with the last 20 stderr lines, and every restart is a 🐕 watchdog line "restart n of 3 this hour · <first
    stderr line>". Depends: Control Contract (Exit, Status), Runtime Guard, Pump.

**Added 10.10 (P8a/P8b, the minimal PM MCP harness — operator items 2, 3, 4, 5).** Proposals used in the record marked
"(proposed, Q<n>)" in PLAN's decisions table until the operator answers.

23. **What adopt writes into an existing repository.** `project_adopt` must register a repo that already exists (MorphStudio
    itself) without init, commit or push, but the phase sessions push with the PAT the operator PUTs later. Proposal:
    adopt runs exactly one write — `git config credential.helper "store --file=<state>/<p>/git-credentials"` (git keeps
    the user's own helpers and adds this one, so the morph user's existing credentials still work); no `user.name`/
    `user.email`, no `morph init`, no commit. Alternative: write nothing and rely on the morph user's own credentials.
    Depends: Project Adopt (step 4), Daemon Core (AdoptProject).
24. **Where adopt lives.** Proposal: HTTP `POST /projects/adopt` (a literal segment beside `POST /projects`; a project
    could not be named "adopt" through that route — nothing else conflicts) and MCP `project_adopt` on `/mcp`; the
    `control.Control` interface stays at 17 methods (every judge's fake implements it) and `AdoptProject` is the optional
    `control.Adopter` that `control.Adopt` resolves by a type assertion (a Control without it → 500 `internal` "adopt not
    supported", the state between P8a and P8b). Alternative: an 18th method on Control — 4 more re-cuts (control-contract
    + judge, session-tools-judge, …) for the same behaviour. Depends: Adopt Contract, HTTP Handlers, Router, PM Tools.
25. **The wait for the last turn.** Proposal: `MORPHD_DONE_WAIT` 30 s (0 = the old behaviour); the daemon holds the
    `phase_done` end batch until the turn's `result` (or the process's exit), then runs the end check, the kill and the
    next phase; at the deadline it runs it anyway (that turn's cost is then lost, as today). The regulation asks the
    session to make `phase_done` the last tool call of its turn and end the turn at once. Known gap: a daemon restart
    inside the wait loses the pending batch (the loop is "running" → `Exited(-2)` → the phase restarts by the old rule).
    Depends: Daemon Core (PhaseDone, Tick), Pump, Config And Main.
26. **The `events` tool's bounds.** Proposal: text and stderr cut at 200 runes + "…", tool names whole, `limit` absent → 50,
    `since` absent → 0; the HTTP `GET events` keeps the raw entries for the operator's curl. Depends: Event View, PM Tools.
27. **Caps of an adopted project.** Proposal: `Config.Defaults` as for a created project (the PM's `plan_load` carries
    per-phase caps anyway). Depends: Daemon Core (AdoptProject).
28. **The Claude Code hooks under morphd.** `.claude/settings.json` posts 💤 on Stop and 🔔 on Notification through
    `tools/tg.sh`; the daemon posts both itself (the pump's `idle` and `ask`). Proposal: on the VPS remove
    `~/.config/morph/tg.env` for the morph user (then `tg.sh` is a silent no-op) — a PM/operator action, not a card; the
    laptop flow keeps it. Depends: nothing in the record; `docs/AUTONOMY_MORPHD.md` "Observability".
29. **Which regulation a session reads.** `docs/AUTONOMY.md` keeps the tmux/cron text for the laptop flow;
    `docs/AUTONOMY_MORPHD.md` is the morphd variant. Proposal: the rule "when the MCP server `morphd` is in the session's
    tools, AUTONOMY_MORPHD.md overrides the sections it names" is written at the top of AUTONOMY_MORPHD.md; a one-line
    pointer at the top of `docs/AUTONOMY.md` is the PM's data edit after approval (the architect did not touch that file).
    Depends: nothing in the record.
