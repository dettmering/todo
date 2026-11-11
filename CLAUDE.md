# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Working agreements

- **Document API changes in `README.md`.** Whenever you add, remove, or change
  an HTTP endpoint, its query parameters, request/response shape, or events,
  update the matching section in `README.md` in the same change (endpoint
  overview table, the per-endpoint section, the object example, and the events
  table where relevant). The README is the canonical API reference — keep it in
  sync, don't let it drift.
- **Write backend unit tests where sensible.** Business logic in
  `internal/domain` and `internal/service` should come with table-driven tests
  (`go test ./...`), following the existing `*_test.go` files (e.g.
  `internal/domain/absence_test.go`, `internal/service/absence_test.go`). Pure
  domain rules — date/visibility/overlap logic, validation, recurrence — are the
  priority. Don't add tests for trivial DTO plumbing or generated code.

## Commands

```bash
# Run the server (defaults: addr :8080, db ./todo.db)
go run ./cmd/server
go run ./cmd/server -addr :8080 -db ./todo.db

# Compile the whole module
go build ./...

# Static linting / vet
go vet ./...

# Run the test suite
go test ./...

# Container build + run
docker build -t todo:dev .
docker run --rm -p 8080:8080 -v todo-data:/data todo:dev

# Regenerate the license notices (after changing go.mod or an embedded library)
scripts/third-party-notices.sh

# CI workflows:
#   .gitea/workflows/dockerhub.yml      — push to master / tag pushes dettmering/todo
#   .gitea/workflows/build-binary.yml   — workflow_dispatch, uploads todo-linux-amd64
#   .github/workflows/ci.yml            — vet, test, build on the GitHub mirror
```

Gitea is the main repo. GitHub (`github.com/dettmering/todo`) is a push mirror
of it: never commit there directly, the next sync would overwrite it. Gitea runs
only `.gitea/workflows/`, GitHub only `.github/workflows/`.

Go toolchain: `github.com/ncruces/go-sqlite3` v0.37.x requires Go ≥ 1.26, which
is what `go.mod` declares. The Dockerfile and the Gitea workflows build with Go 1.27.

## Architecture

The codebase is a strict layered Go service with an embedded vanilla-JS
frontend. Each layer only depends on the ones below it.

```
cmd/server                 wires everything together, embeds web assets
  └── internal/httpapi     REST handlers (Lastenheft §5.9 paths)
       └── internal/service business logic: validation, IDs, timestamps, event emission
            ├── internal/storage          repository interfaces only
            │    └── internal/storage/sqlite   the only concrete impl (ncruces/go-sqlite3, no CGO)
            ├── internal/events           in-process pub/sub bus
            └── internal/domain           pure types (Task, Project, LocalDate, Priority)
  └── internal/web         go:embed of static/ — frontend ships in the binary
```

**Why this matters when changing things:**

- `internal/domain` has no dependencies on anything else in the project.
  Keep it that way — types here are also used in event payloads and the JSON DTOs map onto them.
- `internal/storage` is interface-only. Add new query knobs to `TaskFilter`
  and the interface; implement in `sqlite/`. New backends would slot in here.
- `internal/service` is the **only** place that mints IDs (`task_*`,
  `project_*`), sets timestamps, publishes events, and runs validation.
  HTTP handlers must not call repos directly.
- `internal/events.Bus` is the integration point planned for webhooks
  (Lastenheft §5.11). Wire new subscribers in `cmd/server/main.go`.
  Handlers run in goroutines and recover from panics — they must not block.
- `internal/httpapi` uses Go 1.22+ pattern routing (`GET /api/tasks/{id}`).
  DTO ↔ domain conversion lives in `internal/httpapi/json.go`.

## Load-bearing conventions

These choices look unremarkable but are easy to break:

- **Migrations are versioned via `PRAGMA user_version`** in
  `internal/storage/sqlite/sqlite.go`. Append a new `{version, sql}` step;
  never edit or reorder existing ones. SQLite `ALTER TABLE` is limited —
  prefer additive columns with `DEFAULT`.
- **Encryption is a VFS, not a column format.** With `ENCRYPTION_KEY` set,
  `sqlite.Open` uses the `adiantum` VFS and sets the key per connection via
  `PRAGMA hexkey` in the connect hook (`internal/storage/sqlite/encrypt.go`);
  the hexkey must run before any other pragma, so pragmas don't go into the
  DSN as `_pragma`. Repos never see ciphertext — don't add per-field
  encryption. A plaintext file is converted once on start via `VACUUM INTO`;
  the file's first 16 bytes (`SQLite format 3\0` or not) decide which case
  applies.
- **Project deletion is archival, not removal.** `service.DeleteProject`
  sets `archived = true`; tasks keep their original `project_id`. The
  frontend loads with `include_archived=true` so historical project labels
  still resolve. Inbox (`id = "inbox"`) is reserved, seeded on first boot,
  and **never deletable**.
- **An epic is archived, never deleted**, like a project — `service.DeleteEpic`
  sets `archived = true` and tasks keep their `epic_id`, so an old label still
  resolves. Its status (`active`/`completed`) is a separate axis and moves only
  through `CompleteEpic`/`ReopenEpic`, never through `UpdateEpic`: the "no open
  tasks left" rule has exactly one home.
- **`task.epic_id` implies `task.project_id`.** Linking a task to an epic moves
  it into the epic's project; moving an epic moves all its tasks. Moving a
  linked task to a different project clears `epic_id` instead — in a patch that
  names both, the epic wins. `epic_id` is an empty string when unset — no
  foreign key, because '' would violate one.
- **Tallies are counted, never loaded.** `TaskRepo.CountByEpic` /
  `CountOpen` and `CommentRepo.CountByTasks` answer with numbers out of a
  covering index; the epic tallies, the sidebar badges and `comment_count`
  all go through them. Never go back to "list everything and count in Go" —
  that made `GET /api/epics` read all 10 000 task rows for a 10 KB answer.
  Migrations 23/24 (`idx_tasks_epic_status`, `idx_tasks_open_counts`) exist
  only for these queries, and their column order *is* the query: add a filter
  column that is not in the index and the plan silently degrades to a table
  scan. Check with `EXPLAIN QUERY PLAN` — it must say COVERING INDEX.
- **gzip is wired per route, never around the whole mux.** `cmd/server/gzip.go`
  wraps `/static/`, `/api/` and the SPA catch-all. `/mcp` is left out on
  purpose: Streamable HTTP streams, and a compressor in front of a stream holds
  bytes back. Wrap `root` and you break MCP without a test failing.
- **List responses carry no comment bodies.** `GET /api/tasks` uses
  `toTaskListDTO` (count only); the single-task responses use `toTaskDTO`
  (bodies included). The frontend fetches bodies per task when a modal opens.
- **Task deletion is soft.** `DELETE /api/tasks/{id}` sets
  `status = 'deleted'`; rows are kept. Project archival can therefore
  always succeed.
- **Default priority is P4** (low). The service layer fills this when
  `priority == 0` is submitted. The schema default is still `3` — don't
  rely on it; the service is the source of truth.
- **Dates are timezone-naive.** `domain.LocalDate` stores `YYYY-MM-DD`.
  `visible_from` (when the task appears) and `deadline` (when it's due)
  are distinct fields. See `Task.VisibleOn` for the visibility rule
  (Lastenheft §5.5).
- **The Lastenheft visibility filter is opt-in** on the API. List queries
  with `scope=all` skip it; the standard "Heute" view omits it (defaults
  on). The frontend uses `scope=all` for every grouped view.

## Frontend conventions

- **No build step.** `internal/web/static/{index.html, style.css, app.js}`
  are embedded via `go:embed` (`all:static`) in `internal/web/web.go`. Edit
  and rebuild. Inter and JetBrains Mono come from Google Fonts (link in the `<head>`), not
  from `/static` — self-hosting it cost six extra requests per load against
  the app's own host, and the reverse proxy's rate limit then rejected
  `app.js`.
- **Icons are inline SVG strings** in `app.js` (object `ICON`). Lucide
  stroke style, `currentColor`. The "Heute" sidebar icon is generated
  dynamically with the current day number (`todayIconSVG`). Paths taken from
  Feather or Lucide are credited in `THIRD_PARTY_NOTICES.md`.
- **CSS tokens live in `:root` of `style.css`.** The design is a
  reinterpretation of the md2pdf web UI (`../md2pdf/src/webui.js`): Inter for
  prose, JetBrains Mono for everything read off a scale, a pressroom
  palette with exactly two signal colors, and one rule for rounding —
  surfaces get `--radius` (4px), anything you press or that labels something
  gets `--radius-pill`. **`STYLEGUIDE.md` is authoritative** — read it before
  adding a color, a font role, a radius, or a component.
- **Quick-add defaults are derived from the current view**
  (`defaultProjectForView` in `app.js`): in a project view → that project,
  otherwise → Inbox; `visible_from` is auto-set to today; priority resets
  to P4 after submit.

## Product spec

`LASTENHEFT.md` is the canonical requirements document (German). API
paths, attribute names, and visibility rules trace back to specific
sections (e.g., comments in `httpapi/server.go` mark "Lastenheft 5.9").
When the spec and code disagree, fix the code or update the spec
deliberately — don't paper over.
