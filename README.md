# MorphStudio — morphd

`morphd` is a Go daemon on the user's VPS that drives the claude CLI session of a Morph-built project, in place of the
PM's ssh + tmux flow. The PM (Claude Code on the operator's laptop) talks to it over a remote MCP endpoint; the operator
over an HTTP API with a bearer token. One user = one VPS = one morphd; many projects per user.

What it does: starts a fresh claude session per phase (`/morph-orchestrator <phase>`, auto-memory off), reads its
stream-json, answers its questions through the PM, queues orders while it is busy, checks the git state at the end of
a phase, walks the approved phase queue between stops, keeps a JSONL event log per session and posts milestones to
Telegram. Not production: no TLS (Caddy in front), no deploy unit, no browser UI, no multi-user, no provisioning.

## How it is built

By [MorphV2](https://github.com/VasyaLutiy/morph) cards, not by hand: the record `contour.yaml` (17 Components,
27 Functions, 199 examples) and `morph-map.json` are cut into 7 phases (P1–P7, 54 cards) in `PLAN.md`. Every code file
and every example test is written by an executor model and accepted by the record's examples. The phases run
autonomously on the VPS by `docs/AUTONOMY.md`; measurements in `docs/MEASURE.md`, decisions in `docs/DECISIONS.md`.

| where | what |
|---|---|
| `PLAN.md`, `contour.yaml`, `morph-map.json` | the plan, the record, the map |
| `docs/START.md`, `docs/DECISIONS.md` | the brief and the operator's decisions |
| `docs/deps/go-sdk.md` | the digest of the one dependency, the official Go MCP SDK v1.8.0 (vendored in `vendor/`) |
| `internal/testhelp`, `tests/fixtures/` | test helpers and fixtures (the measured claude stream-json probe) |
| `decks/tools/` | the Go guard, the layer table, the token scaler |
| `.morph/runs/` | run archives (the first one is the P15L gocrud skeleton this repository started from) |

```
export GOFLAGS=-mod=vendor GOPROXY=off
go build ./... && go vet ./... && go test -count=1 ./...
```

Go 1.25. Stdlib only, plus `github.com/modelcontextprotocol/go-sdk`.
