# Primer: MorphStudio

generated 2026-10-08T18:25:52.619Z · 368 files in the tree · no model call, no network
missing: docs/PLAN.md

## Tests

- 92 tests in 12 test files by the go profile (counted from text, not a run)

## Runs

- archived runs: 4 (V2 4, mrph 0), 2026-10-08 → 2026-10-08
- cards: 38 written of 38 (0 failed, 0 skipped); requests 45, answers kept 45
- cost: $0.2678 over 4 priced runs (0 unpriced)
- by format: V2 4 runs, 38/38 written, $0.2678; mrph 0 runs, 0/0 written, $0.0000
- models: deepseek/deepseek-v4.1-flash (4 runs)
- debt rows (docs/MEASURE.md): none
- running total (docs/MEASURE.md): "Running total: $0.1825 of $35" vs $0.2678 archived here, difference -0.0853 — the two differ by runs made outside this repository (in MEASURE, no archive here) and archived runs MEASURE's total leaves out; debt rows are in neither

## Chronology (docs/MEASURE.md)

- no rows

## File ownership (git, Morph-Card trailers)

- git carries 38 Morph commits: deepseek/deepseek-v4.1-flash 38
- 38 paths written by cards, most recent first; per path its cards, newest first:
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
- … 8 more paths

## What is next

- next phase: none open (docs/PLAN.md)
- handoff (docs/AUTONOMY.md):
  > State at handoff
  > **Next: P3 (`PLAN.md` row P3: claude, github, bootstrap — Launch Args, Process, Repo Access, Project Create; 8 cards),
  > then smoke stop 1 (after P3).** After the smoke: P4, P5, P6, P7 and **smoke stop 2, final** (after P7); the smokes are
  > written in `PLAN.md` "Epics and phases". Running total $0.1217 of $35. Processor `ds` (maxTokens ×3 after the cut:
  > `scale_tokens.py … 3`); fallback `glm53`. Claude auto-memory is off in every phase session

## Last decisions (docs/DECISIONS.md)

- 08.10 · P3 · checks · frozen adds session, eventlog, telegram (P2 code); claude's allow-list adds bytes, strings, time beyond the record's import list (== clock still rejects time.Now/time.Since); bootstrap has no os/exec; github gets == net · layers.json stdlib_rules.
- 08.10 · P3 · mutants · 30 on a scratch reference (args 7, process 11, access 7, create 5), each under timeout 120; final round 30/30 killed at == probe, 0.41 min; no known risk carried.
- 08.10 · P3 · run 20261008-181657 · 8/8 written on ds, $0.0608, 7.5 min, 2 retries (repo-access v1 imports strconv, outside its allowed list; project-create-judge v1 guard: three example literals missing); no failure class, no fix needed.
- 08.10 · P3 · Process · read defect, not fixed by hand: `exited` is set after Lines() closes (after Wait), so a Write between the close of Lines() and the ExitStatus returns a pipe error, not ErrExited (§2.2: set before the close) · example 2 writes after `<-Exit()`; known risk: callers treat any Wr…
- 08.10 · P3 · Process · read defect, not fixed by hand: a stdout line over 16 MiB stops the reader but Wait waits for it, so a child still writing blocks and Exit never arrives until Kill/cancel (§2.2: "Exit follows") · not tested; the runtime guard's Kill bounds it.

## Open issues (.morph/issues.json)

- not read: no .morph/issues.json (the primer makes no network call; `gh issue list --state open --json number,title,labels > .morph/issues.json` writes it)
