# Primer: p9a-claude

generated 2026-10-12T08:26:20.253Z · 712 files in the tree · no model call, no network
missing: docs/PLAN.md

## Tests

- 237 tests in 30 test files by the go profile (counted from text, not a run)

## Runs

- archived runs: 16 (V2 16, mrph 0), 2026-10-08 → 2026-10-10
- cards: 88 written of 138 (37 failed, 0 skipped); requests 208, answers kept 206
- cost: $4.0024 over 16 priced runs (0 unpriced)
- by format: V2 16 runs, 88/138 written, $4.0024; mrph 0 runs, 0/0 written, $0.0000
- models: deepseek/deepseek-v4.1-flash (16 runs)
- debt rows (docs/MEASURE.md): none
- running total (docs/MEASURE.md): "Running total: ≈ $4.94 of $35 executor + the unknown cost of the interrupted run 20261010-205103" vs $4.0024 archived here, difference 0.9376 — the two differ by runs made outside this repository (in MEASURE, no archive here) and archived runs MEASURE's total leaves out; debt rows are in neither

## Chronology (docs/MEASURE.md)

- no rows

## File ownership (git, Morph-Card trailers)

- git carries 101 Morph commits: deepseek/deepseek-v4.1-flash 100, claude-fable-5-1 1
- 71 paths written by cards, most recent first; per path its cards, newest first:
- cmd/morphd/config_examples_test.go ← config-and-main-judge.r2 (deepseek/deepseek-v4.1-flash, run 20261010-212705); config-and-main-judge (deepseek/deepseek-v4.1-flash, run —); config-and-main-judge (deepseek/deepseek-v4.1-flash, run 20261008-211009)
- daemon/pump_examples_test.go ← pump-judge.r1 (deepseek/deepseek-v4.1-flash, run 20261010-212705); pump-judge (deepseek/deepseek-v4.1-flash, run —); pump-judge (deepseek/deepseek-v4.1-flash, run 20261008-211009)
- daemon/daemon_examples_test.go ← daemon-core-judge.r2 (deepseek/deepseek-v4.1-flash, run 20261010-212705); daemon-core-judge (deepseek/deepseek-v4.1-flash, run —); daemon-core-judge.r1 (deepseek/deepseek-v4.1-flash, run 20261008-211009)
- cmd/morphd/config.go ← config-and-main.r1 (deepseek/deepseek-v4.1-flash, run 20261010-212705); config-and-main (deepseek/deepseek-v4.1-flash, run —); config-and-main.r1 (deepseek/deepseek-v4.1-flash, run 20261008-211009)
- cmd/morphd/main.go ← config-and-main.r1 (deepseek/deepseek-v4.1-flash, run 20261010-212705); config-and-main (deepseek/deepseek-v4.1-flash, run —); config-and-main.r1 (deepseek/deepseek-v4.1-flash, run 20261008-211009)
- daemon/pump.go ← pump (deepseek/deepseek-v4.1-flash, run 20261010-212705); pump (deepseek/deepseek-v4.1-flash, run —); pump (deepseek/deepseek-v4.1-flash, run 20261008-211009)
- mcpserver/pm_examples_test.go ← pm-tools-judge (deepseek/deepseek-v4.1-flash, run 20261010-212705); pm-tools-judge (claude-fable-5-1, run —)
- mcpserver/mount_examples_test.go ← mcp-mount-judge (deepseek/deepseek-v4.1-flash, run 20261010-212705); mcp-mount-judge (deepseek/deepseek-v4.1-flash, run 20261008-200807)
- mcpserver/pm.go ← pm-tools (deepseek/deepseek-v4.1-flash, run 20261010-212705); pm-tools (deepseek/deepseek-v4.1-flash, run 20261008-200807)
- daemon/daemon.go ← daemon-core.r1 (deepseek/deepseek-v4.1-flash, run 20261010-212705); daemon-core (deepseek/deepseek-v4.1-flash, run —); daemon-core (deepseek/deepseek-v4.1-flash, run 20261008-211009)
- api/router_examples_test.go ← router-judge.r1 (deepseek/deepseek-v4.1-flash, run 20261010-183047); router-judge.r1 (deepseek/deepseek-v4.1-flash, run 20261008-193337); router-judge (deepseek/deepseek-v4.1-flash, run 20261008-083312)
- api/router.go ← router (deepseek/deepseek-v4.1-flash, run 20261010-183047); router (deepseek/deepseek-v4.1-flash, run 20261008-193337); router (deepseek/deepseek-v4.1-flash, run 20261008-083312)
- api/handlers_examples_test.go ← http-handlers-judge.r1 (deepseek/deepseek-v4.1-flash, run 20261010-183047); http-handlers-judge (deepseek/deepseek-v4.1-flash, run 20261008-193337); project-handlers-judge (deepseek/deepseek-v4.1-flash, run 20261008-083312)
- bootstrap/adopt_examples_test.go ← project-adopt-judge (deepseek/deepseek-v4.1-flash, run 20261010-183047)
- api/handlers.go ← http-handlers (deepseek/deepseek-v4.1-flash, run 20261010-183047); http-handlers (deepseek/deepseek-v4.1-flash, run 20261008-193337); project-handlers (deepseek/deepseek-v4.1-flash, run 20261008-083312)
- control/events_examples_test.go ← event-view-judge (deepseek/deepseek-v4.1-flash, run 20261010-183047)
- control/adopt_examples_test.go ← adopt-contract-judge.r1 (deepseek/deepseek-v4.1-flash, run 20261010-183047)
- bootstrap/adopt.go ← project-adopt (deepseek/deepseek-v4.1-flash, run 20261010-183047)
- control/events.go ← event-view (deepseek/deepseek-v4.1-flash, run 20261010-183047)
- control/adopt.go ← adopt-contract (deepseek/deepseek-v4.1-flash, run 20261010-183047)
- supervisor/loop_examples_test.go ← phase-loop-judge (deepseek/deepseek-v4.1-flash, run 20261010-165924); phase-loop-judge (deepseek/deepseek-v4.1-flash, run —); phase-loop-judge (deepseek/deepseek-v4.1-flash, run 20261008-193337)
- supervisor/loop.go ← phase-loop.r1 (deepseek/deepseek-v4.1-flash, run 20261010-165924); phase-loop (deepseek/deepseek-v4.1-flash, run —); phase-loop (deepseek/deepseek-v4.1-flash, run 20261008-193337)
- supervisor/guard_examples_test.go ← runtime-guard-judge (deepseek/deepseek-v4.1-flash, run —); runtime-guard-judge.r1 (deepseek/deepseek-v4.1-flash, run 20261008-193337)
- supervisor/guard.go ← runtime-guard (deepseek/deepseek-v4.1-flash, run —); runtime-guard (deepseek/deepseek-v4.1-flash, run 20261008-193337)
- control/control_examples_test.go ← control-contract-judge (deepseek/deepseek-v4.1-flash, run —); control-contract-judge.r1 (deepseek/deepseek-v4.1-flash, run 20261008-185347)
- control/control.go ← control-contract (deepseek/deepseek-v4.1-flash, run —); control-contract (deepseek/deepseek-v4.1-flash, run 20261008-185347)
- mcpserver/session_examples_test.go ← session-tools-judge (deepseek/deepseek-v4.1-flash, run 20261008-200807)
- mcpserver/mount.go ← mcp-mount (deepseek/deepseek-v4.1-flash, run 20261008-200807)
- mcpserver/session.go ← session-tools (deepseek/deepseek-v4.1-flash, run 20261008-200807)
- gitrules/end_examples_test.go ← phase-end-check-judge.r1 (deepseek/deepseek-v4.1-flash, run 20261008-185347)
- … 41 more paths

## What is next

- next phase: none open (docs/PLAN.md)
- handoff (docs/AUTONOMY.md):
  > State at handoff
  > **Next: Smoke stop 3** (PLAN "Smoke stop 3": the PM over the systemd morphd on the VPS, ≤ $0.30 of claude), after the
  > VPS morphd is rebuilt from main (RUNBOOK_VPS "Update after a merge"); then the PM removes the morph user's
  > `~/.config/morph/tg.env` (Q28). No phase remains: a green smoke 3 closes the stretch (`end` 🎉); a red one → a
  > DECISIONS line and an issue labelled for the Component.

## Last decisions (docs/DECISIONS.md)

- 11.10 · VPS · morphd moved behind the Cloudflare Tunnel `morphd` (operator: the domain is on Cloudflare): cloudflared with a token file, Caddy reduced to a localhost-only origin on :7190 (session MCP 404, Host rewrite kept), the A record replaced by the tunnel's CNAME; only ssh listens publicly; de…
- 11.10 · product direction · D2 changed (operator): testers connect their own laptops with the claude CLI to morphd over MCP (per-user tokens, through the Cloudflare Tunnel); the Telegram bot stays for onboarding, alerts and maybe a chat with the orchestrator — it is no longer the only remote. Cloud…
- 11.10 · Gate 0 · probe D GREEN (privilege, by the PM on the VPS, claude $0.0012): (1) morph alone → `systemd-run --uid` refused by polkit; (2) ambient `CAP_SETUID CAP_SETGID` → still refused for systemd-run (only a bare `setpriv` works, without cgroup limits or sandbox) — rejected; (3) chosen: a ro…
- 11.10 · Gate 0 · probe E GREEN (tester runner, by the PM on the VPS, $0): `tools/morph-run` installed as `/usr/local/sbin/morph-run`, sudoers now `morph ALL=(root) NOPASSWD: /usr/local/sbin/morph-spawn, /usr/local/sbin/morph-run`; a shared read-only morph copy `/opt/morph` (100 MB). As morph throug…
- 11.10 · product direction · **no subscription OAuth tokens** (operator): morphd never takes a user's subscription token — neither `CLAUDE_CODE_OAUTH_TOKEN` (`claude setup-token`) nor any other provider's coding-plan token (z.ai and the like); terms risk, a red flag for investors. Users bring an API…

## Open issues (.morph/issues.json)

- not read: no .morph/issues.json (the primer makes no network call; `gh issue list --state open --json number,title,labels > .morph/issues.json` writes it)
