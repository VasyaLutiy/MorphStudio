# TASK P4 — registry, gitrules, control

Phase P4 of `PLAN.md` ("Epics and phases"): Components `registry` (Project Registry, Secret Files), `gitrules` (Phase
Start Check, Phase End Check), `control` (Control Contract). One code card and one judge card per Function, 10 cards.
The cards are the deck (`decks/P4/deck.json`), not this file. No stop after P4: P5 follows.

## 1. Why this

- The daemon (P7) keeps its projects, their state blobs and their secrets only through `registry` (the GitHub token of
  `PUT github-token`, the git credentials file of the first push, the session token of each spawn — record rule 6:
  files of mode 0600); the supervisor (P5) starts and ends every phase only through `gitrules` (scenarios S4, S5 of the
  brief: fresh / resume / pull / diverged at start, pushed + clean + no unmerged `morph/*` + the MEASURE row at end);
  the HTTP API (P5) and the MCP server (P6) both call only `control` (the `status` JSON of scenario S1, the error codes,
  the 17-method Control interface). 3 of 17 packages, 5 of 27 Functions, 35 record examples (7 + 6 + 8 + 7 + 7; no
  example added, two rewritten).
- The tree after P3: 9 product packages, 92 example tests. After P4: 12 packages, 5 more code files, 5 more example
  test files, 127 example tests.
- Record size of the Components (YAML dump of the group): registry 7 238 → 7 501 B, gitrules 8 118 → 8 418 B, control
  7 865 → 7 874 B; all under the 30 KB limit.
- No open issue is labelled for P4 (`P4-registry`, `P4-gitrules`, `P4-control`: `gh issue list` empty).

## 2. Contract

### 2.1. INPUT data shapes the code must build

- **One fixture file**: `tests/fixtures/git/measure.md` (unchanged since P0) — a MEASURE table, one header row
  (`| phase | builder | …`), one separator row, two data rows whose first cells are `P16 primer-go` and `P17 debt`.
  Phase End Check examples 1, 2, 3, 5 copy it whole to `<dir>/docs/MEASURE.md`; on it EndCheck finds a row for `P17` and
  `P16` and none for `P18` (also none for `P1`, `P7`, `primer-go`, `debt` — probe). Example 7 writes its own two-row
  MEASURE.md (no fixture); example 4 has no MEASURE.md.
- **Project Registry / Secret Files** work on a base `filepath.Join(t.TempDir(), "state")` (missing at Open); every file
  is under it. Example 6 of the registry writes the text `{not json` to `base/projects.json` before Open.
- **Phase Start / End Check** never run git: every command goes to a `*runner.Fake` (P1, `runner/runner.go`, in the
  slice of all four gitrules cards). Every literal lives in the record.
- **Control Contract** builds literals only (json.Marshal of views, `Percent`, `Code`, the interface by reflect).
- **Preconditions of the callees** (gitrules judges): runner · `Key(name, args...)` joins with single spaces, so the
  Script keys are `"git rev-parse --abbrev-ref HEAD"` etc. · every example. runner · an unscripted key returns
  `Fake.Default`, zero = `Result{Code: 0, Stdout: ""}` · Start example 1 and End example 1 leave `git fetch -q origin`
  unscripted (Code 0), Start example 1 leaves both merge-base keys unscripted (never called). runner · `Errors[key]`
  returns `(Result{Code: -1}, errors.New(msg))` before Script · Start example 8. runner · `Fake.Calls` records `Dir`,
  `Name`, `Args` (never env) · Start example 1 (every Dir "/p"), End example 1 (`Calls[5].Args`).
  (control judge): session · `ErrNoQuestion`, `ErrBadOption`, `ErrNotBusy` are package variables (P2) · example 6.
  github · `Access` marshals `{"checked","reachable","push","reason","checked_at"}` in that order (P3) · example 1.
- **Carried known risks and P4**: none of P1–P3's read defects is reachable from a P4 example. queue `Load` untrimmed IDs
  — registry stores only `queue.Caps`; runner's cannot-start error lacks the name — gitrules' Reason prefixes the
  command key, so the name is there (`git fetch -q origin failed: runner: exec: …`); Session Apply on a nil Result,
  Post on a nil Do, eventlog's "open" write error, Process Write/ErrExited window, the 16 MiB line — control declares
  `eventlog.Entry` and `session.OrderResult` types only and calls none of them.
- **Text an environment can change**, pinned by prefix only: `registry: read ` (the os/json text follows). Every other
  message is exact.
- **Declared dependencies**: none in this phase (the go-sdk is used by `mcpserver` only). Shared test helpers:
  `internal/testhelp` (`Equal`, `WriteFile`), in the slice of every judge.

### 2.2. OUTPUT data shapes

The exported signatures are the record's, verbatim (`contour.yaml`, groups registry, gitrules, control: "Declares in
…"). Tables: example → given → result are the record's examples (35), rendered into the judge cards by `morph plan`.

**Gaps decided here** (each also one line in `docs/DECISIONS.md`):

- Phase Start Check · RECORD CHANGED: "a Run error of any command (6 and 7 included), or a non-zero Code of a command
  marked (!), → Mode "error"" (was "a Run error or a non-zero Code of the commands marked (!)") · the old text let a Run
  error of a merge-base be read as Code ≠ 0 (a broken git reported "diverged"); probe variant (merge-base Run error →
  Mode "error", 6 calls).
- Phase End Check · RECORD CHANGED: example 1 inlines the fake (it named Phase Start Check example 1, which no gitrules
  judge slice holds) and says `git fetch -q origin` is unscripted (Code 0 by the Fake's Default) · the P2 lesson.
- Project Registry · RECORD CHANGED: example 7 writes `p[0].Name` of the List result and the Get copy, then compares
  List and Get with the stored project (was "append to p", which no aliasing List would fail) · probe also checks two
  Lists are separate slices.
- Imports · RECORD CHANGED: registry adds `slices`, `strings`; gitrules adds `errors`; control adds `context` (the
  Control interface takes `context.Context`) · a code card's allowed imports are its record description's list (P3
  lesson: strconv in github); `== imports` of `checks.json` allows exactly these lists.
- Project Registry · Open wraps a read error other than not-exist and a decode error alike, `registry: read <path>: <err>`
  (probe: `{not json` and a JSON object) · example 6 pins the prefix; Open of a base that cannot be created returns the
  os error as is.
- Project Registry · Base() returns base verbatim; Add validates only Name (Language, Dir, Caps stored as given), checks
  the name before existence (example 3); a failed write leaves the list unchanged; Add never creates `base/<name>/`
  (example 2: base holds only projects.json; probe: after SaveState only `alpha/` appears) · the record's behaviour.
- Project Registry · projects.json is `json.MarshalIndent(list, "", "  ")`, a trailing newline optional (probe compares
  with the newline trimmed); state.json's indent and every temporary file name are not pinned; no temporary file is
  left (probe lists the directories) · "with two spaces" names only the registry file.
- Project Registry · LoadState: a read error other than not-exist and an unmarshal error are returned as is; SaveState
  of a value json cannot marshal returns that error · "every error of the file system is returned as is".
- Secret Files · PutSecret checks in the order project → kind → value (examples 3 in that order); an unknown project
  creates no directory (probe) · the behaviour's order.
- Secret Files · HasSecret, ReadSecret, DeleteSecret do not validate kind (callers pass SecretKinds constants);
  HasSecret of an unknown project is false; DeleteSecret of an unknown project → ErrNotFound; a project registered with
  no directory reads ErrNoSecret (probe) · the record names only Put's kind check.
- Secret Files · the value is written verbatim, white space and newlines kept (probe: `"  t-1 \n\n"`) · example 2 keeps
  the newline.
- Phase Start Check · Branch, Local and Remote are the trimmed stdout; a dirty path is a status line that is not blank
  after TrimSpace; a merge-base Code other than 0 (1, 128, …) takes the "else" branch; when Mode is "error" the fields
  read before the failure are not pinned (callers read Mode and Reason) · the record's order and examples.
- Phase Start Check · a failed (!) command with empty stderr gives `"<key> failed: "` (nothing after the colon and
  space) · the record's formula read literally; no example.
- Phase End Check · a branch-list line is trimmed of spaces, then of a leading `"* "`, then of spaces; docs/MEASURE.md
  is `filepath.Join(dir, "docs", "MEASURE.md")`; any ReadFile error → "docs/MEASURE.md missing"; a row is a line that
  begins with "|" as read (no left trim); the header row's cell `phase` counts as a row (no phase is called that) ·
  examples 4, 7; probe variants (padded cell `|   P23   |`, `P170` matching `P170 debt`).
- Control Contract · Code maps by errors.Is in the record's order, so an error wrapping two sentinels takes the first in
  that order (probe: ErrBusy + ErrUnknownProject → 404; ErrNotWaiting + ErrBadToken → 401); an error that only has a
  sentinel's text maps to 500 · "errors.Is in this order".
- Control Contract · the views without an example (ProjectView, Milestone, TokenResult, Events, the zero Status) marshal
  by their declared tags; probe variants pin each JSON exactly (TokenResult's push_error omitted when empty) · the
  record declares the tags verbatim; P5/P6 judges compare these bodies.
- Control Contract · registry.ErrExists and control.ErrExists are different variables with the same text; Code of
  registry.ErrExists is (500, "internal") — the daemon (P7) translates registry errors into control's · out of scope here.
- Map · budgets from the expected answers: phase-start-check-judge 16 000 → 20 000 (the judge with the most examples,
  8), phase-end-check-judge 16 000 → 18 000 (the fixture copy and five Problems lists), secret-files-judge 14 000 →
  16 000 and control-contract-judge 14 000 → 16 000 (TASK_TEMPLATE floor), control-contract 10 000 → 12 000 (≈ 6 KB of
  declarations).
- Map · two test files per package in different generations, neither in the other's slice: helper prefixes `rg` /
  `sf` (registry), `sc` / `ec` (gitrules), `cc` (control); the gitrules judges write every command key and Problem as
  its full literal (the guard wants them); the control judge writes the JSON as raw strings and counts methods with
  reflect.

### 2.3. Names

- Code (one file per card, code only — no smoke test target): `registry/registry.go`, `registry/secrets.go`,
  `gitrules/start.go`, `gitrules/end.go`, `control/control.go`; packages `registry`, `gitrules`, `control` of module
  `morphstudio` (layers already in `decks/tools/layers.json`: registry → queue; gitrules → runner; control → queue,
  github, eventlog, session, stream).
- Judge files, one test per record example, in example order: `registry/registry_examples_test.go`
  (`TestProjectRegistryExample1..7`), `registry/secrets_examples_test.go` (`TestSecretFilesExample1..6`),
  `gitrules/start_examples_test.go` (`TestPhaseStartCheckExample1..8`), `gitrules/end_examples_test.go`
  (`TestPhaseEndCheckExample1..7`), `control/control_examples_test.go` (`TestControlContractExample1..7`).
- Probes (data, inlined into the code cards' acceptances, removed after the step): `decks/P4/parts/_<card>_probe_test.go`,
  tests `TestProbe<Function>Example<N>`; helpers prefixed `pPR`, `pSF`, `pSC`, `pEC`, `pCC`.

### 2.4. What must not break

Frozen byte for byte (every acceptance's `== frozen`): `contour.yaml`, `morph-map.json`, `docs`, `decks`,
`tests/fixtures`, `go.mod`, `go.sum`, `internal`, `vendor`, and the code of P1–P3: `stream`, `runner`, `queue`,
`session`, `eventlog`, `telegram`, `claude`, `github`, `bootstrap`. The suite before P4: 92 example tests; after: 127.
No ripple: every target is a new file in a new package.

## 3. Acceptance

Built by `morph plan --checks decks/P4/checks.json` (Go profile, vendor mode), every stage printing `== <stage>`.

- Code card: `== build` (overlay hides the generation's sibling targets) → `== vet` → `== gofmt` → `== guard` (layers)
  → `== probe` → `== imports` (allow-lists = the record's lists: registry `encoding/json, errors, fmt, os,
  path/filepath, regexp, slices, sort, strings, sync, time, morphstudio/queue`; gitrules `context, errors, fmt, os,
  path/filepath, strings, morphstudio/runner`; control `context, encoding/json, errors, net/http, time,
  morphstudio/queue, morphstudio/github, morphstudio/eventlog, morphstudio/session` — no os/exec, no syscall anywhere)
  → `== clock` (no `time.Now`, no `time.Since`, all five) → `== net` (control: no `http.DefaultClient`, `http.Get`,
  `http.Post`, `http.PostForm`, `http.Head`) → `== full` → `== frozen` → no untracked file.
- Judge card: `== vet` → `== gofmt` → `== guard <file>` (min = examples, max = min + 6: registry 7..13, secrets 6..12,
  start 8..14, end 7..13, control 7..13; the literals: every `Test<Function>ExampleN` plus distinct values of the
  examples, in `checks.json`) → `== own` → `== full` → `== frozen` → no untracked file.
- Measured on a scratch worktree (`/tmp/p4-scratch`, data committed there per generation, removed after): every code
  acceptance red at `== probe` on zero-value stubs (untagged structs, an empty Control interface, sentinels
  `errors.New("stub")`), one `--- FAIL: TestProbe…ExampleN` per example, 35 of 35 (registry 7, secrets 6, start 8,
  end 7, control 7), each with an `example N …` line, no panic (a nil `*Registry` from the stub Open is a Fatalf line);
  judges red at `== guard` on a one-test stub (the count line plus every missing literal: 23, 20, 27, 20, 28); all 10
  green on a reference written in the scratch tree only (judges green with test files derived from the probes); every
  chain timed with a cold Go cache: max 35.4 s (control-contract), red chains ≤ 1.1 s (the first judge red on a cold
  cache 26 s).

## 4. Constraints

- Module `morphstudio`, Go 1.25, `GOFLAGS=-mod=vendor GOPROXY=off`.
- Stubs and helpers only from `morphstudio/internal/testhelp`; a test writes only under `t.TempDir()`; no test runs git
  (a `*runner.Fake`), opens a listener or dials.
- `registry`, `gitrules`, `control` read no clock (CreatedAt and At come from the caller); `gitrules` runs nothing
  except through `runner.Runner`; `control` is declarations and two pure functions.
- A judge writes only its test file; a code card writes only its code file (no test target, so no smoke cap).
- Generations (measured by the cut): [control-contract, phase-start-check, project-registry], [control-contract-judge,
  phase-end-check, phase-start-check-judge, project-registry-judge, secret-files], [phase-end-check-judge,
  secret-files-judge]. `gitrules/end.go` and `gitrules/start_examples_test.go`, `registry/secrets.go` and
  `registry/registry_examples_test.go` are written in the same generation in the same package: the overlay hides each
  from the other's acceptance; neither is in the other's slice.

## 7. Out of scope

- Who calls these: the supervisor's phase loop (P5: StartCheck/EndCheck and the Stop views), the HTTP handlers and the
  router (P5: `Code`, the views as bodies), the MCP tools (P6), the daemon that implements `Control`, translates
  registry errors and writes the secrets at the token PUT and each spawn (P7).
- A real git, a real GitHub; morphd never commits, merges or pushes (PLAN).
- Encrypting secrets at rest, rotating them, a registry delete or rename (not in v1).
- Any record example beyond the 35 (the probes' extra variants follow the behaviour text; they are not examples).

## 8. How to run

```
node /tmp/morph-bin-P4/dist/cli.js plan --root . --spec contour.yaml --map morph-map.json \
  --component registry --component gitrules --component control --judge \
  --checks decks/P4/checks.json --out decks/P4/deck.json
python3 decks/tools/scale_tokens.py decks/P4/deck.json 3
node /tmp/morph-bin-P4/dist/cli.js deck check --root . --deck decks/P4/deck.json        # errors 0
node /tmp/morph-bin-P4/dist/cli.js run --root . --deck decks/P4/deck.json --processor ds --deadline 2400
```

The cut holds exactly the 10 P4 cards: no filter script is needed. `/tmp/morph-bin-P4` is the copy of the MorphV2
binary made for this phase.

## 9. Pre-registration

- 10 cards, 3 generations [3, 5, 2]; processor `ds`, maxTokens ×3: control-contract 36 000, phase-start-check 30 000,
  project-registry 36 000, phase-end-check 30 000, secret-files 24 000, control-contract-judge 48 000,
  phase-start-check-judge 60 000, project-registry-judge 54 000, phase-end-check-judge 54 000, secret-files-judge
  48 000 (sum 420 000).
- Bill: forecast ≈ $0.08 (P1–P3 actuals $0.0495, $0.0722, $0.0608 for 8 cards each, ≈ $0.0076 a card, × 10 cards;
  P4's slices are small — the largest 18 899 B at the cut, ≈ 24 KB with the written control.go). Ceiling if every card
  used its whole budget once: 420 000 output × $1.20 / M = $0.50 + ≈ 80 000 input × $0.30 / M = $0.02 → $0.53 (P2's
  conservative prices). Cap $5; gate ≤ $1.
- Expected regenerations: secret-files (an author writing the file in place with os.WriteFile keeps the 0644 of example
  4: "example 4 mode after rewrite"); project-registry (a List returning its own slice: "example 7 List after writes to
  returned values"); phase-start-check (the merge-base Run error read as a Code: "example 4 variant (Run error of
  command 6)"); control-contract (the 17-method interface by reflect: "example 6 Control methods"); the gitrules judges
  (guard literals of long keys).
- Falsifiable: 10/10 written within one fix; 127 example tests green on `go test ./...`; no card over 1 regeneration.

## 10. What to record

The `docs/MEASURE.md` row and §11: attempts per card, first red per burned variant, neighbour-red, judge defects,
guard rejections, lines by hand, max slice bytes, minutes, $.

## 11. Actual

### Preparation (before the gate)

- Orchestrator: Opus 5.5 agent (fresh context); its tokens, tool calls and minutes are measured by the session from the
  agent's run.
- `morph plan` exit 0; `morph deck check` errors 0, warnings 0, hazards 0; generations [3, 5, 2]; the cut holds exactly
  the 10 phase cards.
- Largest slice at the cut: control-contract-judge, 18 899 B of existing files (≈ 24 KB with the written control.go);
  no slice near 200 KB.
- Probes red per example on stubs: 35 / 35 (registry 7, secrets 6, start 8, end 7, control 7); judges red at
  `== guard` on a one-test stub (every missing literal named).
- Acceptance chains (cold Go cache, one per card): max 35.4 s (control-contract green), all 10 green on the scratch
  reference; red chains ≤ 1.1 s on a warm cache.
- Mutants: 30 on the reference (registry 9, secrets 5, start 8, end 5, control 3), each under `timeout 120`, run against
  the card's probe: 30 / 30 killed (one first-round kill was a build error of the mutant itself, re-done as a real
  mutation and killed at `== probe`); 0.14 min for the final round, ≈ 0.4 min in all. Survivors: none.
- Record changes: Phase Start Check behaviour (Run error of any command); Phase End Check example 1 (inlined fake);
  Project Registry example 7 (writes to returned values); the import lists of registry, gitrules, control. Map: 5
  budgets, 5 judge instructions.
- 0 lines of product code by hand (the reference and the stubs lived only in a scratch worktree, removed).
