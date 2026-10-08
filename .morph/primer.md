# Primer: MorphStudio

generated 2026-10-08T19:47:54.966Z · 429 files in the tree · no model call, no network
missing: docs/PLAN.md

## Tests

- 159 tests in 21 test files by the go profile (counted from text, not a run)

## Runs

- archived runs: 6 (V2 6, mrph 0), 2026-10-08 → 2026-10-08
- cards: 56 written of 56 (0 failed, 0 skipped); requests 67, answers kept 67
- cost: $0.3939 over 6 priced runs (0 unpriced)
- by format: V2 6 runs, 56/56 written, $0.3939; mrph 0 runs, 0/0 written, $0.0000
- models: deepseek/deepseek-v4.1-flash (6 runs)
- debt rows (docs/MEASURE.md): none
- running total (docs/MEASURE.md): "Running total: $0.3086 of $35 executor" vs $0.3939 archived here, difference -0.0853 — the two differ by runs made outside this repository (in MEASURE, no archive here) and archived runs MEASURE's total leaves out; debt rows are in neither

## Chronology (docs/MEASURE.md)

- no rows

## File ownership (git, Morph-Card trailers)

- git carries 56 Morph commits: deepseek/deepseek-v4.1-flash 56
- 52 paths written by cards, most recent first; per path its cards, newest first:
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
- github/access.go ← repo-access.r1 (deepseek/deepseek-v4.1-flash, run 20261008-181657)
- claude/args.go ← launch-args (deepseek/deepseek-v4.1-flash, run 20261008-181657)
- session/orders_examples_test.go ← order-queue-judge.r1 (deepseek/deepseek-v4.1-flash, run 20261008-173606)
- session/machine_examples_test.go ← session-machine-judge (deepseek/deepseek-v4.1-flash, run 20261008-173606)
- session/orders.go ← order-queue (deepseek/deepseek-v4.1-flash, run 20261008-173606)
- telegram/post_examples_test.go ← milestone-post-judge (deepseek/deepseek-v4.1-flash, run 20261008-173606)
- … 22 more paths

## What is next

- next phase: none open (docs/PLAN.md)
- handoff (docs/AUTONOMY.md):
  > State at handoff
  > **Next: P6** (`PLAN.md` row P6: mcpserver — PM Tools, Session Tools, MCP Mount; 6 cards, est. $0.20), with no stop
  > before it; then P7 and **smoke stop 2, final** (after P7: post 🧪 and stop for the PM, who runs smoke 2; the PLAN's ≤
  > $0.50 holds, one Opus turn ≈ $0.07). Operator order 08.10 (through the PM): resume P4–P7 without stops between them.
  > Running total $0.3086 of $35 executor; claude for the smokes $0.1386 of $0.55. Processor `ds` (maxTokens ×3 after the

## Last decisions (docs/DECISIONS.md)

- 08.10 · P5 · run 20261008-193337 · 8/8 written on ds, $0.0817, 10.2 min, 2 retries (router-judge v1 guard: literal "405" missing; runtime-guard-judge v1 red only at == frozen on a stray .morph-map.json.swp left beside the map during the run, not the card's — gone after); no failure class, no fix ne…
- 08.10 · P5 · Runtime Guard · read defect, not fixed by hand: Exited overwrites Resumes with the last hour's resumes (guard.go countResumes), so the list is pruned (§2.2: "appended, never pruned") · no example pins it; count and cap correct; known risk for P7: the saved Loop JSON loses older resumes.
- 08.10 · P5 · Phase Loop · read defect, not fixed by hand: StartChecked fresh appends " · " + Reason only for a Reason beginning "pulled to " (§2.2: any non-empty Reason) · no example pins it; gitrules returns only "" or "pulled to …" today; latent if gitrules gains a fresh reason.
- 08.10 · P5 · Phase Loop · known risk for P7 (code follows the record): Begin/Continue return the queue's own error (e.g. nothing to continue), which control.Code maps to 500 "internal" · P7's daemon (or a record change) should map it to a 409 sentinel; EndChecked with an empty Outcome begins phase …
- 08.10 · P5 · Runtime Guard · example 8 (all answers nil) is tautological in the kept judge file; only the probe's contrast variant made it red at the gate · known risk: a later change that answers there is not caught by the example tests.

## Open issues (.morph/issues.json)

- not read: no .morph/issues.json (the primer makes no network call; `gh issue list --state open --json number,title,labels > .morph/issues.json` writes it)
