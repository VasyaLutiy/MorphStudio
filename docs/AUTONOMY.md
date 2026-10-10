# Autonomous mode

> Under morphd (a session started by the daemon) the regulation is docs/AUTONOMY_MORPHD.md; this file is the laptop/tmux scheme.

The regulation that replaces the operator at every point where a human would answer during a phase. The operator
confirms it before the first autonomous phase and can change any line; the session reads it at the start of every phase.

## State at handoff

**P7b is done (10.10, merged by fast-forward a37d6d6).** After five red runs (VPS 20261010-070606; laptop old-scheme runs 084526, 092439, and on P24 123801, 133009 — see MEASURE and DECISIONS 10.10) the PM salvaged run 20261010-092439's archived answers with MorphV2 18dab5b's group `morph accept --from-run … --pick daemon-core-judge.r2.v1,pump-judge.r1.v1,config-and-main-judge.r1.v1 --commit`: 12/12 green on one tree, 12 commits with Morph-Card/Model/Variant/Run trailers, no Morph-Debt, $0. Verify: go build/vet ok, gofmt empty, 210 example tests in 17 packages, race ×3 ok on control, supervisor, daemon, cmd/morphd; D1 (UUID v4 session ids), D2 (crash = exit before phase_done/wait_operator, Restarts, Exited with stderr, crash stop) and D3 (null limits until the first rate_limit_event) are in the code. The VPS's unpushed 10.10 P7b prep commits are dropped (operator 10.10); the VPS binary for any later phase is MorphV2 ≥ 18dab5b.

**Next: smoke stop 2, final** (PLAN "Smoke stop 2, final": the PM on the VPS over HTTP and MCP, ≤ $0.50 of claude,
of which $0.1386 is spent on smoke 1). No phase remains after it: the session posted 🧪, touched
`~/.morph-wait-operator` and stopped; the PM runs the smoke and the operator restarts the session after a check (a
green smoke → `end` 🎉 and the stretch closes; a red one → a DECISIONS line and an issue labelled for the Component).
Running total $0.5473 of $35 executor; claude for the smokes $0.1386 of $0.55; claude for debts $2.4027 (P6).
Processor `ds` (maxTokens ×3 after the cut); fallback `glm53`. Claude auto-memory is off in every phase session
(`CLAUDE_CODE_DISABLE_AUTO_MEMORY=1`, set by `tools/vps-session.sh`).

P7 is done (run 20261008-211009, merged 3719ec7): packages `daemon` (daemon.go, pump.go) and `cmd/morphd` (config.go,
main.go); 201 example tests green (P1 31 + P2 32 + P3 29 + P4 35 + P5 32 + P6 20 + P7 22), race ×3 green on daemon and
cmd. Run: 6/6 on ds, $0.1401, 22.7 min, 2 retries (config-and-main `== listen`, daemon-core-judge vet/gofmt), 0 burned.
Prep by a fresh Opus agent: 360k tokens, 97 calls, 29 min; record changed (DECISIONS 08.10 · P7 "RECORD CHANGED":
startPump set by pump.go's init, one daemon mutex, the pump's line handling under it, the MCP path `morphd-mcp.json`,
inlined examples with varied constants, Daemon Core example 10 CreateProject → control sentinels, Continue's queue error
→ ErrNotWaiting, main passes `http.DefaultClient.Do`); `.gitignore` gains `/morphd`. **Smoke-2 risks for the PM**
(DECISIONS 08.10 · P7, TASK_P7 §11): a failing Spawn (wrong `MORPHD_CLAUDE_BIN`) or a claude dying at once respawns
without bound (Exited → start check "fresh" → spawn; a Spawn error recurses under d.mu to a stack overflow) — check the
claude binary first; "POST order while idle" may get 409 once S1's `phase_done` ended the session — order before
phase_done or give S1 two turns; a resumed session gets no first line (waits for the 30-min nudge); morphd listens on
127.0.0.1 only, so the laptop steps need the reverse proxy. Read defects (none pinned): PlanLoad drops Begin's error;
a spawn failing after its kill leaves the dead process current (its exit counted twice); Usage gives StretchCostUSD
without a machine; the pumps map is never pruned. Mutant survivors: PlanLoad with Caps hard-coded {30, 3, 5}; an old
process's lines applied to the new machine; Wait returning at once. `.morph/primer.md` re-generated after P7.

P6 is done (runs 20261008-200807, fix 20261008-201843, debt a1d948a by Claude Fable 5.1 xhigh $2.4027): package
`mcpserver`. Its known risks: a nil Projects() as [] is pinned by no example; pm.go exports `type Empty struct{}`;
Session Tools tests miss the call-list length, the second PhaseDone call and the single Content entry. Carried from P5:
Runtime Guard `Exited` prunes `Resumes`; Phase Loop fresh StartChecked appends only a "pulled to" Reason; Guard example
8 tautological; the crash-restart "fresh" path clears the resume list; the stretch total undercounts
stopped-then-continued and resumed sessions. From P4: Registry Add leaves a project in memory after a failed write;
Open's non-decode read error is unwrapped; DeleteSecret of an unknown project returns nil. P3's two stand (Process:
Write between Lines() closing and the ExitStatus returns a pipe error, not ErrExited; a stdout line over 16 MiB hangs
the exit until Kill). P2's two stand (Apply panics on a "result" with nil Result; Append's write error says "open");
P1's two stand (queue `Load` does not trim IDs; runner's cannot-start error lacks the name).

P1–P5 done (runs 20261008-171107, 20261008-173606, 20261008-181657, 20261008-185347, 20261008-193337). The phase dir is `decks/<PHASE>/`
(upper-case id); Go probes are `decks/<phase>/parts/_<card>_probe_test.go`. The binary copy per phase:
`/tmp/morph-bin-<PHASE>/` = `cp -r /home/morph/MorphV2/dist` + `package.json`, `node_modules` and `templates`
symlinked (MorphV2 bc311aa). The run script recipe: source `/home/morph/MorphProject/morph-lab/.env` in a subshell,
export `MORPH_PROCESSOR_ds_<KEY>` from `MRPH_PROCESSOR_ds_<KEY>` (lower-case `ds`, only keys that are set), unset
`MRPH_*`, run the binary copy under nohup with `--deadline 2400`, stdout (the Run Document) to
/tmp/morph-<PHASE>-run.stdout.json, stderr to /tmp/morph-<PHASE>-run.log, `exit=` appended to the log.

P0 (the scaffold, hand data) is done: Go 1.25.14, `go.mod` `go 1.25.0` with go-sdk v1.8.0 vendored (`vendor/`
committed), `internal/testhelp/probe.go`, `decks/tools/*` (layers by `PLAN.md` "Scaffold"; the per-card stdlib rules for
each phase's `checks.json` `extra` are in `layers.json` `stdlib_rules`), fixtures under `tests/fixtures/`,
`docs/deps/go-sdk.md`.

Known limits carried: none open.

Lessons for the next preparations: default code targets add a test file (give a smoke cap or code-only targets); a new
file needs `"intent": "generate"` in the map; a new code folder needs its layer in `decks/tools/layers.json`; size a
judge from its expected answer (≥ 28 000 for a ~20 KB answer, before any processor factor); vary every constant the code
must not hard-code across the examples; measure any timing an example promises on the real shell before the cut (P3:
dash leaves a script's children holding the pipes — kill the process group); a §2.2 gap no example pins is not enforced —
P1, P2 and P3 each left 2–3 such defects; when a gap matters (a panic, an error prefix a caller reads), add a record
example or a probe variant for it, not only a §2.2 line; an example that refers to another Function's example must
inline it (the slice never holds the other); the frozen list carries every earlier phase's code; every mutant run under a
120 s timeout; kill leftover watchers and workers of the scratch tree by process group, never by name (`pkill -x sleep`
hits every process of the user); a judge's guard wants every example literal in its test file — say so in the judge
instruction for long argv literals and fixture paths (P4: "../tests/fixtures/git/measure.md"); a judge instruction names
the arity of every multi-value function it tests (P4: `Code` returns (int, string)); a code card's allowed imports are its record description's list — name every one the
code plausibly needs (P3: strconv in github); a judge instruction names every short literal its guard will want (P5: the status "405"); an example
whose every answer is nil is tautological in the judge file — give it a contrast in the record, not only in the probe;
probe-derived judge files in the mutant tree must not share the probes' helper names (P5: the first mutant round was all
build errors); no editor or stray file in the tree while a run is in flight (P5: a `.morph-map.json.swp` failed a judge
at `== frozen`). a re-cut of ONE card after its generation's siblings were accepted must drop the
overlay that blanks those accepted files (P6: `Replace {"mcpserver/session.go":""}` broke every build once mount.go
existed — check the acceptance's overlay.json and read the WHOLE stub log, not only the guard count); a judge that
mixes two SDK calls returning different types in one example needs the instruction to name a variable per type (P6:
ListTools vs CallTool); an example's "the fake saw X" must state the whole call list when other calls precede X; a
test file may import only the standard library, the module and go.mod's direct requirements (guard.mjs since P6).

The session rewrites this section at the end of every phase.

## Machine

- User `morph` (not root). The repository `/home/morph/MorphStudio` (origin `github.com/VasyaLutiy/MorphStudio`, `main`,
  pushed only by fast-forward). Go 1.25.14 at `/usr/local/bin/go`; every Go command offline with
  `GOFLAGS=-mod=vendor GOPROXY=off`.
- The `morph` binary: MorphV2 at `/home/morph/MorphV2` (read-only for this project: never edit, commit, build in place or
  touch its flags; never touch its tmux session `morph`). A COPY of its `dist/` (with `node_modules` and `templates/`
  linked beside it) in `/tmp/morph-bin-<phase>/` runs the deck, so a card that rebuilds cannot replace the running binary.
- Processor environment: `MORPH_PROCESSOR_<P>_<KEY>` = `MRPH_PROCESSOR_<P>_<KEY>` of
  `/home/morph/MorphProject/morph-lab/.env` (P = `ds`, or `glm53` for the fallback) for KEY in TYPE, API_KEY, MODEL,
  ROUTE, CONCURRENCY, PROVIDER_ORDER, REASONING_MAX_TOKENS (only the keys that are set), exported by indirection in a
  subshell, never printed (MorphV2 recipe: `decks/p7/smoke/run.sh`).
- Anything longer than a minute runs under `nohup`/`tmux` with a log file; the session must survive an SSH drop.

## One phase, one session

Every phase gets a fresh session: the memory of a phase is in the repository (record, TASK §11, MEASURE, DECISIONS,
this file), never in the session's context. The session works exactly the phase "State at handoff" names, records it,
merges it, rewrites "State at handoff" for the next phase and pushes. Then it ends in one of two ways:

- **the next phase is queued** (named in "State at handoff" with no stop before it): post 🔀, `touch
  ~/.morph-phase-done` and stop. The watchdog (cron, 10 min) kills the session and starts a fresh one;
- **a stop** (a smoke stop, an emergency stop, a gate stop, no next phase): post the stop, `touch ~/.morph-wait-operator`
  and stop. The watchdog never nudges it; the operator restarts it with `tools/vps-start.sh` after a check.

## Shell hygiene

Never chain `cd` with a write or a delete in one command (`cd X && rm …`, `cd X && go mod vendor`): Claude Code stops
such a command for a manual approval even with permissions bypassed, and an autonomous session waits on it. Use
absolute paths or the tool's own directory flag (`git -C`, `go -C`, `npm --prefix`, `make -C`). Every brief to a
preparation agent repeats this line.

## The cycle of one phase

1. **Prepare** (an orchestrator agent with a fresh context): the spec by `docs/TASK_TEMPLATE.md`; the record and the map
   entries (a new file needs `"intent": "generate"`); fixtures; `decks/<phase>/checks.json` and the probes
   `decks/<phase>/parts/<card>.probe.*`; the cut:
   `morph plan --root . --spec contour.yaml --map morph-map.json --component <C>… --judge --checks
   decks/<phase>/checks.json --out decks/<phase>/deck.json`. When the cut holds cards outside the phase, keep the phase's
   cards with a short filter script over the deck file, committed beside the deck. For a processor that thinks before it
   writes, every card's maxTokens ×3 after the cut: `python3 decks/tools/scale_tokens.py decks/<phase>/deck.json 3`. Then
   `morph deck check --root . --deck decks/<phase>/deck.json`.
   Every acceptance is run red **per example** on stubs in a scratch worktree, with a readable line. Mutants: only on
   the record's examples and the contracts the phase changes, **at most 30 per phase and at most 20 min in total**,
   every mutant run under a 120 s timeout (a timeout counts as killed); survivors beyond the cap go to
   `docs/DECISIONS.md` as a known risk naming the file and the mutation. The data and the deck are committed on `main`.
2. **Gate without the operator.** The run starts by itself only when ALL hold: `morph plan` exit 0; `morph deck check`
   errors 0; every probe red per example with a readable line on the stubs; every acceptance chain under 250 s; the
   mutation cap kept; forecast ≤ $1 for the phase; no slice over 200 KB. Otherwise the phase stops with a report in
   `docs/MEASURE.md` (a row "stopped at gate: <reason>"), touches `~/.morph-wait-operator` and stops.
3. **Run**: from the repository root, the binary copy:
   `node /tmp/morph-bin-<phase>/dist/cli.js run --root . --deck decks/<phase>/deck.json --processor <processor> --deadline
   2400`, its stdout (the Run Document) to a file under /tmp. The run opens `morph/<runId>`, commits each accepted card
   with its trailers and archives `.morph/runs/<runId>/`. The tree is not touched while the run is in flight. A re-run of
   failed cards uses a deck file of those cards only. A failed run's archive commit is cherry-picked onto `main`.
4. **Verify** on the run branch: `git status --short` empty; the project's parse, lint, full test suite and build green;
   the written code and tests read once against §2.2 and the record; defects recorded, never fixed by hand.
5. **Record**: §11 of the TASK and the row of `docs/MEASURE.md`, one commit on the run branch.
6. **Merge and push**: `git checkout main && git merge --ff-only morph/<run-id> && git push origin main`. Fast-forward
   only; a non-ff state stops the session with a report.

## Dependencies

- Only the libraries `PLAN.md` "What is known" (the SDK) and "Zero Contour" (Dependency) lists, approved by the operator with the plan, are declared in
  `contour.yaml` (`System.dependencies`, exact versions) and named by a Component's `uses`. The session never adds,
  upgrades or removes one on its own: a card that needs another library is a stop for the operator.
- The scaffold phase installs them once with the network (Go: vendored and committed; TypeScript: `package-lock.json`
  committed, `npm ci`; Python: a venv from a pinned requirements file). Every later acceptance is offline: Go builds with
  `GOFLAGS=-mod=vendor GOPROXY=off` (chosen by `morph plan` when `vendor/modules.txt` exists), TypeScript from the
  installed `node_modules`, Python from the venv. A deck cut before the vendoring is re-cut after it.
- Each dependency's API digest `docs/deps/<name>.md` (2–5 KB, signatures + one example, from the library's own docs and
  types, values measured) is data; it rides in the slice of every card of a Component that uses it.

## Decisions the session makes alone

- **Gaps in the record** (a shape, an order, a message the record does not pin): decided in §2.2 of the phase's spec AND
  appended to `docs/DECISIONS.md`, one line each (phase, Function, decision, why). The record itself is edited only when
  an example is wrong or missing, with the change named in DECISIONS.
- **Fixtures**: by `docs/TASK_TEMPLATE.md` §2.1; recorded answers of a real service may be used after cleaning.
- **Issues labelled for a phase**: before preparing a phase, read the open issues labelled `<phase>-<component>` and
  build them into that Component's record; the phase's DECISIONS lines name the issues they answer.

## Failure

Every failed card gets one class, decided from the run log and the attempt files, written as a tag in its DECISIONS line
and in the MEASURE row:

| Class | Sign | Fix, by the session alone |
|---|---|---|
| data | the answer is whole; the acceptance is red on logic, types or the guard | ONE re-cut: the spec wording or the fixtures, never the card instruction |
| budget | the answer is cut at `max_tokens` (finish reason length, an unclosed fence) | ONE raise of that card's `max_tokens` in `morph-map.json` ×1.5–2, ceiling 32 000, then a re-run of that card |
| environment | npm, network, provider error, `exit null` timeouts on a green log | no re-cut; one plain re-run later, then stop the phase |
| code defect | the judge finds a real bug in accepted code | ONE re-cut of the code card |

- A phase with failed cards after the run: ONE fix of the failed cards by their class, merged if green.
- **Emergency stop**: a card still red after its one fix stops the whole autonomous stretch — no next phase, no merge
  beyond the green code already merged. Debts are not carried forward. Before stopping, the session pays the debt once:
  1. `morph card --root . --deck <deck> --id <card> --md` — the debt brief (`.markdown` of the JSON document: the
     targets, the instruction, the acceptance, the context slice as it is now, the last run's reason, log and answers);
  2. a stronger model (the payer agent) gets that brief and writes ONLY the card's targets;
  3. `morph accept --root . --deck <deck> --id <card> --model <payer model> --commit` — the card's own acceptance on the
     current tree; green commits only the targets with `Morph-Card`, `Morph-Model`, `Morph-Acceptance-Exit: 0`,
     `Morph-Debt: true`; a change outside the targets or a red acceptance refuses;
  4. green → `tools/tg.sh debt` 💸, a "debt" row in `docs/MEASURE.md`, the data lesson in DECISIONS, and the loop goes on;
     red → the emergency stop stands: open an issue (`gh issue create --label debt`: run ids, what was tried, the
     attempts' reasons verbatim, the code left unguarded), post "EMERGENCY STOP" with the card, the class and the issue,
     and stop for the operator.
  Cap: one paid debt per phase, ≤ 30 min.
- An environment red still red after its re-run: emergency stop with class `environment` (no debt applies; the operator
  fixes the environment).
- **Smoke stops**: after the phases `PLAN.md` names, a live smoke of the product on a tiny project, then
  `tools/tg.sh smoke` 🧪 with the numbers, then STOP and wait; the session never resumes itself. A red smoke is an
  emergency stop.

## Money

- $5 of executor per phase (PLAN), forecast ≤ $1 at the gate; Claude caps per phase $30 list price and 3 h (DECISIONS).
- $35 for the whole autonomous stretch; the running total is kept in `docs/MEASURE.md`; reaching it stops the session.

## What never happens

- Code or test files written by hand.
- `contour.yaml` in a slice. The record's history in the record.
- A paid run on a deck that did not pass the gate above.
- A push of anything but `main` fast-forward; a force-push; a rewrite of history.
- Printing a secret.

## Observability

- The session runs in the tmux session `MorphStudio`, started by `tools/vps-start.sh`; the operator attaches with
  `tools/vps-start.sh attach` (detach `Ctrl-b d`) or follows the log with `tools/vps-start.sh log`.
- `tools/tg.sh <kind> "<headline>" "<numbers>"` posts to the operator's Telegram channel (token and chat id only in
  `~/.config/morph/tg.env`, written by the operator, mode 600; a missing file is a silent no-op). Kinds: `start` 🚀 phase
  start; `gate` 🚦 gate passed (`stop` if not, with the reason); `run` 🏁 run green (written, $, minutes, burned
  variants); `fail` ❌ run with red cards; `merge` 🔀 merge and push done; `smoke` 🧪 a smoke stop; `stop` 🛑 any stop,
  the emergency stop included; `debt` 💸 a debt paid; `end` 🎉 the end of the stretch; `info` 💬 anything else. Example:
  `tools/tg.sh run "<phase> <component>: run green" "8/8 written · \$0.12 · 14 min · 0 burned"`.
- The watchdog (`tools/vps-watchdog.sh`, cron every 10 min) posts as `watchdog` 🐕; the Claude Code hooks in
  `.claude/settings.json` post `idle` 💤 on Stop and `ask` 🔔 on Notification, so an idle session is never silent.

## Order of the phases

By the record's Components, see `PLAN.md`, section "Epics and phases".
