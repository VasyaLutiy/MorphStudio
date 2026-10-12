# TASK_P9a_users.md — new `users`; re-cut `registry · Project Registry`

The row P9a of PLAN_ONBOARDING_VPS.md §5. The card breakdown is in the deck (`decks/P9a/deck.json`), cut by
`morph plan`; never here.

## 1. Why this

The VPS onboarding epic (E7) needs per-user tokens (PLAN_ONBOARDING_VPS.md §7 "Per-User Tokens"): every HTTP / MCP bearer
token belongs to one user and sees only that user's projects; only the operator's admin token sees all. Today there is
no user at all: `registry` stores projects with no owner. Two changes: a new Component `users` (user store, user
secrets, token lookup) and `registry.Project` gains `User` — the Linux user a project's sessions run as, which `users`
and `spawn` later read through the registry. Record sizes: `users` ~9.9 KB new (limit 30 KB), `registry` ~7.2 KB → ~7.4 KB.

## 2. Contract

### 2.1. INPUT data shapes the code must build

- `base/users.json` — a JSON array of `users.User` objects (`name`, `linux_user`, `chat_id`, `token_hash`,
  `created_at`), read by `Open` into `[]User`, written whole by `Add` (json.MarshalIndent, two spaces, empty → `[]`).
  Defined in `users/registry.go` (Component `users`, Function "User Registry").
- `tests/fixtures/users/users-3.json` — one object of that type: an array of three `User` (ada, mid, zoe, sorted by
  name). `Open` over a copy → `List` equals the fixture decoded (`[]User{ada, mid, zoe}`), `Get("mid")` → mid
  (LinuxUser "mu-mid", ChatID 555, TokenHash "73e7b18a…", CreatedAt 2026-10-11T09:02:00Z). Read only by example 5 of
  "User Registry" (its `given` is exactly that file) and its judge; no other example has a file input.
- A user directory `base/<name>/` (0o700) with secret files `<kind>` (0o600), kinds `anthropic-key` and `github-token`
  (`SecretKinds`), read and written whole by value (`users/secrets.go`, Function "User Secrets").
- `registry.Project` literals — the record's examples construct them; `queue.Caps` (`queue/queue.go`) is in the slice of
  every card that builds a `Project` (the re-cut `project-registry` and its judge).
- The bearer token of a lookup — a plain string; hashed with sha256 + lowercase hex (`hashToken` in `users/registry.go`),
  compared constant-time (`crypto/subtle`) in `users/token.go` (Function "Token Lookup").

**Declared dependencies:** none new — package `users` imports the standard library only
(crypto/sha256, crypto/subtle, encoding/hex, encoding/json, errors, fmt, os, path/filepath, regexp, sort, strings, time)
and nothing of the module; `registry` is unchanged (encoding/json, errors, fmt, os, path/filepath, regexp, sort, time,
`morphstudio/queue`). Acceptances run offline (`-mod=vendor`, `GOPROXY=off`); the vendor tree is committed.

**A judge's setup across Components:** no judge of this phase calls another Component — the `users` judges build only
their own store (`Open` + `Add` per their Function's examples); the `project-registry` judge is unchanged in shape.
Distinct markers: user names ada/mid/zoe, tokens tok-ada/tok-zoe/tok-mid/admin-0/tok-9, secrets sk-ant-xyz1/sk-ant-new/
ghp_abc/sk-very-secret-2/ghp-very-secret-2, Linux users mu-* — none a substring of another. No harness skeleton beyond
the store literal (`Open` + `Add`).

**Output budget from the answer:** `user-registry` 12000, `user-secrets` 8000, `token-lookup` 8000,
`user-registry-judge` 20000, `user-secrets-judge` 16000, `token-lookup-judge` 12000, `project-registry` 12000,
`project-registry-judge` 18000 (unchanged; ~20 KB answer ≥ 28 000 after the ×3 factor).

### 2.2. OUTPUT data shapes

The exported signatures are in the record (`contour.yaml`, Functions "User Registry", "User Secrets",
"Token Lookup", "Project Registry"); every error text and ordering rule is there, pinned: the `users: ` read-error
prefix, `users: unknown secret kind %q`, the sentinels ("user not found", "user exists", "bad user name",
"empty token", "no secret"), sorting by `Name` with `json.MarshalIndent` (two spaces) and the empty list as `[]`,
the 0o700/0o600 modes, `Add` overwriting the given `TokenHash` with the hash of the token, `Lookup`/`IsAdminToken`
comparing hex hashes constant-time. `registry.Project` gains `User string` (`json:"user"`), stored, written and read
like the other string fields; nothing else of `registry` changes.

### 2.3. Names

Modules: `users/registry.go`, `users/secrets.go`, `users/token.go` (new); `registry/registry.go` (re-cut).
Test files: `users/registry_examples_test.go`, `users/secrets_examples_test.go`, `users/token_examples_test.go`
(new, one `Test<Function>Example<n>` per record example, in example order), `registry/registry_examples_test.go`
(re-cut, the same seven names). Probes cover every code card (`decks/P9a/parts/`); no card targets a test file, so
no card has a smoke test.

### 2.4. What must not break

`registry/secrets.go` and `registry/secrets_examples_test.go` byte for byte (frozen in `decks/P9a/checks.json`);
`registry`'s sentinels, name rule, state files and secret paths unchanged; every existing package and its tests
green (`go test ./...` after the run: 237 + the new tests). The ripple of `Project` gaining a field: literals that
construct `Project` are keyed, so every existing caller compiles unchanged (daemon `CreateProject`/`AdoptProject`,
`bootstrap`, `cmd`); JSON files on disk gain a `"user"` key with no old key lost.

## 3. Acceptance

Built by `morph plan --checks decks/P9a/checks.json` (`--only` — a subset transaction: `registry` holds the Function
"Secret Files" outside this phase). Narrow to broad: parse → lint → `decks/tools/guard.mjs` (imports per the record,
no clock) → the probe (red per example) → the full suite (`-mod=vendor`, `GOPROXY=off`) → frozen files unchanged → no
untracked file left. Judge bounds: `user-registry-judge` 7–13, `user-secrets-judge` 7–13, `token-lookup-judge` 4–10,
`project-registry-judge` 7–13 (min = the examples; max = min + 6).

## 4. Constraints

Module `morphstudio`, Go 1.25, one directory per package, the standard library plus the vendored go-sdk (not used by
`users`); no `any` outside `SaveState`/`LoadState` (unchanged); stubs only from the shared helpers module
(`morphstudio/internal/testhelp`, `Equal`); a test writes only under `t.TempDir()`; no clock (`time.Now`/`time.Since`
nowhere — the dates are arguments); `users` imports nothing of the module; a card writes only its target; a judge
writes only its test file. The guard allows in package `users` only: crypto/sha256, crypto/subtle, encoding/hex,
encoding/json, errors, fmt, os, path/filepath, regexp, sort, strings, time.

## 7. Out of scope

- Per-user auth on HTTP and MCP (401/404 bodies, `projects_list` filtering) → P9c.
- The `spawn` Component and Tester Argv/Runner → P9b.
- Telegram chat binding (`/start <code>`, One Chat One User) → P10a.
- Any use of `Project.User` beyond storing it (the daemon and `spawn` read it in P9b/P9c).
- The VPS smoke (probe E territory) → the PM's, not this phase's.

## 8. How to run

```
morph plan --root . --spec contour.yaml --map morph-map.json --component users --component registry --judge \
  --checks decks/P9a/checks.json --only user-registry,user-secrets,token-lookup,project-registry,user-registry-judge,user-secrets-judge,token-lookup-judge,project-registry-judge \
  --out decks/P9a/deck.json
python3 decks/tools/scale_tokens.py decks/P9a/deck.json 3
morph deck check --root . --deck decks/P9a/deck.json
~/bin/morphv2 run --root . --deck decks/P9a/deck.json --processor ds --deadline <from the deck> 
```

## 9. Pre-registration

Eight cards, three generations: gen 1 `user-registry`, `project-registry` (+ nothing else); gen 2 `user-secrets`,
`token-lookup`, `user-registry-judge`, `project-registry-judge`; gen 3 `user-secrets-judge`, `token-lookup-judge`.
Every card regenerates whole (intent `generate`; `project-registry` re-cuts an existing file and never sees the old
text — its stub is the names of the record, every body throwing, so every probe row goes red on the stubs by design,
the unchanged rows included). First red per burned variant is the probe's example line naming the example. Bill:
forecast ≈ $0.20 on `ds` (row P9a), cap $5 (header). Falsifiable claim: after the run, `go test ./...` is green with
the new examples, `registry/secrets.go` unchanged, and a user's token never appears in any error text or file.

## 10. What to record

The row of `docs/MEASURE.md` and §11 of this file: attempts, first red per burned variant, neighbour-red, judge
defects, guard rejections, lines by hand, max slice bytes, minutes, $.

## 11. Actual

(filled after the gate and the run)