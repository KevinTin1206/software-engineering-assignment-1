# Rental Application Implementation Plan

> For agentic workers: use superpowers:executing-plans task by task; no per-task implementer delegation.

**Goal:** Deliver independently runnable ski a+b and bicycle c ZIP projects.
**Architecture:** Standard library HTTP server, domain rules, locked memory store, HTML/JS assets.
**Tech Stack:** Go >=1.23, HTML, CSS, JavaScript; no external dependencies.
**Spec:** DESIGN.md (approved user plan and upload amendment).

## Global Constraints
- German UI, integer euro cents, inclusive calendar dates, 5 MiB JPG/PNG uploads.
- Memory-only data; seeds restored after restart. Local Git, no publishing.
- Two ZIPs, each independently runs `go run .` and `go test ./...`.

## Review Focus
- Concurrent requests cannot book/checkout the same equipment twice.
- Pricing tiers, invalid dates and old quotes after edits remain correct.
- Missing people, bad JSON and oversized/corrupt uploads fail without mutation.
- Cancelled/returned records release inventory; physical overdue loans block checkout.
- User-provided strings display as text; ZIPs have no external-path dependencies.

### Task 1: Business rules and memory store
Files: internal/domain/{model,pricing}.go, internal/store/{store,catalog,rentals,documents}.go and tests.
Interfaces: domain.CalculateQuote(Rate,start,end) (Quote,error); store.New(clock) *Store;
Store methods provide item/person/rate CRUD, Quote, Reserve, Transition, Snapshot and documents.
- [x] Write domain tests for inclusive dates, leap days, tier thresholds, money limits.
- [x] Run tests and confirm missing implementation fails, then implement and pass.
- [x] Write store tests for rental lifecycle, overlap, races, snapshots and references.
- [x] Run tests RED, implement store, run full Go suite GREEN, commit.

### Task 2: HTTP API and identity upload
Files: internal/handler/server.go and server_test.go; web/assets.go.
Interface: handler.New(*store.Store, Config, fs.FS) http.Handler.
- [x] Write httptest flows for CRUD, quote/booking, upload/read/replace/delete, bad payloads.
- [x] Run RED, implement bounded JSON/multipart handlers and image validation, run GREEN.
- [x] Commit API and tests.

### Task 3: Demonstrable browser frontend
Files: main.go, web/{index.html,app.js,styles.css}, main_test.go.
- [x] Write smoke test for seed configuration and static/API routing; run RED.
- [x] Implement server and seeded data; HTML forms/tables, dashboard and image controls.
- [x] Run Go checks and browser interaction for a complete rental flow and pricing/upload.
- [x] Commit working ski application.

### Task 4: Independent bicycle project and delivery
Files: bicycle copy of tested common code, distinct main.go/config, READMEs and archives.
- [x] Adapt categories/fields/seeds with tests establishing bicycle domain, run RED then GREEN.
- [x] Write run/demo/API/reuse/limits documentation; retain original template as CLAUDE.md.
- [x] Fresh final review; fix consequential findings with regression tests.
- [x] Run test/race/vet/build/format checks for both projects, commit all changes.
- [x] Create two ZIPs without .git/cache/binaries; extract separately and rerun tests/build/start.
