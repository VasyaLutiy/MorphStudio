# PLAN_ONBOARDING_VPS.md — beta onboarding and a multi-user VPS (epic E7; E8 deferred)

> **Status: DRAFT 10.10, not approved.** Designed by Claude Code in plan mode with the operator (operator 10.10: no
> PM flow, no Fable architect for this). `PLAN.md` stays frozen until this draft is approved by commit hash; then the
> record (`contour.yaml`, `morph-map.json`) gains the Components below phase by phase, and `PLAN.md` is rewritten
> whole with epic E7 (a plan change rewrites, never appends). Decisions: `docs/DECISIONS.md` 10.10 "product
> direction". Base: morphd P0–P8b built, Smoke stop 3 green (51f9686), 237 example tests.

## 1. Decisions (operator 10.10) — binding

| # | decision |
|---|---|
| D1 | Beta: **3 users, ONE VPS, ONE morphd**. A VPS per user is deferred (§8). |
| D2 | **Two surfaces** (operator 11.10, replaces "the bot is the only remote"): each tester works from **their own laptop with the claude CLI connected to morphd over MCP** (`/mcp`, `/mcp/<p>` through the Cloudflare Tunnel, a per-user token); the **Telegram bot** carries onboarding, alerts (milestones, stops, questions) and possibly a chat with the orchestrator. Cloudflare Access service tokens (one per laptop) — later, not in E7. |
| D3 | **Secrets never in chat.** The bot sends a one-time link (TTL 10 min, one use) to one HTTPS form served by morphd behind Caddy (`themorph.stream`). |
| D4 | **Isolation = one Linux user per tenant** (`mu-<name>`, home 0700). Every claude session runs under that uid via `systemd-run` with `PrivateTmp`, `MemoryMax`, `CPUQuota`, `ProtectSystem=strict`, `ReadWritePaths=<home>`. morphd's own state stays 0700 under `morph`. Containers later. |
| D5 | **Credentials per tenant**: `CLAUDE_CODE_OAUTH_TOKEN` (from `claude setup-token`) or `ANTHROPIC_API_KEY`, plus `CLAUDE_CONFIG_DIR` / `HOME` of the tenant; read from 0600 files; never in argv, a log, a reply, a milestone or an error. GitHub: a fine-grained PAT per tenant over the existing `PUT github-token` path, now per user; the tester creates the empty repo. |
| D6 | One session per tenant, ≤ 3 in total (max-parallel 3, per-user 1). |
| D7 | **No Postgres.** JSON files behind a `Store` interface; pure-Go SQLite (`modernc.org/sqlite`, vendored, a declared Dependency) only when queries or history are needed. |
| D8 | New-project planning is **interactive, plan-mode style**: a claude session in plan mode relayed through the bot (questions as buttons), the plan in the MorphV2 PLAN template, approve by a button = the commit hash, then `plan_load`. |

## 2. Target picture

```
tester's phone                         one VPS (≥ 4 vCPU, 16 GB)
┌──────────────┐  Bot API (long poll)  ┌──────────────────────────────────────────────────────────┐
│ Telegram     │ ◀──────────────────▶ │ morphd (User=morph, state 0700)                          │
│  /start code │                       │  ├ tgbot: getUpdates → commands, buttons → control       │
│  /status …   │                       │  ├ users: users.json, per-user secrets 0600, tokens      │
│  [Option 1]  │                       │  ├ onboard: invite, one-time form, credential check      │
└──────┬───────┘                       │  ├ supervisor/daemon (built): queue, loop, guard, pump   │
       │ one-time link                 │  └ spawn: systemd-run --uid=mu-<n> … claude -p …         │
       ▼                               │        ├ /home/mu-alice/projects/<p>  (claude, morph, go)│
  https://themorph.stream/onboard/<t>  │        ├ /home/mu-bob/projects/<p>                       │
  (Caddy → morphd, form only)          │        └ /home/mu-carol/projects/<p>                     │
                                       └──────────────────────────────────────────────────────────┘
```

## 3. Gate 0 — probes before any record (PM/operator on the VPS, ≤ $0.10 of claude)

Every probe ends with a DECISIONS line and, where named, a recorded fixture. The record is written from the fixtures,
never from memory.

| probe | how | pass criteria | yields |
|---|---|---|---|
| **A. two tenants at once** | `useradd -m -s /bin/bash mu-a` and `mu-b` (home 0700); per tenant a 0600 env file with `CLAUDE_CODE_OAUTH_TOKEN` (never typed on a command line); two parallel `systemd-run --uid=mu-X --pipe --quiet --collect -p PrivateTmp=yes -p MemoryMax=2G -p CPUQuota=100% -p ProtectSystem=strict -p ReadWritePaths=/home/mu-X -p EnvironmentFile=<env> --setenv=CLAUDE_CONFIG_DIR=/home/mu-X/.claude --setenv=HOME=/home/mu-X claude -p --input-format stream-json --output-format stream-json --verbose …`, one PONG turn each | both `init` + `result`; from A `cat /home/mu-b/x` fails; `/tmp` differs; nothing written under `/home/morph/.claude`; `rate_limit_event` per tenant (two tokens) or identical (one token — noted); exit ≤ 5 s after stdin close | `tests/fixtures/stream/probe-tenant.jsonl`; the exact sandbox argv |
| **B. headless plan mode** | `claude -p --permission-mode plan` over stream-json, with and without `--dangerously-skip-permissions`, "write a 3-line plan for a hello CLI" | `ExitPlanMode` arrives as `control_request can_use_tool` (or not); the answer that approves and the one that sends feedback are recorded | `tests/fixtures/stream/probe-planmode.jsonl`; red → P12 falls back to "plan written as a file + approve button" without plan mode |
| **C. Bot API** | a test bot: `getUpdates` long poll with `offset`, `sendMessage` with `reply_markup.inline_keyboard`, a `callback_query` + `answerCallbackQuery`, `deleteMessage` | every call 200; the update shapes recorded; $0 | `tests/fixtures/tg/*.json` (ids and tokens scrubbed) |
| **D. privilege** | morphd (User=morph) starts `systemd-run --uid=mu-a …`: (1) unit `AmbientCapabilities=` or (2) one sudoers line `morph ALL=(root) NOPASSWD: /usr/bin/systemd-run --uid=mu-* *` | the one that works with the least privilege is chosen | the unit/sudoers text in `docs/RUNBOOK_VPS.md` |

## 4. Hand data and ops (never cards)

- VPS: ≥ 4 vCPU, 16 GB (three claude + `go test` do not fit cx23's 2 vCPU / 4 GB) — rescale or a new box (§9.3).
- `tools/add-tenant.sh <name>`: `useradd -m mu-<name>`, `chmod 700`, shared read-only toolchains (Go 1.25, node,
  claude, the morph binary copy) on `PATH`, `~/projects/`; run by the operator (root). morphd never creates users.
- The morphd unit / sudoers change from probe D; the bot token in morphd's `.env` (`TG_BOT_TOKEN`; the per-chat
  `TG_CHAT_ID` becomes the operator's admin chat only).
- Caddy: `handle /onboard/*` → morphd; everything else as now (the session MCP stays 404 from outside).
- Backups of `~/morphd/state` (users, secrets, project state) — a nightly tar 0600 to the operator.
- The beta tenant list (3 names, Telegram handles) — the operator.

## 5. Epic E7 — beta onboarding on one VPS: phases

Each row is one orchestrator phase (one run + at most one fix, cap $5). Card counts are estimates for `morph plan`
with `--judge`; the cut decides. "Stop" = a smoke by the PM/operator before the next phase.

| phase | Component (new / re-cut) | Functions | cards (est.) | depends | oracle and fixtures | by hand before the run | $ est. | risk → treatment | stop after |
|---|---|---|---|---|---|---|---|---|---|
| **P9a** | **new `users`**; re-cut `registry` | User Registry (users.json: name, linux user, chat id, API token hash, created); User Secrets (kinds `claude-token`, `anthropic-key`, `github-token`, 0600, never returned); Token Lookup (bearer → user, constant-time compare); Project Registry + `User` field | 8 | Gate 0 | `tests/fixtures/users/users-3.json`; literal 0600 modes; error texts with the `users: ` prefix | TASK, checks, probes, fixture | 0.20 | a secret in an error text → a Guardrail example per kind | — |
| **P9b** | **new `spawn`** (split off package daemon: daemon/spawn.go, daemon is 27.7 KB); re-cut `claude · Launch Args` | Sandbox Argv (pure: the systemd-run argv of probe A from a Tenant); Launch Args (`Launch.Tenant{User, Home, ConfigDir, EnvFile}`; `Env` = auto-memory off + `CLAUDE_CONFIG_DIR` + `HOME`; the token only via `EnvironmentFile`); Spawn Session (the daemon's spawn through Sandbox Argv for the project's user) | 8 | P9a, probes A, D | argv literals of probe A; `probe-tenant.jsonl` lines | TASK, checks (freeze all but claude, daemon), `_stubs` | 0.30 | cross-file daemon re-cut → subset transaction; caller check of every private symbol (record rule 8) | **smoke T1**: two tenants' sessions in parallel through morphd; probe A's isolation checks hold |
| **P9c** | re-cut `api`, `mcpserver · MCP Mount`, `control` | Router + HTTP Handlers (token → user; a foreign project → 404 `unknown_project`; the operator's admin token sees all); MCP Mount (per-user tokens on `/mcp` and `/mcp/<p>`); Control list filters by user | 8 | P9a | the 401/404 bodies as literals; two users' tokens in examples | TASK, checks, `_stubs` | 0.25 | judges that list routes/tools → re-cut together (P8b lesson) | — |
| **P10a** | **new `tgbot`** (pure) | Update Parse (getUpdates JSON → Update: message / command / callback); Command Route (`/start <code>`, `/status`, `/usage`, `/continue`, `/order <text>`, `/interrupt`, `/restart`, `/help` → a control call for the chat's user; an unknown chat → "not registered"); Keyboard (a pending question → `inline_keyboard`; callback data → `answer {option}`; approve / revise buttons for a plan) | 6 | P9c, probe C | `tests/fixtures/tg/*.json` | TASK, checks, fixtures | 0.20 | Bot API drift → every example cites a fixture | — |
| **P10b** | **new** Poll Loop (in `tgbot`); re-cut `telegram · Milestone Post`, `pump`, `cmd` | Poll Loop (getUpdates with `offset`, 30 s long poll, injected `Do` and clock, backoff on 409/5xx); Milestone Post to the chat of the project's user (admin chat for system lines); the pump's ask/turn posts carry the buttons; Config And Main wires the bot | 8 | P10a | recorded updates; the exact sendMessage bodies | TASK, checks, `_stubs` | 0.30 | pump re-cut (186 KB judge slice) → keep pump-judge's old examples, add new ones only | **smoke T2**: the operator drives one project from Telegram only — status, an AskUserQuestion answered by a button, continue |
| **P11** | **new `onboard`** | Invite (operator `/invite <name>` → one-time code; `/start <code>` binds the chat to the user); Secret Form (one-time link token, TTL 10 min, one use; GET renders the form, POST stores via users; HTML-escaped; nothing echoed back); Credential Check (claude PONG under the tenant sandbox, haiku, `--max-budget-usd 0.01`; GitHub access via `github`); Checklist (chat ✅ claude ✅ github ✅ TG test ✅ → ready, as one bot message) | 8 | P10b | the form HTML as a literal; probe A's PONG lines; github 200/401/404 bodies | TASK, checks, Caddy route (§4) | 0.30 | replay / CSRF → one-use + TTL examples; a token in a log → Guardrail example | **smoke T3**: a real tester onboarded end to end from a phone |
| **P12** | re-cut `supervisor · Phase Loop` (mode "plan"), `tgbot · Keyboard` | Plan Session (spawn with probe B's flags, the idea as the first line; `ExitPlanMode` → the bot posts the plan file + approve / revise; approve → the commit hash → `plan_load`; revise → the feedback as the answer) | 6 | P11, probe B | `probe-planmode.jsonl` lines | TASK, checks, `_stubs` | 0.25 | plan mode absent in headless → probe B's fallback (file + button) | **smoke T4**: three tenants in parallel — idea → plan → approve → first phase |

**Totals**: 7 phases, ≈ 52 cards; executor ≈ $1.80 on `ds` (cap $5 per phase); claude for the smokes ≤ $1.00.
**Done** when smoke T4 is green: three testers, each onboarded from Telegram, each with an approved plan and a first
phase run under their own uid, credentials and subscription.

## 6. Per-phase recipe for the orchestrator

The header to paste, filled per phase (the order of work is the skill, not this file):

```
Order of work: read /home/john/Documents/Work2026/MorphV2/.claude/skills/morph-v2-orchestrator/SKILL.md whole and follow it.
Project: /home/john/Documents/Work2026/MorphStudio (morphd, Go 1.25). Phase: <P9a…P12>. Cut from main @ <hash>.
Morph CLI: ~/bin/morphv2 (binary /tmp/morphbin at MorphV2 <commit>). Processor: ds (maxTokens x3 after the cut).
Go: ~/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.25.0.linux-amd64/bin first on PATH (/usr/bin/go is 1.22).
Constraints: cap $5; one run + at most one fix; data only (record inside this phase's Functions, map, TASK,
decks/<P>/*, fixtures); PLAN.md and PLAN_ONBOARDING_VPS.md frozen; no merge, no push, no VPS; explicit git paths.
Phase text: the row <P> of PLAN_ONBOARDING_VPS.md §5 + its stop.
Live reproduction: <the row's smoke, or the $0 laptop part of it>.
```

Rules that keep phases fast (paid for in P7b–P8b):
1. New Functions get their own files; a phase is ≤ 12 cards.
2. A re-cut names in the record every symbol of its files used outside the subset, private ones too.
3. Judges that list routes or tool names are re-cut with the code that changes the list.
4. A Component stays ≤ 30 KB: split before it grows (P9b splits `spawn` off daemon first).
5. Probes are red per example on stubs before the gate; fixtures are recorded from probes, never invented.
6. The orchestrator never polls the run; the PM merges, records MEASURE / DECISIONS and runs the stop's smoke.

## 7. Record additions

- Guardrail **Tenant Isolation**: every claude spawn goes through Sandbox Argv under the project's user; no package
  but `claude` builds a process argv.
- Guardrail **Secrets Never Logged**, widened: claude tokens, API keys, PATs and one-time links never appear in an
  argv, a log, a reply, a milestone or an error.
- Requirement **One Chat One User**: a Telegram chat maps to at most one user; an unknown chat gets only `/start`.
- Requirement **Per-User Tokens**: every HTTP/MCP token belongs to one user; only the operator's admin token sees all
  projects.
- Layers: `users` imports nothing of the module; `tgbot` → control, users, stream; `onboard` → users, github, runner;
  `spawn` → claude, users. A new package is a plan change.

## 8. Deferred epic E8 — a VPS per user (operator 10.10: not now)

Shape: an always-on small **studio-bot** (Telegram, onboarding, the remote, hcloud with the operator's token) and a
VPS per user created from that user's latest snapshot; morphd on the user's VPS says when sleeping is safe (a stop,
everything pushed, no session); sleep = snapshot → delete; wake = create from snapshot → wait for `/healthz`.

Facts to design against: a stopped Hetzner server still bills — only snapshot + delete saves money; a wake from a
≈ 30 GB snapshot takes minutes; a re-created server gets a new IP (keep a Primary IP or reach it only through the bot);
the snapshot holds the user's secrets; hcloud gotchas seen 04–05.10: `--ssh-key` by id (113295515), a type missing in
a location (loop fsn1 / nbg1 / hel1), `datacenter list` deprecated, `-o format` `.Prices` fails; existing snapshots
`morph-stand-20260923/-0930`; the current VPS 188.245.28.11 is hcloud `ethsc-watch2` (cx23, nbg1).

Sketch phases (only on the operator's word): H1 hcloud client (injected `Do`: create from snapshot, snapshot, delete,
wait for the action); H2 lifecycle policy (sleep on a stop + idle N minutes, wake on a command); H3 the bot as the
control plane over many morphd.

## 9. Open questions (recommended default first)

1. A second claude credential for probe A — **default:** run probe A on one token and note that per-tenant limits
   stay unproven until the first tester's token.
2. The bot — **default:** a new bot for the beta; "Checker" stays the operator's milestone channel.
3. The VPS — **default:** rescale 188.245.28.11 (it already runs morphd, Caddy and the DNS name) to ≥ 4 vCPU / 16 GB.
4. Anthropic's terms for a subscription token used by a server on the user's behalf — check before smoke T3
   (SKETCH §7.2); the API-key path (D5) is the fallback.
5. Who creates Linux users — **default:** the operator with `tools/add-tenant.sh`; morphd stays unprivileged.
6. After D2's change (11.10): where does a tester plan a new project — **default:** in their own Claude Code on the
   laptop in native plan mode, then the plan is committed and loaded with `plan_load` over MCP; P12 (a plan session
   relayed through the bot, probe B) shrinks to optional. How does the per-user MCP token reach the laptop — **default:**
   the onboarding form shows the `claude mcp add --transport http morph https://themorph.stream/mcp/<p> --header …`
   line once. Which bot commands stay in P10a — **default:** `/start <code>`, `/status`, `/help` and answering a
   session question by a button; orders, continue and approve go over MCP.

## 10. Out of scope

A web UI beyond the one secret form; billing; a VPS per user (E8); GitHub App / OAuth device flow (later);
containers (rootless podman, later); more than 3 tenants; the PM flow over MCP (stays as it is, unused by testers).
