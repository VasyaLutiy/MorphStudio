# Primer: MorphStudio

generated 2026-10-08T17:17:16.993Z · 312 files in the tree · no model call, no network
missing: docs/PLAN.md

## Tests

- 31 tests in 4 test files by the go profile (counted from text, not a run)

## Runs

- archived runs: 2 (V2 2, mrph 0), 2026-10-08 → 2026-10-08
- cards: 22 written of 22 (0 failed, 0 skipped); requests 25, answers kept 25
- cost: $0.1348 over 2 priced runs (0 unpriced)
- by format: V2 2 runs, 22/22 written, $0.1348; mrph 0 runs, 0/0 written, $0.0000
- models: deepseek/deepseek-v4.1-flash (2 runs)
- debt rows (docs/MEASURE.md): none
- running total (docs/MEASURE.md): "Running total: $0 of $35." vs $0.1348 archived here, difference -0.1348 — the two differ by runs made outside this repository (in MEASURE, no archive here) and archived runs MEASURE's total leaves out; debt rows are in neither

## Chronology (docs/MEASURE.md)

- no rows

## File ownership (git, Morph-Card trailers)

- git carries 22 Morph commits: deepseek/deepseek-v4.1-flash 22
- 22 paths written by cards, most recent first; per path its cards, newest first:
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
  > **Next: P1 (`PLAN.md` row P1: stream, runner, queue — Parse Event, Encode Lines, Exec Runner, Phase Queue; 8 cards).**
  > After it: P2, then P3 and **smoke stop 1** (after P3), P4, P5, P6, P7 and **smoke stop 2, final** (after P7); the smokes
  > are written in `PLAN.md` "Epics and phases". Running total $0 of $35. Processor `ds` (maxTokens ×3 after the cut:
  > `scale_tokens.py … 3`); fallback `glm53`. Claude auto-memory is off in every phase session

## Last decisions (docs/DECISIONS.md)

- 08.10 · P1 · Phase Queue · example 6's running queue = example 1 after Start; example 7's done queue = the end of example 5 · the record names states, not how to reach them.
- 08.10 · P1 · Phase Queue · RECORD CHANGED: example 8 defaults {20, 2, 4} → Caps {20, 1, 4} (was {30, 3, 5} → {30, 1, 5}) · the same defaults as example 1 let hard-coded 30/3/5 pass; the probe also varies the Hours default (a zero-caps phase under {20, 2, 4}).
- 08.10 · P1 · map · all 27 judge instructions: "per example of `<file>` taken from go.mod" → "listed below (the examples of the Function)" · the examples come from the record, not go.mod (a wording defect of the architect's map).
- 08.10 · P1 · probes · Go probes are named decks/P1/parts/_<card>_probe_test.go (the name morph plan reads for Go), not <card>.probe.* · MorphV2 builder/buildAcceptances.ts probeFile.
- 08.10 · P1 · mutants · 30 on a scratch reference, 30 killed, none survive; no known risk carried.

## Open issues (.morph/issues.json)

- not read: no .morph/issues.json (the primer makes no network call; `gh issue list --state open --json number,title,labels > .morph/issues.json` writes it)
