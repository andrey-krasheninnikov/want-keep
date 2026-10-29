# task-5.3: receipts and line items

[Русский](task-5.3-receipts.md)

## Result

The backend links an accepted attachment and selected household account to durable processing of safe raster pages. Each page uses the shared budget-controlled Responses gateway, a separate `receipt_extraction_v1` contract and server validation of persisted output. A receipt records one expense, requests clarification, skips with a reason or links to a confirmed payment.

Items and discounts use exact six-asset arithmetic. Unknown facts never become zero; conflicting pages and mixed receipts do not post money. Category aliases are bounded by frozen context and the current active revision. Allocation, matching, corrections, audit and refunds reuse existing services. File replays retain authored evidence without another call. A date without time retains `day` precision.

API: `POST /receipts`, `GET /receipts/{id}`, `POST /receipts/{id}/answers` and `POST /receipts/{id}/allocations`. Mutations require session, Origin/CSRF, Idempotency-Key and revision checks; unknown command outcomes recover through `/commands/*`. Migration 025 is additive; 001–024 remain unchanged. Page images never enter `ai_attempts.allowed_input`.

## Evidence

Passed `make check` (formatting, vet/lint, TypeScript, unit/tooling, builds, RU/EN and reproducible contracts), integration suites `receipts`, `ai-budget`, `ai-commands`, `jobs`, `audit`, `matching`, `family-allocation`, `refunds`, `ledger`, `storage`, and mandatory `privacy` with the isolated processor. Race suites `receipts`, `ai-budget`, `jobs` and focused extraction-contract race tests passed. Later decoder and OpenAPI edits were checked through affected unit/HTTP/typecheck scenarios; unchanged successful checks were reused. `git diff --check` passed. No final CI result is claimed here.

The PostgreSQL `receipts` suite covers HTTP, exact six-asset round trips, the 600 + 400 − 100 = 900 discount, two authors of one file, replay/changed payload, foreign resources, Origin, concurrency, rollback, clarification and version conflict. A possible match requires confirmation and creates no second expense. Mixed items give A 400 / B 600; returning shared items worth 200 preserves the original month and 100 per member. Unknown provider outcomes are not retried. The offline suite checks the strict extraction schema and rejection of extra commands/fields.

Live verification used a synthetic RU image, an EN image with USDC 0.000001, a two-page PDF, a menu and a prompt-injected receipt. All five final scenarios passed through the current gateway and PostgreSQL; PDF processing ran in an isolated networkless process. Ten generation calls include correction of a test comparison, continuation-page support and verification of the atomic cap journal. Persisted usage at pinned prices totals USD 0.076896 of the authorized USD 2. Missing cache-write counts use conservative pricing. Provider billing was not independently reconciled; no live outcome remains unresolved.

`make eval-ai SUITE=receipts MODE=live` requires `WANT_KEEP_RECEIPTS_LIVE=1`, explicit `WANT_KEEP_OPENAI_API_KEY_FILE`, `WANT_KEEP_RECEIPTS_LIVE_JOURNAL`, `WANT_KEEP_TEST_PROCESSOR_SOCKET` paths and isolated test PostgreSQL. The private journal must already exist with mode 0600 and initial contents `[]`. Never recreate it between attempts: an unresolved entry blocks further paid calls, and each new reservation must fit the remaining USD 2. The key stays out of source, images, arguments and the spending journal.

## Limits

Backend portions of AC-015/016/017/022/050/060/068/081/091/093 are covered. Shared chat, upload/question screens, Chrome/Arc, real user receipts, production and full model qualification remain unverified. Skip reasons and clarification are exposed through the API; their UI belongs to subsequent tasks. Browser acceptance from #97 does not prove this new pipeline.

The shared USD 50 household cap, `gpt-5.6-terra`/`xhigh` and `production_admitted=false` remain. There is no automatic model downgrade. SDD remains **Ready for development**. Root `VERSION` 0.0.1 does not grant operational admission.

Additional regressions cover distinct equal refunds, another scan of a refund, corrected current totals, terminal validation failure, more than 20 candidates and a candidate change outside the visible page. The live cap journal rejects empty state, updates by atomic replacement under a separate stable lock and preserves concurrent history.

Each page category is checked again against its frozen alias/revision at finalization. Renaming or archiving leaves classification empty without losing the confirmed payment; persisted extraction stays unchanged.
