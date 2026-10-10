# Primer: MorphStudio

generated 2026-10-10T16:54:36.943Z · 601 files in the tree · no model call, no network
missing: docs/PLAN.md

## Tests

- 210 tests in 27 test files by the go profile (counted from text, not a run)

## Runs

- archived runs: 13 (V2 13, mrph 0), 2026-10-08 → 2026-10-10
- cards: 67 written of 117 (37 failed, 0 skipped); requests 176, answers kept 174
- cost: $3.5364 over 13 priced runs (0 unpriced)
- by format: V2 13 runs, 67/117 written, $3.5364; mrph 0 runs, 0/0 written, $0.0000
- models: deepseek/deepseek-v4.1-flash (13 runs)
- debt rows (docs/MEASURE.md): none
- running total (docs/MEASURE.md): "Running total: ≈ $4.25 of $35 executor" vs $3.5364 archived here, difference 0.7136 — the two differ by runs made outside this repository (in MEASURE, no archive here) and archived runs MEASURE's total leaves out; debt rows are in neither

## Chronology (docs/MEASURE.md)

- no rows

## File ownership (git, Morph-Card trailers)

- git carries 80 Morph commits: deepseek/deepseek-v4.1-flash 79, claude-fable-5-1 1
- 65 paths written by cards, most recent first; per path its cards, newest first:
- cmd/morphd/config_examples_test.go ← config-and-main-judge (deepseek/deepseek-v4.1-flash, run —); config-and-main-judge (deepseek/deepseek-v4.1-flash, run 20261008-211009)
- daemon/pump_examples_test.go ← pump-judge (deepseek/deepseek-v4.1-flash, run —); pump-judge (deepseek/deepseek-v4.1-flash, run 20261008-211009)
- daemon/daemon_examples_test.go ← daemon-core-judge (deepseek/deepseek-v4.1-flash, run —); daemon-core-judge.r1 (deepseek/deepseek-v4.1-flash, run 20261008-211009)
- cmd/morphd/config.go ← config-and-main (deepseek/deepseek-v4.1-flash, run —); config-and-main.r1 (deepseek/deepseek-v4.1-flash, run 20261008-211009)
- cmd/morphd/main.go ← config-and-main (deepseek/deepseek-v4.1-flash, run —); config-and-main.r1 (deepseek/deepseek-v4.1-flash, run 20261008-211009)
- daemon/pump.go ← pump (deepseek/deepseek-v4.1-flash, run —); pump (deepseek/deepseek-v4.1-flash, run 20261008-211009)
- supervisor/guard_examples_test.go ← runtime-guard-judge (deepseek/deepseek-v4.1-flash, run —); runtime-guard-judge.r1 (deepseek/deepseek-v4.1-flash, run 20261008-193337)
- daemon/daemon.go ← daemon-core (deepseek/deepseek-v4.1-flash, run —); daemon-core (deepseek/deepseek-v4.1-flash, run 20261008-211009)
- supervisor/guard.go ← runtime-guard (deepseek/deepseek-v4.1-flash, run —); runtime-guard (deepseek/deepseek-v4.1-flash, run 20261008-193337)
- supervisor/loop_examples_test.go ← phase-loop-judge (deepseek/deepseek-v4.1-flash, run —); phase-loop-judge (deepseek/deepseek-v4.1-flash, run 20261008-193337)
- supervisor/loop.go ← phase-loop (deepseek/deepseek-v4.1-flash, run —); phase-loop (deepseek/deepseek-v4.1-flash, run 20261008-193337)
- control/control_examples_test.go ← control-contract-judge (deepseek/deepseek-v4.1-flash, run —); control-contract-judge.r1 (deepseek/deepseek-v4.1-flash, run 20261008-185347)
- control/control.go ← control-contract (deepseek/deepseek-v4.1-flash, run —); control-contract (deepseek/deepseek-v4.1-flash, run 20261008-185347)
- mcpserver/pm_examples_test.go ← pm-tools-judge (claude-fable-5-1, run —)
- mcpserver/mount_examples_test.go ← mcp-mount-judge (deepseek/deepseek-v4.1-flash, run 20261008-200807)
- mcpserver/session_examples_test.go ← session-tools-judge (deepseek/deepseek-v4.1-flash, run 20261008-200807)
- mcpserver/mount.go ← mcp-mount (deepseek/deepseek-v4.1-flash, run 20261008-200807)
- mcpserver/session.go ← session-tools (deepseek/deepseek-v4.1-flash, run 20261008-200807)
- mcpserver/pm.go ← pm-tools (deepseek/deepseek-v4.1-flash, run 20261008-200807)
- api/router_examples_test.go ← router-judge.r1 (deepseek/deepseek-v4.1-flash, run 20261008-193337); router-judge (deepseek/deepseek-v4.1-flash, run 20261008-083312)
- api/router.go ← router (deepseek/deepseek-v4.1-flash, run 20261008-193337); router (deepseek/deepseek-v4.1-flash, run 20261008-083312)
- api/handlers_examples_test.go ← http-handlers-judge (deepseek/deepseek-v4.1-flash, run 20261008-193337); project-handlers-judge (deepseek/deepseek-v4.1-flash, run 20261008-083312)
- api/handlers.go ← http-handlers (deepseek/deepseek-v4.1-flash, run 20261008-193337); project-handlers (deepseek/deepseek-v4.1-flash, run 20261008-083312)
- gitrules/end_examples_test.go ← phase-end-check-judge.r1 (deepseek/deepseek-v4.1-flash, run 20261008-185347)
- registry/secrets_examples_test.go ← secret-files-judge (deepseek/deepseek-v4.1-flash, run 20261008-185347)
- registry/secrets.go ← secret-files (deepseek/deepseek-v4.1-flash, run 20261008-185347)
- registry/registry_examples_test.go ← project-registry-judge (deepseek/deepseek-v4.1-flash, run 20261008-185347)
- gitrules/start_examples_test.go ← phase-start-check-judge (deepseek/deepseek-v4.1-flash, run 20261008-185347)
- gitrules/end.go ← phase-end-check (deepseek/deepseek-v4.1-flash, run 20261008-185347)
- registry/registry.go ← project-registry (deepseek/deepseek-v4.1-flash, run 20261008-185347)
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

- 10.10 · smoke stop 2 re-run (after P7b) · by the PM on the VPS: morphd built from main 46af05c, port 7181, state /tmp/smoke2 reused (project `smoke`, loop reset to idle by hand, old sessions moved to sessions.0810), laptop over an ssh tunnel; claude $0.2459 (smokes total $0.3845 of $0.55). HELD: no…
- 10.10 · smoke 2 re-run · NEW DEFECT D4 (record, supervisor · Phase Loop StartChecked): a start check "resume" (here: local main ahead of origin by the PM's unpushed order commit) on a loop with no session yet takes a NEW id and spawns `--resume <new id>`; claude refuses ("No conversation found with…
- 10.10 · P7c · operator: a small phase for D4 (Phase Loop + its judge, 2 cards); the architect changes the record and adds the PLAN row; APPROVED IN ADVANCE without reading ("даю сразу апрув без вычитки") — the PM still runs check 1 (the row covers D4 with an example) and the pre-flight (plan exit 0…
- 10.10 · cutover to morphd (operator order 08.10, "очень жду" 10.10), by the PM: the morph user's crontab watchdog line removed (backup ~/crontab.bak-20261010; crontab now empty); tmux session MorphStudio killed (idle; markers removed); morphd installed as systemd unit `morphd.service` (User=morph, …
- 10.10 · P7c · record change by the architect (Phase Loop: "resume" with SessionID "" → fresh spawn with first line, post "<phase> resumed on a fresh session"; example 3 changed, examples 10–11 new; map phase-loop-judge 30 000 tokens; PLAN row P7c in de7a5d3). Its contour.yaml hunk landed inside the…

## Open issues (.morph/issues.json)

- not read: no .morph/issues.json (the primer makes no network call; `gh issue list --state open --json number,title,labels > .morph/issues.json` writes it)
