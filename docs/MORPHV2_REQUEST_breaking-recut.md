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
- No workaround taken: the operator chose (09.10) to fix MorphV2 first and run P7b as approved (cc46ed1). The
  alternative, compat names in the record (`Resumes` with json "restarts", two exit methods), was dropped: it leaves
  a mismatch in the code, and every future "repair what is built" phase with a rename would hit the same wall.

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
5. **Required (operator 09.10, amended):** `morph deck check` on an `--only` deck builds the tree each generation's
   acceptance would see — earlier generations' targets as stubs, the generation's own targets as stubs — and exits
   with an error naming `file:line` when a file that is not that generation's own target fails `build` or `vet`.
   The check lives in the tool, not in a skill's text: the PM's pre-flight runs `deck check`, so the risk surfaces
   before the approval with no rule to remember.

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
- Requirement 3.5: `morph deck check` exits non-zero naming `supervisor/guard.go:20` on a deck whose generations still
  break each other (the cc46ed1 P7b subset cut the old way, or a test fixture of that shape), and exits 0 on the
  deck the fixed cut produces.
- MorphV2's own identity checks green (go-mini, P15; the P20 smoke).
- The PM re-runs it in a scratch copy, as for P6 (DECISIONS 09.10 · P6 re-run without Fable), and reports the
  numbers; main of MorphStudio is not touched.

## 6. Amendment after P21a (09.10, operator: fix MorphV2, no workaround)

P21a (MorphV2 01498f5, direction A) failed the live smoke on go-p7b (2/8): a judge retry in gen 2 ran `== full` after
phase-loop had written the new `loop.go`, next to the old `guard.go` (gen 3) → `guard.go:20: l.Resumes undefined`
again. `guard.go` cannot be hidden: `mcp/session.go`, outside the subset, needs the whole `supervisor` package.
Conclusion of both sides: an intermediate tree of a rename inside one package cannot be made consistent → atomicity
(direction B). The operator orders P21b = requirement 3.5 + B, with these points from MorphStudio's code:

1. **The atomic unit is the API closure, not one package.** `Exited`'s new argument breaks callers in another package
   of the subset: `daemon/daemon.go:109, 609–641` and `daemon/pump.go:100` (targets of daemon-core and pump, later
   generations). `== full` builds the whole module, so a per-package group stays red until `daemon` is rewritten too.
   The group is every subset card whose target defines or uses a changed identifier (for P7b: practically the whole
   subset); the simple rule "the whole `--only` subset is one group" is acceptable.
2. **Probes too.** A card's probe compiles its whole package, siblings' old files included (`supervisor`: new
   `loop.go` + old `guard.go`). So in the group every acceptance stage of every card — build, vet, probe, own, full —
   runs on the tree with ALL group cards written (their latest versions), not on an intermediate tree.
3. **Retries.** A red card in the group is retried against that same full tree; after the retry, the group's shared
   stages run again for every card (a retry can break a sibling). The case that broke P21a — "a retry after a sibling
   has already written" — is in the gate's demo and in the smoke.
4. **Blame.** A red shared stage names the card whose target holds the failing `file:line`; a file outside the
   subset is named as "outside the subset" (the record breaks main → the gate stops before a paid run, req. 3.3).
5. **Rejected:** "keep later files hidden at run time" (the run decides on the fly what to hide; `mcp/session.go`
   still breaks).

Acceptance stays §5, plus: the live smoke on go-p7b green (8/8 or red only by card content), including a forced judge
retry after its gen-2 sibling has written.
