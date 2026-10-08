# Task template (`docs/TASK_<phase>_<component>.md`)

> The spec of one phase. The card breakdown is never written here: that is the deck, cut by `morph plan` from the record
> and `morph-map.json`. The spec is the contract a reader understands months later.

## 1. Why this

The motive with numbers (what reddens, how often, measured where; which issue). No number, no section. For a Component
that grows: its record size before and after (bytes of its block in `contour.yaml`, limit 30 KB).

## 2. Contract

### 2.1. INPUT data shapes the code must build

For every shape the code or a test must **construct or parse**: what it looks like and which module or file defines it;
that module is in the slice of every card that builds it.

**Fixtures are described by type, not by name:**

- Every fixture is described by the **type the Function takes** — one object or an array, text or parsed, one file or a
  tree — and by **what each Function returns on it, counted** ("three hazards: write-write a,b; read-write c,a; ...").
- A file is called an example's input **only if it is exactly that input**. Otherwise say which element of which file
  holds the example's `given` literal, or that the example has no file and its literal lives in the record.
- Every count and every illustration is **checked against the code or a spike before the gate**.
- Text an environment can change (a runtime's error message, a locale, a path separator, a file mode under a umask) is
  pinned by prefix or by rule, never verbatim.

**Declared dependencies:**

- A Function whose code imports a library names it through its Component's `uses`; the library is declared in the
  record with an exact version and an API digest (`PLAN.md` "What is known" (the SDK) and "Zero Contour" (Dependency)). §2.1 lists the calls the code makes
  into it, as in the digest, and every example's literal is measured on the declared version.
- The guard allows exactly the declared names in the code files of the Components that use them; tests keep the test
  framework and the shared helpers only. Acceptances run offline (Go: `-mod=vendor`, `GOPROXY=off`).

**A judge's setup across Components:**

- **Preconditions of the callees.** When the judged Function calls Functions of other Components, list every fact a
  test's setup depends on, as "<Component> · <fact> · what happens without it"; each example that needs it cites it.
- **Distinct markers.** Strings a test or an acceptance looks for are distinct and none is a substring of another.
- **A harness skeleton, not a prohibition list.** A judge whose setup builds more than a literal (a tree, a repository,
  a transport, a clock) gets a skeleton of ≤ 10 lines here, built only from the shared helpers module.
- **Output budget from the answer.** Each card's `max_tokens` in `morph-map.json` is sized from the expected file
  (code ≈ 12 000, judges 16 000–24 000, a judge answering ~20 KB ≥ 28 000; the judge with the most examples ≥ 20 000),
  before any processor factor.

### 2.2. OUTPUT data shapes

The exact shape of every result: the exported signatures as code, every union as literals, every message text pinned or
pinned by prefix, every ordering rule, every default. One table per Function: example → given → result. Where a shape
already exists in the tree, reference it.

**Gaps decided here**: every place the record left open, one bullet "decision · why", and the same as one line in
`docs/DECISIONS.md`.

### 2.3. Names

Modules (the language profile's code target), exported names, the test files: the author's smoke test (≤ 5 checks) or
none when a probe covers the card; the judge's examples file, one test per record example, in example order, named
"<Function> example <N>: <what>" (Go: `Test<Function>Example<N>`; Python: `test_<function>_example_<n>`).

### 2.4. What must not break

The files no card of this phase writes, byte for byte; the test count that stays green; the ripple (tests that pin the
old behaviour, measured by a spike) excluded deck-wide until their judge patches them.

## 3. Acceptance

Built by `morph plan --checks decks/<phase>/checks.json`, narrow to broad, every stage printing `== <stage>`: parse
(per-card config excluding the generation's other targets) → lint on the card's files → `decks/tools/guard.mjs` (layers,
the test-file rules, the example literals of a judge) → the probe (`decks/<phase>/parts/<card>.probe.*`, red per example
with a readable line) → the card's own tests → the full suite → frozen files → no untracked file left. Judge bounds are
derived: min = the examples, max = min + 6. The whole chain is timed on a dry tree: under 250 s.

## 4. Constraints

The project's (module resolution and imports, no `any` or its language's equivalent, stubs from the shared helpers
module only, a test writes only under a temporary directory, pure layers read no clock, randomness or environment, a
card's smoke test ≤ 5 checks, a judge writes only its test file); a file a card writes is in no sibling's slice in the
same generation. Plus the phase's own.

## 7. Out of scope

An explicit list, or the deck sprawls; the later phase that owns each item.

## 8. How to run

```
morph plan --root . --spec contour.yaml --map morph-map.json --component <C> --judge \
  --checks decks/<phase>/checks.json --out decks/<phase>/deck.json
python3 decks/tools/scale_tokens.py decks/<phase>/deck.json 3        # a processor that thinks before it writes
morph deck check --root . --deck decks/<phase>/deck.json             # errors 0
node <binary copy>/dist/cli.js run --root . --deck decks/<phase>/deck.json --processor <processor> --deadline 2400
```

## 9. Pre-registration

Cards and generations, the cards you expect to regenerate and their first red, the bill (forecast and cap),
falsifiable claims.

## 10. What to record

The row of `docs/MEASURE.md` and §11: attempts, first red per burned variant, neighbour-red, judge defects, guard
rejections, lines by hand, max slice bytes, minutes, $.

## 11. Actual

Filled after the gate (preparation numbers) and after the run; a fix gets its own sub-section with every spec change
before/after.

## Spec author checklist before the gate

- Every fixture named by the type the Function takes and by its counted result (§2.1).
- Every illustration checked against the table it illustrates (§2.2).
- Every message pinned exactly or by prefix; nothing environment-dependent verbatim.
- Every judge that calls other Components: their preconditions listed and cited, markers distinct, a harness skeleton
  when the setup is more than a literal, `max_tokens` sized from the expected answer.
- Every slice path exists on disk at plan time; `contour.yaml` is in no slice.
- Every library the code imports is declared (exact version, `uses`, a 2–5 KB digest) and installed by the scaffold
  phase; nothing is fetched by an acceptance.
- `morph plan` exit 0; `morph deck check` errors 0; every acceptance red per example on stubs in a scratch worktree;
  mutants within the cap killed or recorded; chain under 250 s.
