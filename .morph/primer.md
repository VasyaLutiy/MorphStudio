# Primer: MorphStudio

generated 2026-10-08T20:33:39.225Z · 462 files in the tree · no model call, no network
missing: docs/PLAN.md

## Tests

- 179 tests in 24 test files by the go profile (counted from text, not a run)

## Runs

- archived runs: 8 (V2 8, mrph 0), 2026-10-08 → 2026-10-08
- cards: 61 written of 63 (2 failed, 0 skipped); requests 78, answers kept 78
- cost: $0.4925 over 8 priced runs (0 unpriced)
- by format: V2 8 runs, 61/63 written, $0.4925; mrph 0 runs, 0/0 written, $0.0000
- models: deepseek/deepseek-v4.1-flash (8 runs)
- debt rows (docs/MEASURE.md): none
- running total (docs/MEASURE.md): "Running total: $0.4072 of $35 executor" vs $0.4925 archived here, difference -0.0853 — the two differ by runs made outside this repository (in MEASURE, no archive here) and archived runs MEASURE's total leaves out; debt rows are in neither

## Chronology (docs/MEASURE.md)

- no rows

## File ownership (git, Morph-Card trailers)

- git carries 62 Morph commits: claude-fable-5-1 1, deepseek/deepseek-v4.1-flash 61
- 58 paths written by cards, most recent first; per path its cards, newest first:
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
- control/control.go ← control-contract (deepseek/deepseek-v4.1-flash, run 20261008-185347)
- bootstrap/create_examples_test.go ← project-create-judge.r1 (deepseek/deepseek-v4.1-flash, run 20261008-181657)
- claude/process_examples_test.go ← process-judge (deepseek/deepseek-v4.1-flash, run 20261008-181657)
- github/access_examples_test.go ← repo-access-judge (deepseek/deepseek-v4.1-flash, run 20261008-181657)
- bootstrap/create.go ← project-create (deepseek/deepseek-v4.1-flash, run 20261008-181657)
- claude/process.go ← process (deepseek/deepseek-v4.1-flash, run 20261008-181657)
- claude/args_examples_test.go ← launch-args-judge (deepseek/deepseek-v4.1-flash, run 20261008-181657)
- … 28 more paths

## What is next

- next phase: none open (docs/PLAN.md)
- handoff (docs/AUTONOMY.md):
  > State at handoff
  > **Next: P7** (`PLAN.md` row P7: daemon, cmd — Daemon Core, Pump, Config And Main; 6 cards, est. $0.35), with no stop
  > before it; then **smoke stop 2, final** (after P7: post 🧪 and stop for the PM, who runs smoke 2; the PLAN's ≤ $0.50
  > holds, one Opus turn ≈ $0.07). Operator order 08.10 (through the PM): resume P4–P7 without stops between them. Running
  > total $0.4072 of $35 executor; claude for the smokes $0.1386 of $0.55; claude for debts $2.4027 (P6). Processor `ds`

## Last decisions (docs/DECISIONS.md)

- 08.10 · P6 · pm-tools-judge · fix run 20261008-201843 RED ×3 (class environment of the cut + data): the re-cut single-card deck kept the generation-2 overlay blanking mcpserver/session.go, so with mount.go accepted every build said "mount.go:49: undefined: SessionServer" (the stub check counted the…
- 08.10 · P6 · run 20261008-200807 · 5/6 written on ds, $0.0580, ≈ 9.3 min; pm-tools-judge red ×3 (class data, see the RECORD CHANGED line of the fix); every other card first attempt.
- 08.10 · P6 · debt · pm-tools-judge paid by Claude Fable 5.1 xhigh: $2.4027, 2.8 min, acceptance green on its first run (7 tests), `morph accept --commit` a1d948a (Morph-Debt: true) · ds lesson (the payer's paragraph): example 6 is the only example mixing ListTools and CallTool and its "ListTools; C…
- 08.10 · P6 · PM Tools · known risk (mutant survived): projects_list rendering a nil Projects() as [] is pinned by no example (example 6's fake returns one project); a mutant `views = nil` survives the kept tests · P7 or a record example with an empty list.
- 08.10 · P6 · read · no behaviour defect vs §2.2 (fresh read-only agent); pm.go exports `type Empty struct{}` beyond the record's declared names (unpinned); Session Tools tests do not check the call-list length (example 2), the second PhaseDone call (example 3) or a single Content entry (`stText`); …

## Open issues (.morph/issues.json)

- not read: no .morph/issues.json (the primer makes no network call; `gh issue list --state open --json number,title,labels > .morph/issues.json` writes it)
