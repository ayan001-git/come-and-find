# Verification - 15 October 2026

## Passed

- Go 1.22.2 build (`go build -buildvcs=false`), Linux amd64.
- `go vet ./...`.
- Go unit tests for smart query parsing: passed (`go test ./...`).
- JavaScript syntax (`node --check web/app.js`).
- API regression: **66/66 checks**. See tests/api_results.txt.
- Frontend DOM regression with jsdom and the real Go API: **24/24 checks**.
  See tests/frontend_results.txt.
- Generated report PDFs rendered and visually inspected.

API checks include authentication, role boundaries, repeated additions,
quantity replacement, stock limits, mixed supplier rejection, atomic checkout
failure, payment, forward/backward/skipped statuses, listing approval/withdrawal,
user activation/deactivation and session invalidation, price filters,
registration validation and cross-origin write rejection.

New search checks cover Russian/English names, number words, thousands, category/price extraction, explicit-filter precedence and invalid queries. DOM voice tests use a mocked SpeechRecognition to verify consent-driven start, language, editable transcript, errors and text fallback.

## Important limits

The running SQL test backend was PGlite accessed through a PostgreSQL wire server,
with DB_MAX_OPEN=1. This is not the same as a native PostgreSQL 15/16 server.
Native PostgreSQL concurrency, actual Docker startup and existing-data upgrade
were not executed here. Test these on your demonstration machine.

DOM tests execute the real frontend script and live HTTP requests, with dialog
and scrolling APIs stubbed. They do not verify Chrome/Safari rendering, mobile
layout or real browser cookie security. Chromium download failed in this
environment, so no real-browser screenshots are claimed.

The archive contains source only; no Linux executable is presented as a Windows
program. No real bank gateway, delivery, user interviews, sponsor approval,
grooming ceremony or planning poker votes were tested or fabricated.

Passing these checks fixes the identified regressions; it does not prove that
all possible defects have been eliminated or that this is production-ready.
