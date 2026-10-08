# Primer: MorphStudio

generated 2026-10-08T21:38:23.838Z · 486 files in the tree · no model call, no network
missing: docs/PLAN.md

## Tests

- 201 tests in 27 test files by the go profile (counted from text, not a run)

## Runs

- archived runs: 9 (V2 9, mrph 0), 2026-10-08 → 2026-10-08
- cards: 67 written of 69 (2 failed, 0 skipped); requests 86, answers kept 86
- cost: $0.6326 over 9 priced runs (0 unpriced)
- by format: V2 9 runs, 67/69 written, $0.6326; mrph 0 runs, 0/0 written, $0.0000
- models: deepseek/deepseek-v4.1-flash (9 runs)
- debt rows (docs/MEASURE.md): none
- running total (docs/MEASURE.md): "Running total: $0.5473 of $35 executor" vs $0.6326 archived here, difference -0.0853 — the two differ by runs made outside this repository (in MEASURE, no archive here) and archived runs MEASURE's total leaves out; debt rows are in neither

## Chronology (docs/MEASURE.md)

- no rows

## File ownership (git, Morph-Card trailers)

- git carries 68 Morph commits: deepseek/deepseek-v4.1-flash 67, claude-fable-5-1 1
- 65 paths written by cards, most recent first; per path its cards, newest first:
- cmd/morphd/config_examples_test.go ← config-and-main-judge (deepseek/deepseek-v4.1-flash, run 20261008-211009)
- daemon/daemon_examples_test.go ← daemon-core-judge.r1 (deepseek/deepseek-v4.1-flash, run 20261008-211009)
- cmd/morphd/config.go ← config-and-main.r1 (deepseek/deepseek-v4.1-flash, run 20261008-211009)
- cmd/morphd/main.go ← config-and-main.r1 (deepseek/deepseek-v4.1-flash, run 20261008-211009)
- daemon/pump_examples_test.go ← pump-judge (deepseek/deepseek-v4.1-flash, run 20261008-211009)
- daemon/pump.go ← pump (deepseek/deepseek-v4.1-flash, run 20261008-211009)
- daemon/daemon.go ← daemon-core (deepseek/deepseek-v4.1-flash, run 20261008-211009)
- mcpserver/pm_examples_test.go ← pm-tools-judge (claude-fable-5-1, run —)
- mcpserver/mount_examples_test.go ← mcp-mount-judge (deepseek/deepseek-v4.1-flash, run 20261008-200807)
- mcpserver/session_examples_test.go ← session-tools-judge (deepseek/deepseek-v4.1-flash, run 20261008-200807)
- mcpserver/mount.go ← mcp-mount (deepseek/deepseek-v4.1-flash, run 20261008-200807)
- mcpserver/session.go ← session-tools (deepseek/deepseek-v4.1-flash, run 20261008-200807)
- mcpserver/pm.go ← pm-tools (deepseek/deepseek-v4.1-flash, run 20261008-200807)
- supervisor/guard_examples_test.go ← runtime-guard-judge.r1 (deepseek/deepseek-v4.1-flash, run 20261008-193337)
- api/router_examples_test.go ← router-judge.r1 (deepseek/deepseek-v4.1-flash, run 20261008-193337); router-judge (deepseek/deepseek-v4.1-flash, run 20261008-083312)
- supervisor/guard.go ← runtime-guard (deepseek/deepseek-v4.1-flash, run 20261008-193337)
- api/router.go ← router (deepseek/deepseek-v4.1-flash, run 20261008-193337); router (deepseek/deepseek-v4.1-flash, run 20261008-083312)
- supervisor/loop_examples_test.go ← phase-loop-judge (deepseek/deepseek-v4.1-flash, run 20261008-193337)
- api/handlers_examples_test.go ← http-handlers-judge (deepseek/deepseek-v4.1-flash, run 20261008-193337); project-handlers-judge (deepseek/deepseek-v4.1-flash, run 20261008-083312)
- supervisor/loop.go ← phase-loop (deepseek/deepseek-v4.1-flash, run 20261008-193337)
- api/handlers.go ← http-handlers (deepseek/deepseek-v4.1-flash, run 20261008-193337); project-handlers (deepseek/deepseek-v4.1-flash, run 20261008-083312)
- gitrules/end_examples_test.go ← phase-end-check-judge.r1 (deepseek/deepseek-v4.1-flash, run 20261008-185347)
- registry/secrets_examples_test.go ← secret-files-judge (deepseek/deepseek-v4.1-flash, run 20261008-185347)
- control/control_examples_test.go ← control-contract-judge.r1 (deepseek/deepseek-v4.1-flash, run 20261008-185347)
- registry/secrets.go ← secret-files (deepseek/deepseek-v4.1-flash, run 20261008-185347)
- registry/registry_examples_test.go ← project-registry-judge (deepseek/deepseek-v4.1-flash, run 20261008-185347)
- gitrules/start_examples_test.go ← phase-start-check-judge (deepseek/deepseek-v4.1-flash, run 20261008-185347)
- gitrules/end.go ← phase-end-check (deepseek/deepseek-v4.1-flash, run 20261008-185347)
- registry/registry.go ← project-registry (deepseek/deepseek-v4.1-flash, run 20261008-185347)
- gitrules/start.go ← phase-start-check (deepseek/deepseek-v4.1-flash, run 20261008-185347)
- … 35 more paths

## What is next

- next phase: none open (docs/PLAN.md)
- handoff (docs/AUTONOMY.md):
  > State at handoff
  > **Next: smoke stop 2, final** (PLAN "Smoke stop 2, final": the PM on the VPS over HTTP and MCP, ≤ $0.50 of claude,
  > of which $0.1386 is spent on smoke 1). No phase remains after it: the session posted 🧪, touched
  > `~/.morph-wait-operator` and stopped; the PM runs the smoke and the operator restarts the session after a check (a
  > green smoke → `end` 🎉 and the stretch closes; a red one → a DECISIONS line and an issue labelled for the Component).

## Last decisions (docs/DECISIONS.md)

- 08.10 · P7 · run 20261008-211009 · 6/6 written on ds, $0.1401, ≈ 22.7 min, 2 retries (config-and-main v1 `== listen`: no "127.0.0.1:" literal in main.go; daemon-core-judge v1 vet unused import "morphstudio/github" + gofmt); no failure class, 0 burned.
- 08.10 · P7 · Daemon Core · read defects, not fixed by hand, none pinned by an example: PlanLoad drops Begin's error (daemon.go:433, §2.2 → ErrBadInput; unreachable on a fresh queue); a spawn whose later step fails leaves the killed process as pr.proc (daemon.go:604–642), so its exit is counted agai…
- 08.10 · P7 · Daemon Core, Pump · known risk for smoke 2 (record design, P5 line 135 undecided by P7 §2.2): a failing Spawn or a claude that dies at once loops without bound (Exited → start check "fresh" → Resumes nil → spawn again; synchronously under d.mu on a Spawn error: a TG post per round, the…
- 08.10 · P7 · tests · daemon example 2's "phase 2" assertion re-checks launch(1) instead of the second Spawn for P17 (daemon_examples_test.go:525); example 3 does not check the events' Msg · known risk: those parts are pinned by the probes only.
- 08.10 · P7 · smoke 2 · PLAN's "POST order while idle → {"sent":true}" may meet no live session once S1's phase_done ends it (Order → 409 ErrNoSession) · for the PM: order before phase_done, or a two-turn S1.

## Open issues (.morph/issues.json)

- not read: no .morph/issues.json (the primer makes no network call; `gh issue list --state open --json number,title,labels > .morph/issues.json` writes it)
