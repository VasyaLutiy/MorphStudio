# Request to MorphV2: a re-cut that changes an API across generations must pass the gate

From: MorphStudio (morphd), the PM, 09.10.2026. Operator's order: "сделай ТЗ 3. Доработка Морфа".
For: the MorphV2 operator / orchestrator. This is a problem statement with evidence and acceptance, not a design:
the design is MorphV2's (its record, its phase).

## 1. The problem in one line

When a phase re-cuts already-built cards (`morph plan --only`, P20) and the record changes an exported Go API used by
files of cards in LATER generations, the intermediate trees do not compile: every card's acceptance builds the real
tree, the later cards' files still call the old API, and the gate stops the phase before any run.

## 2. Evidence (MorphStudio, P7b, 09.10, $0 spent)

- Repo github.com/VasyaLutiy/MorphStudio, main beb955f; DECISIONS line "09.10 · P7b · gate · STOPPED AT GATE".
- Cut: `morph plan --only` of 12 cards over Components control, supervisor, daemon, pump, cmd; generations
  [1,2,2,2,1,3,1]; exit 0, deck check 0 errors. MorphV2 binary 179c795.
- The record renamed `supervisor.Loop.Resumes` → `Restarts` and gave `Loop.Exited` a new `stderr` argument.
  - phase-loop (gen 2, writes `supervisor/loop.go`): red at `== build`: `supervisor/guard.go:20:22: l.Resumes undefined`
    — `guard.go` is the target of runtime-guard, gen 3, still the old file.
  - runtime-guard (gen 3): red at `== vet`: `supervisor/guard_examples_test.go:82:42: not enough arguments in call to
    l.Exited` (target of runtime-guard-judge, a later gen); `== full` also red on `daemon/daemon.go` and
    `daemon/pump.go:100` (targets of daemon-core, gen 4, and pump, gen 5).
  - daemon-core (gen 4): the same on `pump.go`.
- Why the existing hiding does not help: P15's overlay blanks only the SAME generation's other targets; P20 computes
  siblings over the subset, also per generation. Files of later generations of the same subset stay visible, old.
- Workaround taken for P7b: the record keeps the old Go names (compat field and a new function). It works, but it
  leaves a mismatch in the code (`Resumes` with json "restarts", two exit methods). Every future "repair what is
  built" phase with a rename will hit the same wall.

## 3. What must hold after the change

1. A `--only` subset whose record changes an exported identifier (rename, removed field, changed signature) used by
   targets of later generations of the SAME subset is cut and runs through the gate without a hand edit and without
   compat names in the record.
2. A card is still judged on a tree that compiles: no card may pass because its package was hidden whole, and no
   stage may silently skip build or vet.
3. Files that are NOT targets of the subset (callers on main outside it) are never hidden: if the record breaks
   them, the cut or the gate must say so by name before any paid run (today: the gate's build/vet line; better: at
   plan time, see 5).
4. Full cuts (no `--only`) and every existing deck stay byte-identical (the P15/P20 identity checks on go-mini and
   MorphV2's own decks).
5. Nice to have: `morph plan` reports, for an `--only` subset, which kept files outside the subset reference an
   identifier the record changes (a warning with file:line), so the PM sees the risk at the pre-flight.

## 4. Candidate directions (for MorphV2 to choose or reject; measured risks only)

- **A. Hide later-generation targets of the subset too.** Extend the overlay from "same generation" to "every target
  of the subset not yet written in this run" for the narrow stages (build, vet, probe, own) and for `full`.
  Risk: an earlier card's own code may call a function that lives in a hidden later file and exists on main — then
  the earlier card fails with "undefined" for the opposite reason. Needs a measurement on P7b's subset.
- **B. Joint acceptance of a generation group.** Cards that change one API and its callers are accepted together
  after all of them are written (one build/vet/full over the group), each card keeping its own probe. Risk: a red
  group must still name the failing card for the retry and the failure class.
- **C. Planner reorders by the API edge.** When the record changes an identifier, every card whose target uses it
  goes into the same generation as the card that defines it (the same-generation overlay then hides them from each
  other). Risk: two cards of one package in one generation (forbidden today by "two code cards of one package
  never share a generation", P15L) — needs that rule revisited.

## 5. Acceptance (how MorphStudio will check it)

- Re-cut MorphStudio P7b as designed BEFORE the compat workaround (contour.yaml at cc46ed1: `Restarts`,
  `Exited(code, stderr, …)`) with the new binary, `--only` the same 12 cards: `morph plan` exit 0, `deck check` 0,
  every generation's acceptance on stubs red only at its own probe/guard line (stubcheck), never at `== build` /
  `== vet` on another card's file; a paid run on ds green or red only by card content.
- MorphV2's own identity checks green (go-mini, P15; the P20 smoke).
- The PM re-runs it in a scratch copy, as for P6 (DECISIONS 09.10 · P6 re-run without Fable), and reports the
  numbers; main of MorphStudio is not touched.
