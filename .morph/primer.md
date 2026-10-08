# Primer: MorphStudio

generated 2026-10-08T19:00:06.133Z · 401 files in the tree · no model call, no network
missing: docs/PLAN.md

## Tests

- 127 tests in 17 test files by the go profile (counted from text, not a run)

## Runs

- archived runs: 5 (V2 5, mrph 0), 2026-10-08 → 2026-10-08
- cards: 48 written of 48 (0 failed, 0 skipped); requests 57, answers kept 57
- cost: $0.3122 over 5 priced runs (0 unpriced)
- by format: V2 5 runs, 48/48 written, $0.3122; mrph 0 runs, 0/0 written, $0.0000
- models: deepseek/deepseek-v4.1-flash (5 runs)
- debt rows (docs/MEASURE.md): none
- running total (docs/MEASURE.md): "Running total: $0.2269 of $35 executor" vs $0.3122 archived here, difference -0.0853 — the two differ by runs made outside this repository (in MEASURE, no archive here) and archived runs MEASURE's total leaves out; debt rows are in neither

## Chronology (docs/MEASURE.md)

- no rows

## File ownership (git, Morph-Card trailers)

- git carries 48 Morph commits: deepseek/deepseek-v4.1-flash 48
- 48 paths written by cards, most recent first; per path its cards, newest first:
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
- eventlog/log_examples_test.go ← event-log-judge (deepseek/deepseek-v4.1-flash, run 20261008-173606)
- telegram/post.go ← milestone-post.r1 (deepseek/deepseek-v4.1-flash, run 20261008-173606)
- session/machine.go ← session-machine (deepseek/deepseek-v4.1-flash, run 20261008-173606)
- eventlog/log.go ← event-log (deepseek/deepseek-v4.1-flash, run 20261008-173606)
- stream/encode_examples_test.go ← encode-lines-judge (deepseek/deepseek-v4.1-flash, run 20261008-171107)
- queue/queue_examples_test.go ← phase-queue-judge (deepseek/deepseek-v4.1-flash, run 20261008-171107)
- stream/parse_examples_test.go ← parse-event-judge (deepseek/deepseek-v4.1-flash, run 20261008-171107)
- runner/runner_examples_test.go ← exec-runner-judge (deepseek/deepseek-v4.1-flash, run 20261008-171107)
- … 18 more paths

## What is next

- next phase: none open (docs/PLAN.md)
- handoff (docs/AUTONOMY.md):
  > State at handoff
  > **Next: P5** (`PLAN.md` row P5: supervisor, api — Phase Loop, Runtime Guard, HTTP Handlers, Router; 8 cards, est.
  > $0.25), with no stop before it; then P6, P7 and **smoke stop 2, final** (after P7: post 🧪 and stop for the PM, who runs
  > smoke 2; the PLAN's ≤ $0.50 holds, one Opus turn ≈ $0.07). Operator order 08.10 (through the PM): smoke stop 1 GREEN by
  > the PM's re-run (budget $0.25, Opus, PONG, $0.0650, a rate_limit_event arrived: Limits non-nil); resume P4–P7 without

## Last decisions (docs/DECISIONS.md)

- 08.10 · P4 · mutants · 30 on a scratch reference (registry 9, secrets 5, start 8, end 5, control 3), each under timeout 120, 30/30 killed at == probe, ≈ 0.4 min in all (one first-round kill was the mutant's own build error, re-done as a real mutation); no known risk carried.
- 08.10 · P4 · run 20261008-185347 · 10/10 written on ds, $0.0444, 4.6 min, 2 retries (control-contract-judge v1 vet: Code used as one value; phase-end-check-judge v1 guard: literal "../tests/fixtures/git/measure.md" missing); no failure class, no fix needed.
- 08.10 · P4 · Project Registry · read defect, not fixed by hand: Add appends to the in-memory list before the write, so a failed write leaves the project in memory (§2.2: list unchanged) · no example pins it; known risk for P7: after a failed Add a retry says ErrExists while projects.json lacks it —…
- 08.10 · P4 · Project Registry · read defect, not fixed by hand: Open returns a read error other than not-exist unwrapped (§2.2: "registry: read <path>: …") · the record's "every error of the file system is returned as is" allows it; callers must not match on the prefix except for a decode error.
- 08.10 · P4 · Secret Files · read defect, not fixed by hand: DeleteSecret of an unknown project returns nil (§2.2: ErrNotFound) · the record says only "missing → nil"; callers check the project first.

## Open issues (.morph/issues.json)

- not read: no .morph/issues.json (the primer makes no network call; `gh issue list --state open --json number,title,labels > .morph/issues.json` writes it)
