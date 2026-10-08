# Primer: MorphStudio

generated 2026-10-08T17:47:29.554Z · 340 files in the tree · no model call, no network
missing: docs/PLAN.md

## Tests

- 63 tests in 8 test files by the go profile (counted from text, not a run)

## Runs

- archived runs: 3 (V2 3, mrph 0), 2026-10-08 → 2026-10-08
- cards: 30 written of 30 (0 failed, 0 skipped); requests 35, answers kept 35
- cost: $0.2070 over 3 priced runs (0 unpriced)
- by format: V2 3 runs, 30/30 written, $0.2070; mrph 0 runs, 0/0 written, $0.0000
- models: deepseek/deepseek-v4.1-flash (3 runs)
- debt rows (docs/MEASURE.md): none
- running total (docs/MEASURE.md): "Running total: $0.1217 of $35" vs $0.2070 archived here, difference -0.0853 — the two differ by runs made outside this repository (in MEASURE, no archive here) and archived runs MEASURE's total leaves out; debt rows are in neither

## Chronology (docs/MEASURE.md)

- no rows

## File ownership (git, Morph-Card trailers)

- git carries 30 Morph commits: deepseek/deepseek-v4.1-flash 30
- 30 paths written by cards, most recent first; per path its cards, newest first:
- session/orders_examples_test.go ← order-queue-judge.r1 (deepseek/deepseek-v4.1-flash, run 20261008-173606)
- session/machine_examples_test.go ← session-machine-judge (deepseek/deepseek-v4.1-flash, run 20261008-173606)
- session/orders.go ← order-queue (deepseek/deepseek-v4.1-flash, run 20261008-173606)
- telegram/post_examples_test.go ← milestone-post-judge (deepseek/deepseek-v4.1-flash, run 20261008-173606)
- eventlog/log_examples_test.go ← event-log-judge (deepseek/deepseek-v4.1-flash, run 20261008-173606)
- telegram/post.go ← milestone-post.r1 (deepseek/deepseek-v4.1-flash, run 20261008-173606)
- session/machine.go ← session-machine (deepseek/deepseek-v4.1-flash, run 20261008-173606)
- eventlog/log.go ← event-log (deepseek/deepseek-v4.1-flash, run 20261008-173606)
- stream/encode_examples_test.go ← encode-lines-judge (deepseek/deepseek-v4.1-flash, run 20261008-171107)
- queue/queue_examples_test.go ← phase-queue-judge (deepseek/deepseek-v4.1-flash, run 20261008-171107)
- stream/parse_examples_test.go ← parse-event-judge (deepseek/deepseek-v4.1-flash, run 20261008-171107)
- runner/runner_examples_test.go ← exec-runner-judge (deepseek/deepseek-v4.1-flash, run 20261008-171107)
- stream/encode.go ← encode-lines (deepseek/deepseek-v4.1-flash, run 20261008-171107)
- queue/queue.go ← phase-queue.r1 (deepseek/deepseek-v4.1-flash, run 20261008-171107)
- stream/parse.go ← parse-event (deepseek/deepseek-v4.1-flash, run 20261008-171107)
- runner/runner.go ← exec-runner (deepseek/deepseek-v4.1-flash, run 20261008-171107)
- api/router_examples_test.go ← router-judge (deepseek/deepseek-v4.1-flash, run 20261008-083312)
- api/router.go ← router (deepseek/deepseek-v4.1-flash, run 20261008-083312)
- api/handlers_examples_test.go ← project-handlers-judge (deepseek/deepseek-v4.1-flash, run 20261008-083312)
- store/file_examples_test.go ← file-store-judge (deepseek/deepseek-v4.1-flash, run 20261008-083312)
- project/project_examples_test.go ← build-project-judge.r1 (deepseek/deepseek-v4.1-flash, run 20261008-083312)
- api/handlers.go ← project-handlers (deepseek/deepseek-v4.1-flash, run 20261008-083312)
- store/mem_examples_test.go ← memory-store-judge (deepseek/deepseek-v4.1-flash, run 20261008-083312)
- store/file.go ← file-store (deepseek/deepseek-v4.1-flash, run 20261008-083312)
- project/validate_examples_test.go ← validate-input-judge.r1 (deepseek/deepseek-v4.1-flash, run 20261008-083312)
- store/mem.go ← memory-store (deepseek/deepseek-v4.1-flash, run 20261008-083312)
- project/project.go ← build-project (deepseek/deepseek-v4.1-flash, run 20261008-083312)
- auth/basic_examples_test.go ← basic-auth-judge (deepseek/deepseek-v4.1-flash, run 20261008-083312)
- project/validate.go ← validate-input (deepseek/deepseek-v4.1-flash, run 20261008-083312)
- auth/basic.go ← basic-auth (deepseek/deepseek-v4.1-flash, run 20261008-083312)

## What is next

- next phase: none open (docs/PLAN.md)
- handoff (docs/AUTONOMY.md):
  > State at handoff
  > **Next: P2 (`PLAN.md` row P2: session, eventlog, telegram — Session Machine, Order Queue, Event Log, Milestone Post;
  > 8 cards).** After it: P3 and **smoke stop 1** (after P3), P4, P5, P6, P7 and **smoke stop 2, final** (after P7); the
  > smokes are written in `PLAN.md` "Epics and phases". Running total $0.0495 of $35. Processor `ds` (maxTokens ×3 after the
  > cut: `scale_tokens.py … 3`); fallback `glm53`. Claude auto-memory is off in every phase session

## Last decisions (docs/DECISIONS.md)

- 08.10 · P2 · mutants · 30 on a scratch reference (machine 9, orders 7, log 8, post 6), 30 killed at == probe, 0.4 min, each under timeout 120; three die only on probe variants (log bad-line number, post 200-byte cut, token masked once); no known risk carried.
- 08.10 · P2 · run 20261008-173606 · 8/8 written on ds, $0.0722, 10.3 min, 2 retries (milestone-post v1 unclosed fence; order-queue-judge v1 guard: literal "  third  " missing); no failure class, no fix needed.
- 08.10 · P2 · Session Machine · read defect, not fixed by hand: Apply dereferences ev.Result on a "result" with Result nil (§2.2: zero Result) · stream.Parse never yields it; known risk for any caller building Events by hand.
- 08.10 · P2 · Milestone Post · read defect, not fixed by hand: Post panics on a nil Do (§2.2: "telegram: post failed: no Do") · no example pins it; P7 Config And Main must always pass Do.
- 08.10 · P2 · Event Log · read defect, not fixed by hand: a failed Append write says "eventlog: open <path>: …" (§2.2: "eventlog: write …") · no example pins it; callers must not match on the prefix.

## Open issues (.morph/issues.json)

- not read: no .morph/issues.json (the primer makes no network call; `gh issue list --state open --json number,title,labels > .morph/issues.json` writes it)
