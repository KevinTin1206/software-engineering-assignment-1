# Validation — 2026-11-01

Environment: Go 1.23.3, macOS arm64; all production dependencies are Go standard library.

## Automated checks

- `go test -count=1 ./...`: PASS, 17 named tests (table cases within tests exercise additional boundaries).
- `go test -race -count=1 ./...`: PASS.
- `go vet ./...`: PASS.
- `go build`: PASS; compiled output kept outside the source submission.
- `node --check web/app.js`: PASS; Node is used for development verification only.
- `gofmt -l`: no unformatted files; `git diff --check`: PASS.

## Browser verification

Ski: create fictitious person, upload PNG via real file chooser and display saved
indicator; preview three calendar days at 66 EUR, reserve, checkout; change rate,
verify old rental still 66 EUR, return and release equipment.
Bicycle: independently load City/Trekking/E-Bike configuration; City 3-day quote
45 EUR; reserve, checkout, return; Dashboard updates. Browser error logs empty.

Cross-window regression on final ski binary: window A previews 66 EUR; window B
changes duration tariff to 26 EUR/day; A confirmation returns conflict, shows
78 EUR and an explicit price-changed message while keeping the form open.
A second confirmation saves the 78 EUR reservation. Core fix is identical in
both projects and covered by store and HTTP regression tests.

## Review and packaging

A fresh read-only reviewer inspected both projects. One Important stale-price
confirmation defect was found and fixed with failing-then-passing regression
tests. No Critical or additional Minor findings remained.

Each source ZIP was extracted to its own new directory and tested/built again.
The extracted binary was started from a different working directory; embedded
HTML/JS/CSS, API config, initial Dashboard and reserve/checkout/return all passed.
Final archive hashes and commit IDs are recorded in the outer delivery report.

## Deliberate boundaries

- Two independent local Git repositories protect unrelated files in the parent
  Documents repository. Their shared implementation is copied, so future common
  edits must be synchronized deliberately.
- No authentication, database, payment, seasonal prices or plugin framework;
  these are outside the approved prototype scope.
- Seed initialization reads the clock more than once. Starting exactly across
  a local midnight boundary could reject seed creation; restarting resolves it.
  This narrow demonstration edge was considered in review and accepted.
- User uploads are optional; no real identity documents are needed. One image per
  person, 5 MiB/20 MP maximum; all data and images are lost on restart.
