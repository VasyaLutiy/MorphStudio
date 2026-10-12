# TASK P9a — new `users`; re-cut `registry · Project Registry`

Phase P9a of `PLAN_ONBOARDING_VPS.md` §5 (epic E7), cut from `main` @ d92a8b3. The record is `contour.yaml`, groups
`users` (new) and `registry` (Function Project Registry). This file is for the gate and the report. No card reads it.

## 1. Why this

E7 makes one morphd serve 3 testers (D1, D4). P9b–P11 need: who a tester is (name, Linux user `mu-<name>`, Telegram
chat), what a bearer token belongs to, and where that tester's own secrets are (D5: an Anthropic API key and a GitHub
PAT, never in chat). P9c filters projects per user, so every project needs an owner. Today the record has no user,
and morphd has one token (`MORPHD_TOKEN`) that sees everything. Record size: the `registry` block goes from 8 162 to
10 139 bytes, and `users` is new at 15 858 bytes. Both are under 30 000.

## 2. Contract

### 2.1 INPUT shapes

- `tests/fixtures/users/users-3.json` is one file and the exact `base/users.json` of three users, measured with
  `json.MarshalIndent` on Go 1.25 (spike, no trailing newline): ann (chat 111111111, token `mcp-ann-0001`), bob
  (chat 0, token `mcp-bob-0002`), eve (chat 333333333, no token). The hashes in the examples are `sha256sum` of those
  tokens. User Registry examples 2–6 and 8, every User Secrets example and Token Lookup examples 1–4 start from a copy
  of it in `t.TempDir()`. Example 2 rebuilds it byte for byte.
- `queue.Caps` (queue/queue.go) for Project literals.

### 2.2 OUTPUT shapes

Signatures and errors are in the record's behaviours. Every error text of `users` begins with `users: `. Modes are
0o600 for files and 0o700 for dirs.

Gaps decided here:
- Tokens are minted by the caller (cmd/morphd or onboard). `users` only hashes them. **Why:** the canonical Guardrail
  No Clock covers crypto/rand.
- The admin token is an argument of Lookup. It is not stored. **Why:** it stays `MORPHD_TOKEN` of the config.
- The users registry has its own base (`<state>/users`). **Why:** a user dir must not collide with a project's state
  dir.
- `Project.User` uses `json:"user,omitempty"`. **Why:** the VPS projects.json without the key opens unchanged, and
  the operator's projects keep no key.
- Project names stay global, and `Add` does not check `User` against `users`. **Why:** registry does not import
  users.

### 2.3 Names

Targets are `users/users.go`, `users/secrets.go`, `users/tokens.go` and `registry/registry.go`. Judges write
`users/{users,secrets,tokens}_examples_test.go` and `registry/registry_examples_test.go`. The last one is
re-generated, and TestProjectRegistryExample1–7 survive.

### 2.4 What must not break

Every file outside `users/` and `registry/registry*.go`, byte for byte (the frozen list of `checks.json`).
`registry/secrets.go` reads `r.base`, so the record names that field. 237 tests stay green.

## 3. Acceptance

`morph plan --checks decks/P9a/checks.json --only <8 ids>` builds a subset transaction. Layers come from the record
(contract 3): `registry` → `queue`, and `users` → nothing. The rules are users' Guardrails No Shell, No Clock, No
Network, No Env and Stdlib Only.

## 4. Constraints

Tests only under `t.TempDir()`. Every helper is prefixed per file (`ur`, `us`, `tl`, `rg`; probes `pUR`, `pUS`, `pTL`,
`pPR`).

## 7. Out of scope

- Per-user auth on HTTP/MCP, `projects_list` per user, and project create under `/home/mu-<t>` → P9c.
- Spawn through the wrappers, Launch Args, and the env file from User Secrets → P9b.
- Wiring `users.Open` into cmd/morphd config → P9c (with auth).
- Invite codes, chat binding by `/start` and the secret form → P10a/P11.
- The record's §7 additions (Guardrails Tester Isolation and the widened Secrets Never Logged, Requirements Per-User
  Tokens and One Chat One User): System-level text outside this phase's Functions, left for the PM. P9a carries the
  Secrets-Never-Logged examples per kind inside User Registry ex. 4–5 and User Secrets ex. 6.
- Constant-time comparison is stated in the record but not mechanically checked (no canonical Guardrail says it).

## 8. How to run

```
morph plan --root . --spec contour.yaml --map morph-map.json --component registry --component users --judge \
  --checks decks/P9a/checks.json --only project-registry,project-registry-judge,user-registry,user-registry-judge,user-secrets,user-secrets-judge,token-lookup,token-lookup-judge \
  --out decks/P9a/deck.json
python3 $MORPHV2/decks/tools/scale_tokens.py decks/P9a/deck.json 3
morph deck check --root . --deck decks/P9a/deck.json
morph run --root . --deck decks/P9a/deck.json --processor ds --deadline 3540
```

## 9. Pre-registration

8 cards in 3 generations. Forecast $0.20, cap $5. Most likely first red: a judge (a literal or a mode type).
