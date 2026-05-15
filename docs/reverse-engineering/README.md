# Reverse Engineering Notes

This directory holds Toss Securities web protocol research artifacts. The catalog must grow before the Go client grows — any endpoint promoted into `internal/client/` should already appear here with its observed shape and status.

## Documents

| File | Topic |
| --- | --- |
| [`rpc-catalog.md`](rpc-catalog.md) | Canonical endpoint catalog — host map, status legend, all observed endpoints by surface (bootstrap, login, market overview, quote, account/portfolio, transactions, chart vocabulary, realtime strategy, signals). |
| [`order-page-deep-dive.md`](order-page-deep-dive.md) | Full write-up of the order page (`/stocks/{code}/order`) reverse engineering — covers chart timeframes/sessions, realtime quote/orderbook/tick model, full holdings payload, AI signals + news + related stocks, KR vs US orderbook depth. Captured 2026-05-15 on SOXL with cross-checks on Samsung Electronics. |
| [`push-events.md`](push-events.md) | SSE push channel (`sse-message.tossinvest.com/api/v1/wts-notification`) — thin notification taxonomy and re-fetch mapping. |
| [`auth-notes.md`](auth-notes.md) | Login/session/cookie observations (QR flow, session persistence). |
| [`capture-workflow.md`](capture-workflow.md) | How to capture safely, scope rules, what to sanitize before commit. |

## Policy

Do not commit raw captures containing sensitive cookies, tokens, account numbers, or personal data. Use:

- `python3 tools/sanitize_har.py <input.har> <output.har>` for HAR sanitization
- `python3 tools/fetch_public_fixtures.py fixtures/responses/public` for refreshing the public fixture set

Raw `.network-response` payloads from `chrome-devtools` MCP belong under `.captures/` (already gitignored). Promote stable shapes into the catalog and small redacted samples into `fixtures/responses/public/`.

## Read-only scope

The Go client only admits endpoints that are:

- observed in [`rpc-catalog.md`](rpc-catalog.md)
- explicitly classified as `public`, `guest`, or `auth` read paths
- mapped to an approved CLI command

Mutation endpoints (order placement/modification/cancel, watchlist mutation, comment posting, telemetry) stay blocked.
