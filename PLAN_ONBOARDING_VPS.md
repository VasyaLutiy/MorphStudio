# PLAN_ONBOARDING_VPS.md — beta onboarding on one multi-user VPS (epic E7; E8 deferred)

> **Status: DRAFT 11.10, not approved.** Designed by Claude Code in plan mode with the operator (no PM flow, no Fable
> architect). Gate 0 (probes A–E) is done and green; probe F waits for P12. `PLAN.md` stays frozen until this draft is approved by commit
> hash; then the record (`contour.yaml`, `morph-map.json`) gains the Components below phase by phase and `PLAN.md` is
> rewritten whole with epic E7 (a plan change rewrites, never appends). Decisions: `docs/DECISIONS.md` 10.10–11.10.
> Base: morphd P0–P8b built, Smoke stop 3 green (51f9686), 237 example tests, morphd behind the Cloudflare Tunnel.

## 1. Decisions (operator 10.10–11.10) — binding

| # | decision |
|---|---|
| D1 | Beta: **3 testers, ONE VPS, ONE morphd**. A VPS per user is deferred (§9). |
| D2 | **Two surfaces.** Each tester works from **their own laptop with the claude CLI connected to morphd over MCP** (`https://themorph.stream/mcp` for the registry, `/mcp/<project>` per project, a per-user token). The **Telegram bot** carries onboarding and alerts (milestones, stops, a session question answered by a button); a chat with the orchestrator may come later. Cloudflare Access service tokens (one per laptop) — later, not in E7. |
| D3 | **Secrets never in chat.** The bot sends a one-time link (TTL 10 min, one use) to one HTTPS form served by morphd through the Cloudflare Tunnel. |
| D4 | **Isolation = one Linux user per tester** (`mu-<name>`, home 0700). Every process of a tester — the claude session and every git / morph command morphd runs for the tester's projects — runs under that uid through the root wrappers of `tools/` (sudoers allows morphd only those). morphd's own state stays 0700 under `morph`. Containers later. |
| D5 | **Credentials per tester**: claude only via `ANTHROPIC_API_KEY` (a Console key with a chosen expiry). **No subscription OAuth tokens** (`CLAUDE_CODE_OAUTH_TOKEN` / `claude setup-token`, nor any other provider's subscription token) — operator 11.10, terms risk; kept by morphd under its state (0600) and handed to the session only as the systemd unit's `EnvironmentFile`. GitHub: a fine-grained PAT per tester; the tester's own git needs it, so it lives in the tester's home (`~/.git-credentials`, 0600, the tester's). Never in an argv, a log, a reply, a milestone or an error. The tester creates the empty repo. |
| D6 | One session per tester, ≤ 3 in total (max-parallel 3, per-user 1). An API-key tester has no subscription windows (`five_hour` / `seven_day` stay null); spend is bounded by the per-phase cap and the key's own Console limit. |
| D7 | **No Postgres.** JSON files behind a `Store` interface; pure-Go SQLite (`modernc.org/sqlite`, vendored) only when queries or history are needed. |
| D8 | **Planning happens on the tester's laptop** in Claude Code's native plan mode (the way the operator liked it), PLAN.md by the MorphV2 template; the tester commits and pushes it and loads it with `plan_load` over MCP. Plan mode from the phone is a mode of the optional Chat relay (P12, probe B). |

## 2. Target picture

```
tester's laptop                      Cloudflare            one VPS (≥ 4 vCPU, 16 GB)
┌───────────────────────┐   HTTPS   ┌────────┐  tunnel  ┌─────────────────────────────────────────────────────┐
│ claude CLI            │ ────────▶ │ themorph│ ───────▶ │ Caddy 127.0.0.1:7190 (session MCP 404, Host fix)    │
│  mcp "morph"  /mcp     │           │ .stream │          │  └ morphd 127.0.0.1:7180 (User=morph, state 0700)    │
│  mcp "<p>"  /mcp/<p>   │           └────────┘          │     ├ users: users.json, tokens, secrets 0600        │
│  plan mode → PLAN.md   │                                 │     ├ onboard: invite, one-time form, checks         │
│  git push → plan_load  │                                 │     ├ tgbot: getUpdates long poll, alerts, buttons   │
└───────────────────────┘                                 │     ├ supervisor/daemon/pump (built)                 │
tester's phone                                             │     └ sudo -n tools/morph-spawn | morph-run          │
┌───────────────────────┐   Bot API (long poll, no inbound)│          ├ /home/mu-alice/projects/<p> (claude, git) │
│ Telegram @MorphX00_bot │ ◀──────────────────────────────▶│          ├ /home/mu-bob/projects/<p>                 │
│ /start <code>, alerts  │                                 │          └ /home/mu-carol/projects/<p>               │
└───────────────────────┘                                 └─────────────────────────────────────────────────────┘
```

## 3. Gate 0 — results (11.10, all green; details in DECISIONS)

| probe | result | what the record takes from it |
|---|---|---|
| **A. two testers at once** | two `systemd-run --uid` sessions in parallel, each with its own OAuth token (probe only; the product takes API keys, D5) and `CLAUDE_CONFIG_DIR`, 7 s, $0.0037; other homes and `/home/morph` invisible (`ProtectHome=tmpfs` + `BindPaths`), `/tmp` private, separate limit windows per account | the claude binary is a shared root copy `/opt/claude/claude`; logs `docs/probe/probe-tenant-{a,b}.jsonl` → fixture `tests/fixtures/stream/probe-tenant.jsonl` (P9b) |
| **B. headless plan mode** | without skip-permissions `--permission-mode plan` works: the plan file in `$CLAUDE_CONFIG_DIR/plans/`, `ExitPlanMode` as `can_use_tool {plan, planFilePath}`, deny = revise with the message, allow = approve; with skip-permissions plan mode is overridden | kept for P12 (optional); `docs/probe/probe-planmode.jsonl` |
| **C. Bot API** | @MorphX00_bot: getUpdates with offset, inline keyboard, callback_query, answerCallbackQuery, edit, delete, 401 | fixtures `tests/fixtures/tg/*.json` (scrubbed); a second press must be refused by the bot; answer every callback at once (a late answer → 400 "query is too old") |
| **D. privilege** | polkit refuses morph's `systemd-run`; ambient capabilities do not help; a root wrapper + one sudoers line works; SIGTERM to the process group ends the unit | `tools/morph-spawn` (installed, `docs/RUNBOOK_VPS.md`): every systemd property is hand data; a card builds only the wrapper's argv |
| **E. tester runner** | `tools/morph-run` (installed; sudoers now allows morph-spawn and morph-run): as morph, `git clone` / `rev-parse` / `fetch` / `config` in `/home/mu-a/projects` run as mu-a (files owned by mu-a); `--git-credentials <file>` installs the tester's `~/.git-credentials` (mu-a 0600, unreadable to morph) and a private repo clones; `morph init` runs from the shared copy `/opt/morph` (100 MB, root, read-only); refusals: a command outside {git, morph}, a dir outside the home, a bad tester, a direct `sudo git` | `runner.Runner` for a tester's project = `sudo -n /usr/local/sbin/morph-run <t> <dir> -- <git\|morph> …` (no stdin needed: the PAT goes by `--git-credentials`); git's own `-c` options run as the tester only — no more than morph-spawn already allows |
| **F. chat relay extras** — *to do before P12* | @MorphX00_bot: `sendChatAction typing` while a turn runs; a long reply split at 4096 chars with HTML escaping; a photo from the chat (`getFile` → bytes) passed to claude as an image content block in a stream-json `user` line; `--include-partial-messages` + `editMessageText` throttled to ≤ 1 edit/s | the Bot API calls 200; claude answers about the image; the edit rate stays under Telegram's limit; $0 Bot API + cents of claude | fixtures `tests/fixtures/tg/chat-*.json`, `tests/fixtures/stream/probe-image.jsonl` |

## 4. Hand data and ops (never cards)

- VPS: ≥ 4 vCPU, 16 GB (three claude + `go test` do not fit cx23's 2 vCPU / 4 GB) — rescale 188.245.28.11 (§10.2).
- `tools/morph-spawn` (done) and `tools/morph-run` (probe E), root 0755 in `/usr/local/sbin/`; `/etc/sudoers.d/morph-spawn`
  allows morphd exactly these two.
- `tools/add-tenant.sh <name>`: `useradd -m mu-<name>`, home 0700, `~/projects/`, git identity, shared read-only
  toolchains on `PATH` (Go 1.25, node, `/opt/claude/claude`, the morph binary copy); the operator runs it (root);
  morphd never creates users.
- The bot: re-issue @MorphX00_bot's token (it was in the PM chat) and put it in morphd's `.env` as `TG_BOT_TOKEN`;
  `TG_CHAT_ID` becomes the operator's admin chat only.
- Cloudflare Tunnel and Caddy as in `docs/RUNBOOK_VPS.md` (everything reaches morphd; `/onboard/*` needs no change).
- Backups of `~/morphd/state` (users, secrets, project state) — a nightly tar 0600 to the operator.
- The beta tester list (3 names, Telegram handles) — the operator.

## 5. Epic E7 — phases

Each row is one orchestrator phase (one run + at most one fix, cap $5). Card counts are estimates for `morph plan
--judge`; the cut decides. "Stop" = a smoke by the PM/operator before the next phase.

| phase | Component (new / re-cut) | Functions | cards (est.) | depends | oracle and fixtures | by hand before the run | $ est. | risk → treatment | stop after |
|---|---|---|---|---|---|---|---|---|---|
| **P9a** | **new `users`**; re-cut `registry · Project Registry` | User Registry (users.json: name, Linux user, chat id, created; the MCP/API token stored as a SHA-256 hash); User Secrets (kinds `anthropic-key`, `github-token`; files 0600 under the state dir; never returned); Token Lookup (bearer → user, constant-time compare, the operator's admin token → all); Project Registry gains `User` | 8 | Gate 0 | `tests/fixtures/users/users-3.json`; literal 0600 modes; `users: ` error texts | TASK, checks, probes, fixture | 0.20 | a secret in an error text → a Guardrail example per kind | — |
| **P9b** | **new `spawn`** (split off package daemon: `daemon/spawn.go`; daemon is 27.7 KB); re-cut `claude · Launch Args`, `daemon · Daemon Core` | Tester Argv (pure: `sudo -n /usr/local/sbin/morph-spawn <t> <dir> <env-file> [--mcp-config <f>] -- <claude args>`); Tester Runner (a `runner.Runner` that prefixes `sudo -n /usr/local/sbin/morph-run <t> <dir> --`); Launch Args (`Launch.Tester`; for a tester project no `--mcp-config` path of morph's own, the env file instead of the process env); Daemon Core spawns and runs git for a tester's project through them | 10 | P9a, probe E | the wrapper argv as literals; `probe-tenant.jsonl` lines | TASK, checks (freeze all but claude, daemon), `_stubs`, fixture | 0.35 | cross-file daemon re-cut → subset transaction; every private symbol used outside the subset named in the record (P8b) | **smoke T1**: two testers' one-step phases (S3-like) in parallel on the VPS, each under its own uid and token; isolation checks of probe A hold |
| **P9c** | re-cut `api · Router, HTTP Handlers`, `mcpserver · MCP Mount, PM Tools`, `control` | per-user auth on HTTP and MCP (a token → its user; another user's project → 404 `unknown_project`; the admin token sees all); `projects_list` only the caller's projects; `project_create` / `project_adopt` create or adopt under `/home/mu-<t>/projects/<p>` through the Tester Runner | 10 | P9b | 401 / 404 bodies; two users' tokens in examples | TASK, checks, `_stubs` | 0.30 | judges that list routes or tools → re-cut with the code (P8b) | **smoke T2**: from two laptops, each `claude mcp add` with its own token sees only its projects; `project_create`, a plan pushed from the laptop, `plan_load`, a phase runs |
| **P10a** | **new `tgbot`** (pure) | Update Parse (getUpdates → message / command / callback); Command Route (`/start <code>`, `/status`, `/help`; an unknown chat gets only `/start`); Keyboard (a pending question → `inline_keyboard`; callback data `ans:<p>:<req8>:<n>` → `answer {option}`; a second press → "already answered") | 6 | P9c | `tests/fixtures/tg/*.json` | TASK, checks | 0.20 | Bot API drift → every example cites a fixture | — |
| **P10b** | **new** Poll Loop (in `tgbot`); re-cut `telegram · Milestone Post`, `pump`, `cmd · Config And Main` | Poll Loop (getUpdates, `offset = last + 1`, 30 s long poll, answers every callback before any slow work, injected `Do` and clock, backoff on 409 / 5xx); milestones to the chat of the project's user (system lines to the admin chat); the pump's `ask` post carries the buttons; Config wires the bot | 8 | P10a | recorded updates; exact sendMessage bodies | TASK, checks, `_stubs` | 0.30 | pump re-cut (186 KB judge slice) → old pump examples kept, new ones added | **smoke T3**: a tester's alerts arrive in their chat; a session question answered by a button |
| **P11** | **new `onboard`** | Invite (operator `/invite <name>` → a one-time code; `/start <code>` binds the chat); Secret Form (one-time link, TTL 10 min, one use; claude token or API key, GitHub PAT; HTML-escaped; stores via `users` and writes the PAT into the tester's home through `morph-run`; at the end shows once the `claude mcp add --transport http morph https://themorph.stream/mcp --header "Authorization: Bearer …"` line); Credential Check (a PONG under the tester's sandbox, haiku, `--max-budget-usd 0.01`; GitHub access via `github`); Checklist (chat, claude, GitHub, TG test → ready, one bot message) | 8 | P10b | the form HTML as a literal; probe A's PONG lines; github 200 / 401 / 404 bodies | TASK, checks | 0.30 | replay / CSRF → one-use + TTL examples; a token in a log → Guardrail example | **smoke T4 (E7 done)**: a real tester from phone + laptop: invite → form → checklist ✅ → `claude mcp add` → `project_create` → plan in plan mode on the laptop → `plan_load` → a phase runs → alerts in Telegram |
| **P12** *(optional, after T4)* | **new** Chat Relay (in `tgbot`); re-cut `daemon · Daemon Core` (session kind `chat`), `session · Session Machine` (ExitPlanMode as a pending `plan`), `tgbot · Keyboard` | Chat Session (a tester's free chat with their own claude in their sandbox: spawn on the first message through `morph-spawn`, `--resume` the same session id on the next ones, killed after N idle minutes, `/new` starts a fresh one); Relay Out (assistant text and `result` → messages ≤ 4096 chars, HTML-escaped, `typing` while busy, optional live edit ≤ 1/s); Relay In (text → a `user` line; a photo → an image block); Mode Switch (while a phase of this tester runs, the chat talks to the phase session — `order` / `answer`; otherwise to the chat session); Plan Mode (`/plan <idea>`: a chat session without skip-permissions, `--permission-mode plan`; ExitPlanMode → plan text + Approve / Revise, Revise = deny with the tester's text; Approve → the plan committed and `plan_load`) | 10–12 | P11, probes B and F | `probe-planmode.jsonl`, chat fixtures of probe F, the probe log lines | TASK, checks, `_stubs`, fixtures | 0.40 | §8.6–8.10 | **smoke T5**: a tester chats with their claude from the phone (a question with buttons, a photo, `/new`); during a running phase the chat reaches the phase session; `/plan` produces an approved plan that loads |

**Totals (P9a–P11)**: 6 phases, ≈ 50 cards; executor ≈ $1.65 on `ds` (cap $5 per phase); claude for the smokes ≤ $1.
**Done** when smoke T4 is green for one real tester; the other two are onboarded the same way.

## 6. Per-phase recipe for the orchestrator

The header to paste, filled per phase (the order of work is the skill, not this file):

```
Order of work: read /home/john/Documents/Work2026/MorphV2/.claude/skills/morph-v2-orchestrator/SKILL.md whole and follow it.
Project: /home/john/Documents/Work2026/MorphStudio (morphd, Go 1.25). Phase: <P9a…P11>. Cut from main @ <hash>.
Morph CLI: ~/bin/morphv2 (binary /tmp/morphbin at MorphV2 <commit>). Processor: ds (maxTokens x3 after the cut).
Go: ~/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.25.0.linux-amd64/bin first on PATH (/usr/bin/go is 1.22).
Constraints: cap $5; one run + at most one fix; data only (record inside this phase's Functions, map, TASK,
decks/<P>/*, fixtures); PLAN.md and PLAN_ONBOARDING_VPS.md frozen; no merge, no push, no VPS; explicit git paths.
Phase text: the row <P> of PLAN_ONBOARDING_VPS.md §5 and §7.
Live reproduction: the $0 laptop part of the row's smoke (morphd on a local port, fake claude/wrappers); the VPS smoke is the PM's.
```

Rules that keep phases fast (paid for in P7b–P8b):
1. New Functions get their own files; a phase is ≤ 12 cards.
2. A re-cut names in the record every symbol of its files used outside the subset, private ones too.
3. Judges that list routes or tool names are re-cut with the code that changes the list.
4. A Component stays ≤ 30 KB: split before it grows (P9b splits `spawn` off daemon first).
5. Probes are red per example on stubs before the gate; fixtures are recorded from Gate 0, never invented.
6. The wrappers are hand data: a card builds their argv, never their systemd properties; tests never call `sudo`
   (the Tester Runner is tested over `runner.Fake`).
7. The orchestrator never polls the run; the PM merges, records MEASURE / DECISIONS and runs the stop's smoke.

## 7. Record additions

- Guardrail **Tester Isolation**: every claude session and every git / morph command of a tester's project goes
  through `morph-spawn` / `morph-run` under the project's user; only package `claude` builds a process argv; no package
  runs `sudo` with anything else.
- Guardrail **Secrets Never Logged**, widened: claude tokens, API keys, PATs, MCP tokens and one-time links never
  appear in an argv, a log, a reply, a milestone or an error.
- Requirement **Per-User Tokens**: every HTTP / MCP token belongs to one user and sees only that user's projects; only
  the operator's admin token sees all.
- Requirement **One Chat One User**: a Telegram chat maps to at most one user; an unknown chat gets only `/start`.
- Layers: `users` imports nothing of the module; `onboard` → users, github, runner; `tgbot` → control, users, stream;
  `spawn` → claude, runner, users. A new package is a plan change.

## 8. Known risks

1. A leaked tester API key: revocable in the Console and set with an expiry; subscription tokens (`setup-token`, no
   documented revocation) are excluded by D5.
2. Subscription tokens used by a server on the user's behalf — removed (D5, operator 11.10); API keys only.
3. A tester's session can read the tester's own PAT (it needs it to push) — acceptable: it is their PAT, scoped to
   their repos.
4. Cloudflare's 100 s limit per HTTP response — every MCP tool must answer quickly (all do today).
5. D5 of smoke 3 (stretch counted twice after a plan's end) is still open — fix on the operator's word (2 cards).
6. **Chat relay vs one session per tester (D6).** A free chat while a phase runs would be a second claude of the
   tester. Default: during a phase the chat talks to the phase session (`order` / `answer`); outside phases a free
   chat session; raising the limit to 2 sessions per tester only on the operator's word.
7. **Permissions in the chat.** With skip-permissions the chat may do anything — but only inside the tester's sandbox
   and uid. Default for the beta: skip-permissions in a free chat; a cautious mode (writes and commands as buttons, the
   probe B mechanism) as a later option. Plan mode never uses skip-permissions (probe B).
8. **Anthropic's terms for the relay**: a chat relay is "Claude offered through our bot" on the tester's own API key;
   check the commercial terms for that before the pilot.
9. **Telegram in the middle**: code, answers and whatever the tester pastes pass through Telegram's servers — the
   tester is told so at onboarding; secrets still only through the one-time form.
10. **A hijacked Telegram account** reaches the tester's claude and repos. Beta: One Chat One User is enough; later a
   PIN or a confirmation for destructive actions.

## 9. Deferred epic E8 — a VPS per user (operator 10.10: not now)

Shape: an always-on small **studio-bot** (Telegram, onboarding, hcloud with the operator's token) and a VPS per user
created from that user's latest snapshot; morphd says when sleeping is safe (a stop, everything pushed, no session);
sleep = snapshot → delete; wake = create from snapshot → wait for `/healthz`. Facts: a stopped Hetzner server still
bills; a wake from a ≈ 30 GB snapshot takes minutes; a re-created server gets a new IP (the tunnel makes that
harmless); the snapshot holds the user's secrets; hcloud gotchas: `--ssh-key` by id (113295515), a type missing in a
location (loop fsn1 / nbg1 / hel1), `datacenter list` deprecated, `.Prices` template fails. Sketch phases: H1 hcloud
client, H2 lifecycle policy, H3 the bot as the control plane over many morphd.

## 10. Open questions (recommended default first)

1. Who creates Linux users — **default:** the operator with `tools/add-tenant.sh`; morphd stays unprivileged.
2. The VPS — **default:** rescale 188.245.28.11 to ≥ 4 vCPU / 16 GB before smoke T1.
3. Which claude credential testers use — **decided 11.10:** an API key with an expiry only (D5); no subscription
   OAuth tokens.
4. Approve this draft by hash before P9a.

## 11. Out of scope

A web UI beyond the one secret form; billing; a VPS per user (E8); GitHub App / OAuth device flow (later);
containers (rootless podman, later); Cloudflare Access (later); more than 3 testers; the old PM flow.
