# Primer: MorphStudio

generated 2026-10-10T20:43:47.398Z · 678 files in the tree · no model call, no network
missing: docs/PLAN.md

## Tests

- 230 tests in 30 test files by the go profile (counted from text, not a run)

## Runs

- archived runs: 15 (V2 15, mrph 0), 2026-10-08 → 2026-10-10
- cards: 79 written of 129 (37 failed, 0 skipped); requests 192, answers kept 190
- cost: $3.6696 over 15 priced runs (0 unpriced)
- by format: V2 15 runs, 79/129 written, $3.6696; mrph 0 runs, 0/0 written, $0.0000
- models: deepseek/deepseek-v4.1-flash (15 runs)
- skipped: 20261010-190913 (no report.json)
- debt rows (docs/MEASURE.md): none
- running total (docs/MEASURE.md): "Running total: ≈ $4.38 of $35 executor" vs $3.6696 archived here, difference 0.7104 — the two differ by runs made outside this repository (in MEASURE, no archive here) and archived runs MEASURE's total leaves out; debt rows are in neither

## Chronology (docs/MEASURE.md)

- no rows

## File ownership (git, Morph-Card trailers)

- git carries 92 Morph commits: deepseek/deepseek-v4.1-flash 91, claude-fable-5-1 1
- 71 paths written by cards, most recent first; per path its cards, newest first:
- api/router_examples_test.go ← router-judge.r1 (deepseek/deepseek-v4.1-flash, run 20261010-183047); router-judge.r1 (deepseek/deepseek-v4.1-flash, run 20261008-193337); router-judge (deepseek/deepseek-v4.1-flash, run 20261008-083312)
- api/router.go ← router (deepseek/deepseek-v4.1-flash, run 20261010-183047); router (deepseek/deepseek-v4.1-flash, run 20261008-193337); router (deepseek/deepseek-v4.1-flash, run 20261008-083312)
- api/handlers_examples_test.go ← http-handlers-judge.r1 (deepseek/deepseek-v4.1-flash, run 20261010-183047); http-handlers-judge (deepseek/deepseek-v4.1-flash, run 20261008-193337); project-handlers-judge (deepseek/deepseek-v4.1-flash, run 20261008-083312)
- bootstrap/adopt_examples_test.go ← project-adopt-judge (deepseek/deepseek-v4.1-flash, run 20261010-183047)
- api/handlers.go ← http-handlers (deepseek/deepseek-v4.1-flash, run 20261010-183047); http-handlers (deepseek/deepseek-v4.1-flash, run 20261008-193337); project-handlers (deepseek/deepseek-v4.1-flash, run 20261008-083312)
- control/events_examples_test.go ← event-view-judge (deepseek/deepseek-v4.1-flash, run 20261010-183047)
- control/adopt_examples_test.go ← adopt-contract-judge.r1 (deepseek/deepseek-v4.1-flash, run 20261010-183047)
- bootstrap/adopt.go ← project-adopt (deepseek/deepseek-v4.1-flash, run 20261010-183047)
- control/events.go ← event-view (deepseek/deepseek-v4.1-flash, run 20261010-183047)
- control/adopt.go ← adopt-contract (deepseek/deepseek-v4.1-flash, run 20261010-183047)
- supervisor/loop_examples_test.go ← phase-loop-judge (deepseek/deepseek-v4.1-flash, run 20261010-165924); phase-loop-judge (deepseek/deepseek-v4.1-flash, run —); phase-loop-judge (deepseek/deepseek-v4.1-flash, run 20261008-193337)
- supervisor/loop.go ← phase-loop.r1 (deepseek/deepseek-v4.1-flash, run 20261010-165924); phase-loop (deepseek/deepseek-v4.1-flash, run —); phase-loop (deepseek/deepseek-v4.1-flash, run 20261008-193337)
- cmd/morphd/config_examples_test.go ← config-and-main-judge (deepseek/deepseek-v4.1-flash, run —); config-and-main-judge (deepseek/deepseek-v4.1-flash, run 20261008-211009)
- daemon/pump_examples_test.go ← pump-judge (deepseek/deepseek-v4.1-flash, run —); pump-judge (deepseek/deepseek-v4.1-flash, run 20261008-211009)
- daemon/daemon_examples_test.go ← daemon-core-judge (deepseek/deepseek-v4.1-flash, run —); daemon-core-judge.r1 (deepseek/deepseek-v4.1-flash, run 20261008-211009)
- cmd/morphd/config.go ← config-and-main (deepseek/deepseek-v4.1-flash, run —); config-and-main.r1 (deepseek/deepseek-v4.1-flash, run 20261008-211009)
- cmd/morphd/main.go ← config-and-main (deepseek/deepseek-v4.1-flash, run —); config-and-main.r1 (deepseek/deepseek-v4.1-flash, run 20261008-211009)
- daemon/pump.go ← pump (deepseek/deepseek-v4.1-flash, run —); pump (deepseek/deepseek-v4.1-flash, run 20261008-211009)
- supervisor/guard_examples_test.go ← runtime-guard-judge (deepseek/deepseek-v4.1-flash, run —); runtime-guard-judge.r1 (deepseek/deepseek-v4.1-flash, run 20261008-193337)
- daemon/daemon.go ← daemon-core (deepseek/deepseek-v4.1-flash, run —); daemon-core (deepseek/deepseek-v4.1-flash, run 20261008-211009)
- supervisor/guard.go ← runtime-guard (deepseek/deepseek-v4.1-flash, run —); runtime-guard (deepseek/deepseek-v4.1-flash, run 20261008-193337)
- control/control_examples_test.go ← control-contract-judge (deepseek/deepseek-v4.1-flash, run —); control-contract-judge.r1 (deepseek/deepseek-v4.1-flash, run 20261008-185347)
- control/control.go ← control-contract (deepseek/deepseek-v4.1-flash, run —); control-contract (deepseek/deepseek-v4.1-flash, run 20261008-185347)
- mcpserver/pm_examples_test.go ← pm-tools-judge (claude-fable-5-1, run —)
- mcpserver/mount_examples_test.go ← mcp-mount-judge (deepseek/deepseek-v4.1-flash, run 20261008-200807)
- mcpserver/session_examples_test.go ← session-tools-judge (deepseek/deepseek-v4.1-flash, run 20261008-200807)
- mcpserver/mount.go ← mcp-mount (deepseek/deepseek-v4.1-flash, run 20261008-200807)
- mcpserver/session.go ← session-tools (deepseek/deepseek-v4.1-flash, run 20261008-200807)
- mcpserver/pm.go ← pm-tools (deepseek/deepseek-v4.1-flash, run 20261008-200807)
- gitrules/end_examples_test.go ← phase-end-check-judge.r1 (deepseek/deepseek-v4.1-flash, run 20261008-185347)
- … 41 more paths

## What is next

- next phase: none open (docs/PLAN.md)
- handoff (docs/AUTONOMY.md):
  > State at handoff
  > **Next: smoke stop 2, final** (PLAN "Smoke stop 2, final": the PM on the VPS over HTTP and MCP, ≤ $0.50 of claude,
  > of which $0.1386 is spent on smoke 1). No phase remains after it: the session posted 🧪, touched
  > `~/.morph-wait-operator` and stopped; the PM runs the smoke and the operator restarts the session after a check (a
  > green smoke → `end` 🎉 and the stretch closes; a red one → a DECISIONS line and an issue labelled for the Component).

## Last decisions (docs/DECISIONS.md)

- 10.10 · harness item 1 DONE: the operator's domain `themorph.stream` (Cloudflare, apex A → 188.245.28.11, proxy off); Caddy serves it with a Let's Encrypt certificate; Caddy sets `header_up Host {upstream_hostport}` because the MCP SDK refused a non-local Host ("Forbidden: invalid Host header"). Ch…
- 10.10 · rule (operator): straight to the orchestrator (it edits the record itself, as the old mrph order allows) when the change stays inside existing Functions, fits one phase of ≤ 12 cards and needs no new Component; the architect only when new Functions or Components appear, the work does not fi…
- 10.10 · P8a/P8b APPROVED · operator "approve 50a83fd" · Q23–Q29 as the architect proposed (credential helper only; optional Adopter; MORPHD_DONE_WAIT 30 s with the restart gap accepted; events 200 runes / limit 50; adopted caps = defaults; the PM removes the morph user's tg.env after P8b; the PM ad…
- 10.10 · rule tightened (operator): no architect for phases of this size; the PM spawns only the orchestrator agent with the morph-orchestrator skill and its standard flow (primer, scout, …), which edits the record itself; the architect stays for a new project's Start or on the operator's request. P…
- 10.10 · P8a · run 20261010-183047 on ds: 10/10 written (7 first attempt; adopt-contract-judge, http-handlers-judge, router-judge at r1), $0.1005, 13 requests; gate: plan exit 0, deck check 0/0, stubs 18/18 new examples red, 5/5 judges red at guard; a subset transaction with three NEW code files cut…

## Open issues (.morph/issues.json)

- not read: no .morph/issues.json (the primer makes no network call; `gh issue list --state open --json number,title,labels > .morph/issues.json` writes it)
