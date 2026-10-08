# PLAN.md template — <project name>

> The shape of MorphV2's own `docs/PLAN.md` (06.10.2026), the plan Morph built itself from. Every section is
> filled; "—" is allowed only where the section says so. Written by `morph-architect` from `docs/START.md`;
> approved by the operator by commit hash; after approval only the phase data changes without a new approval.

## Context

- The goal in one sentence and who uses it (from the brief, the operator's words).
- The operator's answers that shape the design, as a list (stack, size, money cap, must / must not).
- What the operator gets at the end: the framework, named concretely ("a CLI with 4 commands and 60 tests",
  "an HTTP API with 6 routes and a file store"), and the sentence: **not production**.

## Brief → design (traceability)

| scenario of the brief | phase | Component · Function | examples carrying its values |
|---|---|---|---|

Every rule of the brief: the example or Guardrail that pins it. Every not-build item: listed in "Out of scope".

## What is known

- The language profile and its acceptance (compiler/linter/tests, the guard, the helpers module).
- Comparable projects Morph built and their numbers (cards, first-attempt rate, $), if any.
- Risks of this design for autonomous building (a big Component, a hard-to-test Function, external formats) and
  how the cut handles each.

## Epics and phases

| phase | Component(s) | Functions | cards | gens | depends on | $ est. | stop after |
|---|---|---|---|---|---|---|---|
| P0 | scaffold (hand data) | — | 0 | — | — | 0 | — |

- The smoke stops: after which phase, what is run from outside, the expected numbers.
- The total: phases, cards, $ estimate, the cap.

### Measures per phase

The `docs/MEASURE.md` row every phase fills: written/planned, first attempt, retries, fixes, $, run minutes,
preparation minutes, mutants (count / killed / minutes).

### Decisions on open questions

| Q | question | decision | by (operator / proposed) |
|---|---|---|---|

## Out of scope

Each item of the brief's not-build list and each tempting extra, with the later phase or product that owns it.

## Scaffold (P0, hand data on the VPS side)

Every file, one line each: the module file, lint and test config, the helpers module and what it exports, the
guard's layer table (which Component may import what), `.gitignore`, the fixtures directory layout.

## Zero Contour

The record is `contour.yaml` (committed with this plan). Here: its Components with size in KB, Function count and
example count each, the Requirements and Guardrails in one line each.

## morph-map.json

Committed with this plan. Here: the cards of the first phase as a table (card · target · slice · depends on ·
max_tokens); later phases' entries are complete in the file.

## Verification

How the operator will know the framework is done: the final smoke (commands, inputs, expected outputs), the test
count, the end-to-end check from outside.

## Record rules for this project

1. The record holds only the current contract: no history.
2. Large literals live in fixtures; one Component ≤ 30 KB, else two.
3. <project-specific rules: naming, error format, layering>
