# Want Keep MVP implementation plan

[Русский](plan.md)

Status: **Ready for development**. Gate date: 2026-09-07. Basis: [task-0.10 evidence](evidence/task-0.10-readiness.en.md), D-37–D-43 decisions and the [complete backlog](backlog.en.md).

The plan is decision-complete at SDD level. It permits tasks to start through the dependency graph but does not claim an implemented application. Provider deployment, production and the complete MVP have separate exit gates.

## Execution rules

1. Work within one `task-*` card, its targets and acceptance; preserve stable REQ/AC/task IDs.
2. Before edits, verify exact branch/base, dirty changes, local AGENTS and the owning-layer contract.
3. Never invent provider fields. Post a confirmed monetary effect once; keep a missing field `unknown` and a gap `source_partial`. Only an unverified identity, status or monetary effect and a `source_ambiguous` collision create no posting.
4. External integrations are read-only. Under D-43, provider deployment stays disabled until server-owned admission of the exact binding; `provider_not_admitted` never starts the collector.
5. Run the card's targeted checks, then `make check`; record external readback and CI separately.
6. Merge only a reviewable diff with paired RU/EN and traceability when a contract changes.

## Order and parallelism

| Wave | Tasks | Entry | Parallelism | Exit |
| --- | --- | --- | --- | --- |
| 0 — SDD | task-0.1–task-0.10 | Interview and available research evidence | Complete | Ready evidence and this plan published |
| 1 — core contracts | task-1.1 → task-1.2 → task-1.3 → task-1.4 → task-1.6 → task-1.5 | task-1.1 is complete; task-1.2 is next | After task-1.4, task-1.6; task-1.5 awaits both | Money/API, storage/retention, passkey, family authz and secrets complete |
| 2 — ledger | task-2.1 → task-2.2 → task-2.3; then task-2.4/task-2.5/task-2.6; then task-2.8 → task-2.7/task-2.9 | Storage and membership | 2.4, 2.5, 2.6 in parallel; 2.7 after 2.6/2.8, 2.9 after 2.4/2.8 | Exact ledger, audit, reconciliation, split/refund/debt without duplicate effects |
| 3 — ingestion | task-3.1 → task-3.2 → task-3.3 | Storage, accounts, ledger, secrets | task-5.1 may start after 3.1; FX after 3.2 | Durable jobs, normalized import contract and isolated collector |
| 4 — providers | task-4.1–task-4.6 | 3.3, 2.4, 2.5 and provider-specific research | All six in parallel after shared deps | Contract fixtures + live readback + each provider's deployment gate |
| 5 — AI | task-5.1 → task-5.2 → task-5.3/task-5.4; task-5.5 after task-6.8 | OpenAI evidence, jobs, secrets and ledger | 5.3/5.4 by their deps; AI never blocks ordinary accounting | Budget-safe gateway, every-operation review, receipts/chat/clarifications, insights |
| 6 — finance | task-6.1; task-6.2; task-6.3 → task-6.4; task-6.5; task-6.6 → task-6.7 → task-6.8 | Ledger; provider-derived functions await matching task-4.x | 6.1/6.2/6.5 and preparation for 6.6 by deps | FX/XIRR/credit/savings/budget/goals/daily limits pass contract tests |
| 7 — desktop | task-7.10 → task-7.11; task-7.16; task-7.1 → task-7.9; task-7.2–task-7.8, task-7.13/task-7.14/task-7.15; task-7.12 | API/auth contracts and matching domain read models | Design system alongside backend; screens after their deps | SCR-001–SCR-035, RU/EN, Chrome/Arc, 1280×720/1440×900, accessibility and motion acceptance |
| 8 — operations | task-8.1 → task-8.2 → task-8.3 | Collector, AI, notifications; separate resource authorization | Docs/scripts may start early; provisioning only when authorized | Hardened deployment, measured budget/load, encrypted Mac backup and restore rehearsal |
| 9 — acceptance | task-9.1 → task-9.2 | Every listed dependency and provider gate | None: frozen acceptance candidate | All mandatory ACs, independent final Avida and handoff evidence |

The `catalog.json` graph remains the source for exact dependencies. A range in this table never permits bypassing an individual card dependency.

## Provider entry/deployment gates

| Provider task | Preferred transport | Before mapping implementation | Before deployment enablement |
| --- | --- | --- | --- |
| task-4.1 Alfa | Official structured read, otherwise authorized Playwright | D-47/D-48 read connector; unknown source records without postings | Financial mapping, live checks and deployment admission move to task-4.7 / #94 |
| task-4.2 Raiffeisen | RBO API/CAMT | CAMT 1:N, canonical cross-report fingerprint + optional-ID aliases, CLBD/unknown semantics | OAuth rotation, corrections/reversals, full history, second account, live conformance |
| task-4.3 Ozon | Verified session read | Sanitized HAR projection, route namespaces, parent fee relation | Session permission/lifecycle, stable account ID, history end, second account, reauthentication |
| task-4.4 Bybit | Official RSA read-only API | Route IDs, candidate links, hourly collision policy | Precision/history, two accounts, key rotation/revocation, write-route denial |
| task-4.5 Aifory | Structured session read; Playwright only when necessary | D-33 namespace/unknown rules | Permission, structured fixtures, card lifecycle/fees/FX, pagination, reauthentication, two accounts |
| task-4.6 EMCD | Structured session read; Playwright only when necessary | D-34 namespace/unknown rules | Wallet/Grow/card/P2P fixtures, balance/lifecycle/fees, pagination, reauthentication, two accounts |

A failed provider gate keeps that source disabled. `backend/internal/connections/admission/` and the task-1.3 storage adapter own the aggregate, monotonic `admissionRevision`, atomic combine/invalidate, admission-check + enqueue and the commit-time result fence. Only the admission service combines task-4.x provider evidence and task-8.x host evidence for the exact D-43 binding; a stale binding closes new sync and a stale result remains in quarantine without source/posting. Cancellation of a started read is best effort. It does not block manual accounting, domain features or other admitted sources. No fallback includes write actions.

## Contract entry/exit gates

### task-1.2

Entry: task-1.1 in target; contracts version 10. Exit: Money/Asset/Rate/coverage, generated OpenAPI boundary, `source_partial`, `source_ambiguous`, `valuation_unavailable`, `quote_unavailable`, `command_expired`, `provider_not_admitted`; connection `deploymentGate.status` and binding; domain policy denies sync before admission and user DTOs cannot assign admission; D-41 recent/detail/tombstone semantics covered by tests. Task-3.3/task-4.x/task-8.1 verify actual atomic enforcement before job/IO.

### task-1.3

Entry: versioned task-1.2 API/value objects. Exit: atomic source/posting/revision/outbox; D-39 unique key and collision evidence; D-41 independent cleanup jobs; persisted D-43 aggregate/repository, monotonic admission revision, atomic provider+host combine/invalidate, admission-check + job enqueue and commit-time stale-result quarantine; restart/revocation-before-and-after-IO/binding-race tests on isolated PostgreSQL.

### task-6.1 and task-6.4

FX exit: CBR/Frankfurter/CoinGecko rules, ≤365-day coverage, `valuation_unavailable`, quote separation, quota/attribution runtime check. XIRR exit: Actual/365, same-day aggregation, one sign transition, fractional powers in 50-digit HALF_EVEN decimal with `1e-24` error bound, bisection from `-1 + 1e-12` through `1,000,000`, and irregular/boundary vectors.

### task-9.1/task-9.2

Full-MVP exit requires every linked AC, six admitted provider connectors, retry/duplicate/revision scenarios, family authz, AI injection/cost failures, RU/EN desktop UX, Chrome/Arc, notifications, target deployment and backup/restore. task-9.2 freezes the candidate and leaves no confirmed P0–P3 findings.

## Verification commands

Baseline for every task:

```sh
make docs-check
make check
git diff --check
```

A card adds its targeted command: `make test-go`, `make test-contract`, `make test-integration`, `make test-web`, `make test-collector`, `make e2e`, `make eval-ai`, `make check-deploy`, `make backup-check` or `make restore-check`. A fail-fast missing suite is expected before implementation and is not a pass.

Reports separate local/synthetic, CI, authorized provider readback, real Chrome/Arc, target-host/runtime and backup/restore evidence. One form of evidence never substitutes for another.

## Complete coverage control

All 67 tasks are listed explicitly; the dependency graph above determines order:

- `task-0.1` — Verify Alfa-Bank read contract
- `task-0.2` — Verify Raiffeisenbank Russia read contract
- `task-0.3` — Verify Ozon Bank read contract
- `task-0.4` — Verify Bybit read contract
- `task-0.5` — Verify Aifory Pro read contract
- `task-0.6` — Verify EMCD read contract
- `task-0.7` — Verify free FX and quote sources
- `task-0.8` — Measure OpenAI quality and cost
- `task-0.9` — Verify infrastructure and server budget
- `task-0.10` — Close contracts and review SDD readiness
- `task-1.1` — Create project structure and verification commands
- `task-1.2` — Define money types and API contract
- `task-1.3` — Create storage and transaction boundaries
- `task-1.4` — Implement passkeys and access recovery
- `task-1.5` — Protect secrets and private attachments
- `task-1.6` — Implement household membership and resource permissions
- `task-2.1` — Implement accounts and opening balances
- `task-2.2` — Implement the transaction ledger and states
- `task-2.3` — Add revisions, corrections and audit
- `task-2.4` — Link transfers and prevent duplicates
- `task-2.5` — Reconcile the ledger to source balances
- `task-2.6` — Separate categories, merchants and items
- `task-2.7` — Implement refunds and expense allocation
- `task-2.8` — Allocate household expenses and items to members
- `task-2.9` — Track explicit inter-member debts and reimbursements
- `task-3.1` — Create durable background jobs
- `task-3.2` — Define connector ingestion contracts
- `task-3.3` — Create an isolated browser collector
- `task-4.1` — Implement Alfa-Bank connector
- `task-4.2` — Implement Raiffeisenbank Russia connector
- `task-4.3` — Implement Ozon Bank connector
- `task-4.4` — Implement Bybit connector
- `task-4.5` — Implement Aifory Pro connector
- `task-4.6` — Implement EMCD connector
- `task-4.7` — Verify Alfa financial mapping and live admission (#94); depends on the task-4.1 read connector and blocks production admission and full task-9.1 acceptance.
- `task-5.1` — Create OpenAI gateway and spend control
- `task-5.2` — Review every transaction through AI commands
- `task-5.3` — Process receipts and line items
- `task-5.4` — Implement chat and clarification queue
- `task-5.5` — Generate grounded AI insights
- `task-6.1` — Implement rates and currency valuation
- `task-6.2` — Account for credit cards and grace periods
- `task-6.3` — Calculate accruals and savings forecasts
- `task-6.4` — Compare dated cash-flow returns
- `task-6.5` — Aggregate trading P&L and mining
- `task-6.6` — Plan monthly budgets and income
- `task-6.7` — Reserve money for goals
- `task-6.8` — Calculate daily allowances and liquidity forecast
- `task-7.1` — Create the desktop shell and RU/EN sign-in
- `task-7.2` — Show accounts, transactions and corrections
- `task-7.3` — Create chat with account selection and files
- `task-7.4` — Create the monthly budget editor
- `task-7.5` — Show goals and reservations
- `task-7.6` — Build the dashboard, limits and insights
- `task-7.7` — Show credit cards, savings and returns
- `task-7.8` — Add notifications and web push
- `task-7.9` — Add household context and ownership to the interface
- `task-7.10` — Create tokens, typography and brand assets
- `task-7.11` — Create the Base UI component catalog
- `task-7.12` — Verify desktop UX and visual acceptance
- `task-7.13` — Create connection and reauthorization screens
- `task-7.14` — Create settings and accounting health screens
- `task-7.15` — Add contextual financial-event animations
- `task-8.1` — Prepare deployment and system health
- `task-8.2` — Pull encrypted backups to the MacBook
- `task-8.3` — Verify recovery from a local backup
- `task-9.1` — Run end-to-end acceptance of the full MVP
- `task-9.2` — Run final Avida review and hand off the MVP
