# Task P9a — users (User Registry, User Secrets, Token Lookup) and the registry `User` field

> The spec of one phase. The card breakdown is never written here: that is the deck, cut by `morph plan` from the record
> and `morph-map.json`. The spec is the contract a reader understands months later.

## 1. Why this

Row P9a of PLAN_ONBOARDING_VPS.md §5: onboarding needs real users before any tester connects — who may talk to morphd,
under which Linux user, with which token — and their secret files. Per-user tokens (Requirement, DECISIONS 11.10 D2/D5)
have no owner today: `registry` projects carry no user, and no package knows whose bearer token a request carries.
This reddens once a tester is onboarded at all: today a second tester's token is indistinguishable from the first's.

Measured: the record grows by one Component (`users`, 3 Functions, ≈ 10 KB raw) and `registry · Project Registry`
gains one field; the registry group stays ≈ 9 KB — both far under the 30 KB Component limit.

## 2. Contract

### 2.1. INPUT data shapes the code must build

- `users.User` (the record, Function User Registry): five fields, JSON tags `name`, `linux_user`, `chat_id` (int64),
  `token_hash` (the hex SHA-256 of the bearer token, encoding/hex lowercase), `created_at`. Defined in users/users.go —
  in the slice of every card that builds it.
- `registry.Project` — the existing struct of `registry/registry.go`, this phase with the new `User string` field
  (json tag `user`); built by callers outside this deck, only decoded inside it.
- **Fixture `tests/fixtures/users/users-3.json`** (new, committed data): a JSON array of exactly **three** users
  (alice chat 7, bob chat 42, carol chat 99), sorted by name, each with a real SHA-256 hex `token_hash`
  (hex of "tok-alice-1", "tok-bob-2", "tok-carol-3", measured with sha256sum). It is the **exact input** of Token
  Lookup example 1 (copied as `users.json` into the base and opened); no other Function reads it. LookupToken maps
  each of the three tokens to its user and nothing else; the fixture is never modified by a test.
- The admin token of the examples is the literal `admin-op-9` (hash measured); its value lives in the record examples
  and in no fixture.
- No declared third-party dependency: package users imports nothing of the module, the standard library only
  (crypto/sha256, crypto/subtle, encoding/hex, encoding/json, errors, fmt, os, path/filepath, regexp, sort, strings,
  sync, time). Acceptances run offline (`-mod=vendor`, `GOPROXY=off`).

### 2.2. OUTPUT data shapes

Per Function, the record's tables are the contract (group `users`, three Functions; group `registry`, Function
Project Registry). Signatures as declared in the record; every sentinel and error text pinned there
(`user not found`, `user exists`, `bad user name`, `chat already has a user`, `no secret`, `empty secret`,
`users: read `, `users: unknown secret kind %q`; registry's unchanged). Orderings: List sorted by Name,
empty non-nil. Modes: base 0700, user/project dirs 0700, state and secret files 0600, even when the file existed
with another mode.

Gaps decided here (also in DECISIONS.md):
- `Open(base, adminToken)` takes the admin token at construction (not a file): the operator configures one token,
  and IsAdmin stays a pure constant-time compare; an empty admin token disables admin entirely, never a panic —
  a second config surface was not worth it for one value.
- `ReadSecret` exists (morphd's own use: the claude environment, a credential file) while no HTTP/MCP handler of this
  phase returns it; the Guardrail example per kind pins that no error text ever contains the value.
- `Delete(name)` removes the user's whole directory: no Function of this phase needs per-file cleanup, and a stale
  secret file must never outlive its user.
- `registry.Project.User` is the Linux user name (a plain string, not a users.User reference): registry stays
  decoupled from package users (the registry file must load without the users state).

### 2.3. Names

Package users: `users/users.go`, `users/secrets.go`, `users/tokens.go`; tests `users/users_examples_test.go`,
`users/secrets_examples_test.go`, `users/tokens_examples_test.go` (judges; helper prefixes `ur`, `us`, `tl`),
`registry/registry_examples_test.go` re-cut (prefix `rg`). No smoke test per card: a probe covers every card.

### 2.4. What must not break

Frozen: every directory but `users` (new) and `registry` (re-cut) — the list in `decks/P9a/checks.json`.
Callers of registry.Project (daemon, api, bootstrap, cmd) use keyed literals and compile unchanged (scout + grep:
no positional composite literal of Project exists). 237 existing tests stay green except the re-cut judge's own file.

## 3. Acceptance

Built by `morph plan --checks decks/P9a/checks.json`, narrow to broad: parse → lint → `decks/tools/guard.mjs`
(package imports, no clock, no net; judge example literals) → the probe (`decks/P9a/parts/_<card>_probe_test.go`,
red per example with a readable line) → the card's own tests → the full suite → frozen files → no untracked file.
Judge bounds derived: min = the record's examples, max = min + 6. The chain is timed on a dry tree: under 250 s.

## 4. Constraints

The project's: module imports only, no `any` in users files beyond the two registry state methods, a test writes only
under a temporary directory, package users imports nothing of morphstudio (checked by the guard), stubs from the
record's names only. A file a card writes is in no sibling's slice. Phase's own: the fixture is committed data and
never written by a card; no error text, log or argv may contain a token, its hash or a secret value (Guardrail
Secrets Never Logged, widened this phase).

## 7. Out of scope

- The HTTP/MCP wiring of per-user tokens (api, mcpserver) — P10.
- `onboard`, `tgbot`, `spawn` packages, `Launch.Tester`, the sudo wrappers — P9b/P10/P11.
- The VPS smoke (T1) — the PM's session; this phase's live reproduction is the $0 laptop part only.
- Telegram chat ↔ user handling beyond the ErrChatTaken rule — the bot lands with onboard.

## 8. How to run

```
morph plan --root . --spec contour.yaml --map morph-map.json --component users --component registry --judge \
  --checks decks/P9a/checks.json --only user-registry,user-secrets,token-lookup,project-registry,user-registry-judge,user-secrets-judge,token-lookup-judge,project-registry-judge \
  --out decks/P9a/deck.json
python3 $MORPHV2/decks/tools/scale_tokens.py decks/P9a/deck.json 3        # ds thinks before it writes
morph deck check --root . --deck decks/P9a/deck.json                        # errors 0
~/bin/morphv2 run --root . --deck decks/P9a/deck.json --processor ds --deadline <from the deck>
```

## 9. Pre-registration

8 cards, 2 generations (code then judges), 4 + 4. Transaction: yes (`--only` + `--checks`); the registry Component
holds Secret Files outside the phase, and registry.go is re-cut whole, so every card is written first and a red
card rolls its subset back. Expected first reds: user-registry (probe, `stub Open`), then its dependants;
project-registry re-cut is red on **every** probe row by design (intent generate: the stub throws, the old committed
test file is not the criterion). Bill: forecast $0.20 (row P9a), cap $5. Falsifiable: a green run with the whole
suite green and the fixture-lookup example green means Token Lookup and the User field landed; a red probe row that
is green in the run means the record or probe was wrong, not the card.

## 10. What to record

The row of docs/MEASURE.md and §11: attempts, first red per burned variant, neighbour-red, judge defects, guard
rejections, lines by hand, max slice bytes, minutes, $.

## 11. Actual

Filled after the gate (preparation numbers) and after the run; a fix gets its own sub-section with every spec
change before/after.
