# MorphStudio

The future web UI and backend of a Morph orchestrator SaaS. This repository starts with its first piece: a
**skeleton of the backend API in Go**. The code was written by [MorphV2](https://github.com/VasyaLutiy/morph), not
by hand. It is also MorphV2's first live check of its Go language profile (task `docs/TASK_P15L_gocrud.md` in MorphV2).

## What is here

A RESTful CRUD API for `Project` built on the Go standard library only (Go 1.22, `go.mod` requires nothing):

| package | what |
|---|---|
| `project` | `Input`/`Project`, input validation (name, slug, description, status), `New`, `Apply` |
| `store` | the `Store` interface, an in-memory store and a JSON-file store (atomic rewrite) |
| `auth` | HTTP Basic auth: salted SHA-256 hashes, constant-time check, middleware |
| `api` | handlers for `GET/POST /v1/projects` and `GET/PUT/DELETE /v1/projects/{id}`, the router, `GET /healthz` |

This is a skeleton, not production. It has no database driver, no `cmd/server`, no users CRUD, no TLS and no deploy.

## How it was built

- The data came first: a Contour record (`contour.yaml`), a map, checks and probes. Then `morph plan` cut a deck of 14
  cards in 5 generations (7 code cards, 7 judge cards that write the example tests).
- `morph run` executed the deck on `deepseek/deepseek-v4.1-flash`. Every acceptance ran in this order: `go build` +
  `gofmt -l`, `go vet`, a layer guard, a probe, the package's tests, then `go test ./...`.
- Result: 14/14 cards from one run with no fix (12 at the first attempt), **$0.0852**, 7.1 minutes.
- Every card is its own commit with `Morph-Card` / `Morph-Model` trailers. `.morph/runs/` keeps the deck, the report
  and the model's answers, and `.morph/primer.md` is MorphV2's primer of this tree.

```
GOFLAGS=-mod=mod GOPROXY=off go vet ./... && gofmt -l . && GOFLAGS=-mod=mod GOPROXY=off go test -count=1 ./...
```
