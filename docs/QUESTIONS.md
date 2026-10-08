# Open questions — morphd design (architect, 08.10)

**Answered by the operator 08.10** (relayed by the PM): every proposal accepted as written, except Q4 (changed:
auto-memory off is a hard rule — Launch Args `Env` always sets `CLAUDE_CODE_DISABLE_AUTO_MEMORY=1`, never a
configuration; the AskUserQuestion-under-skip-permissions check is the PM's before approval) and Q7 (PM fact: `morph
init` takes one positional dir and flags `--name --language --module --templates`; Go gets `--module <name>`). The
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
