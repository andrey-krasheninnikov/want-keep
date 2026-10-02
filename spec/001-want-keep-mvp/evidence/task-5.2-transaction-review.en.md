# task-5.2 — transaction review and allowed commands

[Русский](task-5.2-transaction-review.md)

## Result

The backend persists the bounded `transaction_review_v1` contract, frozen projections/reference maps, separate validation jobs for saved responses, proposals, questions and answer revisions. Classification uses active categories/merchants; distribution uses saved rules or an explicit member answer. Links use existing matching only after proposal approval. The task-2.4 mechanism continues handling evidence-proven relationships. Models cannot assign unknown fees, original amounts, dates, bank statuses or protected values.

Provider outcomes and `ai_validation` jobs commit atomically. Server validation rechecks permissions, revisions and field protections. Effects use the existing ledger with audited decisions, outbox and review requests. `no_change` and validation outcomes create no financial revisions. Invalid/stale proposals leave no partial effect. Free-text answers retain their author and distinct answer revision; the original paid call is not repeated.

## Changed areas

- `ai/domain`, `ai/application`: bounded commands, reference checks, response validation and member answers.
- `storage` and migration 023: household relationships, immutable projections/validation receipts/answers and state transition history, recovery of saved outcomes and separate queues; migrations 001–022 remain unchanged.
- `delivery/review`, API/worker wiring and OpenAPI RU/EN: clarifications, proposals, session/Origin/CSRF, idempotency, safe errors and session-bound pagination; transaction detail/history includes status and references.
- Runtime prompt/schema, offline suite, PostgreSQL/race suite, CI and documentation catalog. Go/TypeScript contracts change together.

## Verification

Passed `make check` (including reproducible Go/TypeScript outputs and documentation), offline `make eval-ai SUITE=transactions`, the new PostgreSQL `ai-commands` suite with race detection, and integration suites `ai-budget`, `jobs`, `audit`, `matching`, `family-allocation`, `ledger`. Changed `ai-budget`/`jobs` also passed `-race`. The required privacy suite passed through `make test-integration AREA=privacy` with the isolated document processor. `git diff --check` passed. Intermediate Unix socket and missing-processor failures were resolved with the supported environments; neither failure is presented as pass evidence.

New suite scenarios: separate durable validation after provider outcome; classification without another money movement; a free-text answer with a distinct job; malformed/money/injection boundary; stale revision; competing answers and replay; household isolation; human correction protection and rollback; HTTP/CSRF, safe 404, pagination limits, command replay and clarification completion. Existing monetary/lease/budget/privacy regressions remain separate checks.

## Coverage and limitations

Evidence covers backend parts of AC-012/018/019/021/022/060/064/069/078/081/085/086/090, without claiming full product acceptance. Catalogs/rules are bounded to 100 entries and candidates to 20; projections state incompleteness. Receipt-item classification, chat, budget/goals, notification delivery and UI remain with their owning tasks. Applying budget/goal proposals without their handlers returns `feature_unavailable`.

Preserve `gpt-5.6-terra`, xhigh and the shared USD 50 cap. The new schema/prompt has new fingerprints and offline verification; `production_admitted=false`. Historical 206/206 results do not qualify the new runtime. Live OpenAI IO, provider qualification, Chrome/Arc and production were not exercised. SDD remains **Ready for development**; operational readiness is not established.
