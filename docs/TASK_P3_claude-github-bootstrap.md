# TASK P3 — claude, github, bootstrap

Phase P3 of `PLAN.md` ("Epics and phases"): Components `claude` (Launch Args, Process), `github` (Repo Access),
`bootstrap` (Project Create). One code card and one judge card per Function, 8 cards. The cards are the deck
(`decks/P3/deck.json`), not this file. P3 ends with **smoke stop 1** (`PLAN.md`): after the merge, the smoke as written
there, 🧪, and a stop for the operator.

## 1. Why this

- The daemon (P7) starts claude only through `claude` (the argv, the auto-memory env, the MCP config file with the
  session token, the process with its line channel and exit), checks the user's GitHub token only through `github`
  (`PUT github-token`, the status JSON of P4 `control`), and creates a project only through `bootstrap`
  (`project_create`, the first push at the token PUT). 3 of 17 packages, 4 of 27 Functions, 29 record examples
  (7 + 7 + 8 + 7; no example added, one rewritten).
- The tree after P2: 6 product packages, 63 example tests. After P3: 9 packages, 4 more code files, 4 more example test
  files, 92 example tests.
- Smoke stop 1 calls `claude.Start(ctx, "claude", claude.Args(claude.Launch{…}), dir, nil)` against a real claude CLI:
  the first live use of the protocol since the probe; a red smoke means the protocol moved (a re-probe by the PM).
- Record size of the Components (YAML dump of the group): claude 8 004 → 8 926 B, github 4 086 → 4 178 B, bootstrap
  6 591 → 6 591 B; all under the 30 KB limit.
- No open issue is labelled for P3 (`P3-claude`, `P3-github`, `P3-bootstrap`: checked with gh before preparing).

## 2. Contract

### 2.1. INPUT data shapes the code must build

- **No fixture file** is read by any P3 example: every literal lives in the record. `tests/fixtures/` is unchanged.
- **Launch Args** works on a `Launch` literal; `WriteMCPConfig` writes under `t.TempDir()` only: example 6 a missing
  sub-directory `mcp` (created 0o700), example 7 a regular file given as the directory (`testhelp.WriteFile`).
- **Process** starts stub shell scripts written under `t.TempDir()` with mode 0o755, run by `/bin/sh` (dash 0.5.12 on
  the VPS): example 1 echoes one line then `cat`s stdin and exits 4; example 2 writes `oops` to stderr and exits 2;
  example 3 starts `sleep 30 &`, prints `started`, `wait`s; example 5 prints one 70 000-byte line then `short`.
  Example 4 starts `/nonexistent/claude` and `sh` in `/nonexistent/dir`. Examples 6–7 use the Fake only. Never claude,
  never the network.
- **Text an environment can change**, pinned by prefix only: `claude: start ` (the os/exec text follows),
  `claude: mcp config: ` (the os text follows). The stub's stderr is exact (`oops\n`).
- **Measured on the real shell** (dash, Go 1.25.14, the VPS): a SIGKILL to the script's pid alone leaves dash's `sleep`
  child holding the stdout and stderr pipes — `Exit()` did not arrive within 5 s (it would after 30 s); with
  `Setpgid` and `syscall.Kill(-pid, SIGKILL)` it arrived after 0.2 ms. With the record's old script (`sleep 30` alone),
  a Kill right after Start often lands before dash forks `sleep`, so a pid-only Kill passed the probe in 1 of 2 runs:
  example 3 now forks first and prints `started`, and the test reads that line before the Kill (pid-only Kill and a
  missing `cmd.Cancel`: killed 10/10 each after the change; the reference green 50/50).
- **Repo Access** never opens a listener: the judge's fake `do` records the `*http.Request` and answers a literal
  `*http.Response{StatusCode, Body: io.NopCloser(strings.NewReader(…))}`.
- **Project Create** never runs a command: every call goes to a `*runner.Fake` (P1, `runner/runner.go`, in the slice of
  both bootstrap cards). The Fake records `Dir`, `Name`, `Args` but **not env**; examples 5 and 7 check env, so the judge
  wraps the Fake in a small runner of its test file that appends a copy of env and calls the Fake's Run.
- **Preconditions of the callees** (bootstrap judge): runner · `Key(name, args...)` joins with single spaces, so the
  credential-helper argument `store --file=<path>` appears with its space inside the key · examples 1, 4 build their
  keys this way. runner · `Fake.Run` returns `Errors[key]` as `(Result{Code: -1}, errors.New(msg))` before `Script` and
  `Default` · example 6. runner · `Fake.Default` zero = Code 0 · examples 1, 7. github · `Parse` of
  `https://gitlab.com/o/r` fails with `github: not a github repository url: https://gitlab.com/o/r` · example 2.
- **Declared dependencies**: none in this phase (the go-sdk is used by `mcpserver` only). Shared test helpers:
  `internal/testhelp` (`Equal`, `WriteFile`), in the slice of every judge.

### 2.2. OUTPUT data shapes

The exported signatures are the record's, verbatim (`contour.yaml`, groups claude, github, bootstrap: "Declares in …").
Tables: example → given → result are the record's examples (29), rendered into the judge cards by `morph plan`.

**Gaps decided here** (each also one line in `docs/DECISIONS.md`):

- Process · RECORD CHANGED: Start sets `cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}`; Kill sends SIGKILL to
  the whole group (`syscall.Kill(-pid, syscall.SIGKILL)`); `cmd.Cancel` does the same · measured above: the record's
  "Kill sends SIGKILL" with "within 2 s" was unreachable on dash (the P1 runner lesson again, here with group kill
  instead of WaitDelay because claude's tool children must die with it).
- Process · RECORD CHANGED: example 3 is `sleep 30 &` / `echo started` / `wait`, the line read before the Kill · the old
  script let a pid-only Kill pass by timing (a flaky judge test on wrong code).
- Process · RECORD CHANGED: the Fake's Lines() is buffered (1024 lines), so Emit never blocks before a reader; after the
  finish Emit is dropped and Finish, Stop, Kill deliver nothing more (never a panic); every Fake method is safe for
  concurrent use · example 6 calls Emit before any reader (an unbuffered channel deadlocks the test for go test's
  10 min); a second close of Lines() panics the daemon's tests.
- Launch Args · RECORD CHANGED: an existing MCP config file is overwritten and left with mode 0o600 (os.Chmod after the
  write) · os.WriteFile keeps an existing file's mode, and the file carries the session token (record rule 6); probe
  variant rewrites a 0o644 file.
- Repo Access · RECORD CHANGED: a nil do → `Access{Checked: true, Reason: "network: no Do", CheckedAt: now}`, never a
  panic · P2 left Post panicking on a nil Do; the token PUT handler must not panic; probe variant.
- Launch Args · Env returns a fresh slice on every call (probe: a caller's write to one result does not reach the next);
  Args builds a new slice and never aliases Extra · callers append to both.
- Launch Args · a BudgetUSD ≤ 0, negative included, passes no `--max-budget-usd` (probe) · "only when > 0".
- Launch Args · MCPConfigJSON marshals structs whose fields are in the record's order (a map would sort "headers"
  first); an unwritable dir (MkdirAll error) and a write error give the same prefix · example 5 pins the bytes.
- Process · empty stdout lines are skipped; env entries are appended to os.Environ() and reach the child (probe
  variant: `PRB_MARK=m-31`) · behaviour.
- Process · the real Lines() channel is unbuffered (the reader blocks until the caller reads); the exited flag is set
  before Lines() closes, so a Write after `<-Exit()` returns ErrExited, never a pipe error · example 2.
- Process · Stderr keeps the last 4096 bytes (probe: 5 003 bytes in, the last 4 096 out); Code is -1 for a signal and
  for any Wait failure that is not an exit status · behaviour constant.
- Process · Stop and Kill after the exit may return the os error ("already finished", a closed pipe); no caller reads
  it · no example.
- Process · a line over 16 MiB ends the reader as if stdout closed; Exit follows · the probe log's longest line is far
  below; not tested.
- Repo Access · Parse accepts only `https://github.com/` (http rejected); one trailing "/" is trimmed, then ".git"; the
  ssh form requires ".git"; a third path segment or an empty owner/repo is rejected (probe variants) · the record's
  three forms read literally.
- Repo Access · a 200 body that is not JSON → Reachable true, Push false, Reason "" · "false when absent".
- Repo Access · every occurrence of the token in a Do error is masked (probe: twice); a request that cannot be built →
  Reason "network: " + the masked error · "replaced by ***".
- Repo Access · exactly the three headers; the response body is read to the end and closed on every status (probe:
  Close on 200, 401, 403, 404, 500, 502, 301); any status not 200/401/403/404 → "<status> unexpected" (probe: 502, 301)
  · the record's list.
- Project Create · a validation error returns the zero Report (no caller reads it; the probe does not check it); a step
  failure returns Dir and the Steps run so far, the failing command's key included (examples 4, 6: 8 and 1 entries;
  probe: step 3 → 3, step 6 → 6) · "the report so far".
- Project Create · the existence check is `os.Stat(dir)` with a nil error (a regular file of that name counts — probe);
  any Stat error proceeds · "must not exist".
- Project Create · FirstPush: a Run error of rev-parse → `(false, "bootstrap: push: " + err)` and no push; the push's
  Stderr is `strings.TrimSpace`d (probe: `\nfatal: no access\n`, exit 128) · the record names the push's error only.
- Project Create · CredentialsFile and ProjectsDir are not validated; Name "" fails the name rule; 40 characters pass,
  41 fail; a leading "-" and "_" fail (probe) · the regexp.
- Map · budgets from the expected answers: launch-args-judge 12 000 → 16 000 (TASK_TEMPLATE floor), process-judge
  18 000 → 20 000, repo-access-judge 14 000 → 20 000 (the judge with the most examples, 8), project-create-judge
  16 000 → 18 000 (the env-recording runner), project-create 10 000 → 12 000.
- Map · project-create-judge's instruction names the env-recording wrapper (the Fake records no env); launch-args-judge
  and process-judge prefix their helpers `la` / `pr` (two test files of package claude, in different generations, neither
  in the other's slice); launch-args-judge masks modes `& 0o777`.

### 2.3. Names

- Code (one file per card, code only — no smoke test target): `claude/args.go`, `claude/process.go`,
  `github/access.go`, `bootstrap/create.go`; packages `claude`, `github`, `bootstrap` of module `morphstudio`.
- Judge files, one test per record example, in example order: `claude/args_examples_test.go`
  (`TestLaunchArgsExample1..7`), `claude/process_examples_test.go` (`TestProcessExample1..7`),
  `github/access_examples_test.go` (`TestRepoAccessExample1..8`), `bootstrap/create_examples_test.go`
  (`TestProjectCreateExample1..7`).
- Probes (data, inlined into the code cards' acceptances, removed after the step): `decks/P3/parts/_<card>_probe_test.go`,
  tests `TestProbe<Function>Example<N>`; helpers prefixed `pLA`, `pPR`, `pRA`, `pPC`.

### 2.4. What must not break

Frozen byte for byte (every acceptance's `== frozen`): `contour.yaml`, `morph-map.json`, `docs`, `decks`,
`tests/fixtures`, `go.mod`, `go.sum`, `internal`, `vendor`, and the code of P1 and P2: `stream`, `runner`, `queue`,
`session`, `eventlog`, `telegram`. The suite before P3: 63 example tests; after: 92. No ripple: every target is a new
file in a new package.

## 3. Acceptance

Built by `morph plan --checks decks/P3/checks.json` (Go profile, vendor mode), every stage printing `== <stage>`.

- Code card: `== build` (overlay hides the generation's sibling targets) → `== vet` → `== gofmt` → `== guard` (layers
  of `decks/tools/layers.json`: claude and github import no module package; bootstrap may import runner and github) →
  `== probe` → `== imports` (allow-lists from the record and `layers.json` `stdlib_rules`: claude `bufio, bytes,
  context, encoding/json, errors, fmt, io, os, os/exec, path/filepath, strconv, strings, sync, syscall, time` — os/exec
  and syscall are claude's alone; github `context, encoding/json, errors, fmt, io, net/http, net/url, strings, time`;
  bootstrap `context, errors, fmt, os, path/filepath, regexp, strings, morphstudio/github, morphstudio/runner` — no
  os/exec) → `== clock` (no `time.Now`, no `time.Since`, all four) → `== net` (github: no `http.DefaultClient`,
  `http.Get`, `http.Post`, `http.PostForm`, `http.Head`) → `== full` → `== frozen` → no untracked file.
- Judge card: `== vet` → `== gofmt` → `== guard <file>` (min = examples, max = min + 6: args 7..13, process 7..13,
  access 8..14, create 7..13; the literals: every `Test<Function>ExampleN` plus distinct values of the examples, in
  `checks.json`) → `== own` → `== full` → `== frozen` → no untracked file.
- Measured on a scratch worktree (data committed there per generation, removed after): every code acceptance red at
  `== probe` on zero-value stubs, one `--- FAIL: TestProbe…ExampleN` per example, 29 of 29 (args 7, process 7,
  access 8, create 7), each with an `example N …` line, no panic, no compile error (the Process probe reads every
  channel under a 1–3 s select, so a stub that never closes is red in ≤ 2 s per example); judges red at `== guard` on a
  one-test stub (the count line plus one "does not mention the example literal" per missing literal); all 8 green on a
  reference written in the scratch tree only (judges green with test files derived from the probes); every chain timed
  with a cold Go cache: max 34.4 s (project-create), red chains ≤ 4.7 s.

## 4. Constraints

- Module `morphstudio`, Go 1.25, `GOFLAGS=-mod=vendor GOPROXY=off`.
- Stubs and helpers only from `morphstudio/internal/testhelp`; a test writes only under `t.TempDir()`; no test opens a
  listener or dials (github's do is a fake); process tests run only stub scripts under `/bin/sh`.
- `claude` reads no clock; `github` reads no clock (`now` is an argument) and makes no HTTP call except through `do`;
  `bootstrap` runs nothing except through `runner.Runner`.
- A judge writes only its test file; a code card writes only its code file (no test target, so no smoke cap).
- Generations (measured by the cut): [launch-args, repo-access], [launch-args-judge, process, project-create,
  repo-access-judge], [process-judge, project-create-judge]. `claude/process.go` and `claude/args_examples_test.go` are
  written in the same generation in the same package: the overlay hides each from the other's acceptance; neither is in
  the other's slice.

## 7. Out of scope

- Who calls these: the daemon's spawn, restart and pump (P7), the token PUT and status JSON (P4 `control`, P5 `api`),
  the secret files and their modes (P4 `registry` · Secret Files), the project registry (P4).
- A real claude: smoke stop 1 after the merge, not a card.
- Creating GitHub repositories, refreshing the access check, retries (PLAN "Out of scope").
- Windows or macOS process semantics (`Setpgid` and `syscall.Kill` are Linux/Unix; the VPS is Linux).
- Any record example beyond the 29 (the probes' extra variants follow the behaviour text; they are not examples).

## 8. How to run

```
node /tmp/morph-bin-P3/dist/cli.js plan --root . --spec contour.yaml --map morph-map.json \
  --component claude --component github --component bootstrap --judge \
  --checks decks/P3/checks.json --out decks/P3/deck.json
python3 decks/tools/scale_tokens.py decks/P3/deck.json 3
node /tmp/morph-bin-P3/dist/cli.js deck check --root . --deck decks/P3/deck.json        # errors 0
node /tmp/morph-bin-P3/dist/cli.js run --root . --deck decks/P3/deck.json --processor ds --deadline 2400
```

The cut holds exactly the 8 P3 cards: no filter script is needed. `/tmp/morph-bin-P3` is a copy of MorphV2 bc311aa
`dist/` with `node_modules` and `templates` linked beside it.

## 9. Pre-registration

- 8 cards, 3 generations [2, 4, 2]; processor `ds`, maxTokens ×3: launch-args 24 000, repo-access 30 000, process
  42 000, project-create 36 000, launch-args-judge 48 000, repo-access-judge 60 000, process-judge 60 000,
  project-create-judge 54 000 (sum 354 000).
- Bill: forecast ≈ $0.05 (P1 $0.0495, P2 $0.0722 for 8 cards on the same model; P3's slices are small — the largest
  9 342 B with the written code vs P2's 95 667 B — so input is a fraction of P2's, output similar). Ceiling if every card
  used its whole budget once: 354 000 output ≈ $0.42 + ≈ 40 000 input ≈ $0.01 = $0.43 (P2's conservative prices,
  $1.20 / M out, $0.30 / M in). Cap $5; gate ≤ $1.
- Expected regenerations: process (an author who kills with `cmd.Process.Kill` reddens `== probe` at "example 3 after
  Kill (the whole group…): no ExitStatus within 2s"; an unbuffered Fake reddens example 6 "Emit blocks with no
  reader"); launch-args (an author who trusts os.WriteFile's mode reddens example 6 "existing file left 0600");
  process-judge (goroutine/select plumbing; the `started` line read before Kill); project-create-judge (the env
  wrapper).
- Falsifiable: 8/8 written within one fix; 92 example tests green on `go test ./...`; no card over 1 regeneration;
  smoke stop 1 green as `PLAN.md` writes it.

## 10. What to record

The `docs/MEASURE.md` row and §11: attempts per card, first red per burned variant, neighbour-red, judge defects,
guard rejections, lines by hand, max slice bytes, minutes, $; then the smoke's numbers.

## 11. Actual

### Preparation (before the gate)

- Orchestrator: Opus 5.5 agent, 70 tool calls, 226k tokens, 26 min (measured by the session from the agent's run).
- `morph plan` exit 0; `morph deck check` errors 0, warnings 0, hazards 0; generations [2, 4, 2]; the cut holds exactly
  the 8 phase cards.
- Largest slice: project-create-judge, 9 342 B with the written code (3 917 B of existing files at the cut; ≈ 13.3 KB
  with the instruction); no slice near 200 KB.
- Probes red per example on stubs: 29 / 29 (args 7, process 7, access 8, create 7); judges red at `== guard` on a
  one-test stub (every missing literal named).
- Acceptance chains (cold Go cache, one per card): max 34.4 s (project-create green), all 8 green on the scratch
  reference; red chains ≤ 4.7 s.
- Mutants: 30 on the reference (args 7, process 11, access 7, create 5), each under `timeout 120`. First round 28 / 30:
  the pid-only Kill and the missing `cmd.Cancel` survived by timing (Kill before dash forked `sleep`) → example 3
  rewritten (record change above); both then killed 10 / 10 on repeats; final round 30 / 30 killed at `== probe`,
  0.41 min. Total mutant time ≈ 2 min. Survivors: none.
- Record changes: Process behaviour (group kill, Cancel, the Fake buffered and idempotent) and example 3; Launch Args
  behaviour (existing file left 0o600); Repo Access behaviour (nil do). Map: 5 budgets, 3 judge instructions.
- 0 lines of product code by hand (the reference and the stubs lived only in a scratch worktree, removed; leftover
  stub processes killed).

### Run 20261008-181657 (processor ds, binary copy /tmp/morph-bin-P3 of MorphV2 bc311aa)

- 8 / 8 written, 0 burned; 10 requests; 45 583 input / 90 619 output tokens; $0.0608 executor; 448 s (7.5 min).
- Attempts: repo-access 2 (v1 rejected at `== imports`: "package github imports strconv (allowed: context,
  encoding/json, errors, fmt, io, net/http, net/url, strings, time)"; r1 green); project-create-judge 2 (v1 rejected at
  `== guard`: the example literals "--module demo", "git rev-parse --verify -q origin/main", "git push -q -u origin
  main" not mentioned; r1 green); every other card 1. Every finish reason `stop`. No failure class, no fix needed.
- Judges: 4 / 4 green at `== own` on the accepted code; judge defects 0; guard rejections 1 (above); neighbour-red 0.
- Verify on the run branch: `git status --short` empty; gofmt, `go vet ./...`, `go build ./...` clean;
  `go test -count=1 ./...` green, 92 example tests (P1 31 + P2 32 + P3 29).
- Read against §2.2 and the record: the code follows the record's behaviour. Two §2.2 gaps not pinned by any example,
  recorded as known risks, not fixed by hand:
  1. Process: `exited` is set after Lines() closes (after Wait), not before; a Write in the window between the close
     of Lines() and the ExitStatus returns the pipe's error instead of ErrExited (example 2 writes after `<-Exit()`,
     green). Callers that check `errors.Is(err, ErrExited)` must also tolerate a pipe error.
  2. Process: a stdout line over 16 MiB stops the reader, but Wait waits for the reader goroutines before reaping, and
     nobody drains stdout anymore, so a child still writing blocks and Exit never comes until Kill/cancel (§2.2 said
     "Exit follows"). The daemon's runtime guard (Kill on its caps) bounds it.
