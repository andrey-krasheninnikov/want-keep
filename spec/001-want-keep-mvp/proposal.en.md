# Want Keep MVP product proposal and decisions

Execution mode: autonomous

D-44, 2026-09-07: independent task-1.2 implementation before research completion was previously labeled D-37 on its branch. D-37 now retains the Alfa decision from task-0.10. Branch base and PR target are `docs/want-keep-mvp-sdd`. The SDD is Ready for development; application operational readiness is not yet established. The user accepted D-41 instead of the earlier family-lifetime command retention.

Initially agreed on 2026-09-06; family amendment agreed on 2026-09-07. Basis: the detailed interview and the user's explicit request to implement the agreed documentation and GitHub backlog plan.

## Problem and outcome

Manually maintaining fragmented fiat/crypto finances is time-consuming. Want Keep combines owned accounts, transactions and documents into verifiable accounting, supports monthly planning and explains outcomes through AI.

The first release is a family pilot for a couple. Expected load is hundreds of monthly transactions; daily chat, receipt, budget and clarification work targets up to 45 minutes according to the owner. There is no current application to migrate; importing old Excel records is not mandatory scope.

This stage delivers a bilingual Ready specification, criteria, architecture, integration matrix, full backlog, implementation plan and self-contained GitHub Issues. It does not implement the application, purchase infrastructure or deploy.

## Interview decisions

| ID | Decision |
| --- | --- |
| D-01 | One household, two members with separate sign-in, desktop web on macOS laptops in Chrome and Arc. All 15 capabilities and six integrations are required for the complete MVP. Hundreds of transactions/month; up to 45 minutes/day usage target. |
| D-02 | RUB, USD, USDT, USDC (D-36), BTC, ETH (D-33); cash and bank money are different accounts. Reporting currency switches. Owned money, availability, debt and credit limits are distinct. |
| D-03 | Hourly and on-demand automatic sync. History starts on a selected date; opening balances and coverage boundaries are explicit. |
| D-04 | Owned transfers/exchanges are linked without repeated income/expense; fees remain separate. Duplicates are handled in accounting. Originals and correction history remain; discrepancies are investigated. |
| D-05 | Full expense in the payment month, including annual subscriptions. Refunds recalculate the original purchase month; cash movement retains the refund date. |
| D-06 | Category/subcategory, merchant and receipt item are separate dimensions. Milk is an item; a venue is a merchant, not a mandatory subcategory. |
| D-07 | Text, photos and PDFs in web chat. Receipts require an account dropdown, including cash. Clarify uncertainty; explain skipping obviously irrelevant documents; record/link established transactions. |
| D-08 | AI maximally automates internal accounting and reviews every new/changed transaction. Missing financial values are not invented. Full transactions/receipts may go to OpenAI, excluding secrets. Ordinary accounting continues during AI failure. |
| D-09 | Calendar month; dated expenses, flexible categories and forecast income. Copy plans without remaining/overspend rollover. AI applies approved-plan changes only on an authorized member decision. |
| D-10 | Goals have amount, currency, deadline and virtual-reservation or dedicated-account mode. No double reservation. Daily allowances are overall/per-category and available/forecast with separate funding by currency. |
| D-11 | Credit cards with debt, payments and grace; savings with actual/forecast and comparable dated-cash-flow returns; realized/unrealized trading P&L, fees, funding and mining. No trading terminal. |
| D-12 | Transaction-date expense valuation, current wealth valuation and separate FX effects. Reference rates and available provider buy/sell quotes with fees. USD, USDT and USDC are not automatically equal. |
| D-13 | Alfa-Bank products under D-47; Raif Russia individual entrepreneur current account through RBO API (D-35); Ozon debit card and main account (D-32); Bybit Funding USDT/USDC/ETH/BTC, Easy Earn and P2P (D-36); Aifory RUB accounts, USDT, ETH and existing card (D-33); EMCD used cards, Coinhold/Grow, USDT wallet and P2P history (D-34). Read-only; browser automation is allowed for a proven API gap. |
| D-14 | Separate passkeys and personal one-time recovery codes for each member; partner-assisted reset is unavailable. Protected secrets, attachments and sessions; AI has no payment authority. |
| D-15 | For the whole household: server up to $40/month in DE/NL/BG, OpenAI up to $50/month; separately sourced data must be free. Hourly MacBook backups while reachable; visible backup age, conditional RPO and recovery target within four hours. |
| D-16 | UI, AI interaction and documentation in RU/EN. Dashboard covers plan/actuals, income/expenses, goals and daily allowances. In-app and web-push notifications. |
| D-17 | Go, managed PostgreSQL, React/TypeScript/Vite, separate Playwright TypeScript collector and Docker Compose. Production uses a 2 vCPU/4 GB/50 GB application server plus 1 vCPU/2 GB/20 GB managed PostgreSQL in one German private VPC; local environments use isolated PostgreSQL. Financial domain remains independent of transport/storage/UI/AI SDK with exact arithmetic and explicit boundaries. |

## Preserved boundaries

The application reads platforms and changes only its own accounting. Payments, actual transfers, trade orders and bypassing MFA/CAPTCHA are outside the product. Public registration, member exit/replacement, additional roles, mobile screens/adaptation, home-screen installation, native mobile, tax reporting, mining control and old Excel migration are outside this MVP.

Manual cash and receipt entry is mandatory. Manual bank-statement upload may aid research but does not fulfill automatic connection of a mandatory service.

## Engineering defaults

These are defaults, not user answers: initial `Europe/Moscow` timezone explicitly selected by the owner; online operation with clear offline status; two category levels; private push without monetary details; annualized money-weighted XIRR for investment comparison; retained originals for audit. Formulas, file limits and errors are in [contracts.en.md](contracts.en.md).

Fundamental contract changes require both languages, ACs and dependent tasks to change. Library versions are pinned during implementation within the selected stack. Provider-specific permissions, structured fixtures, a second account and production conformance are explicit task-4.x/task-8.x entry/deployment gates; unknown fields are never filled by inference.

## Readiness

On 2026-09-07 task-0.10 declared the SDD **Ready for development**: fundamental business and system decisions are closed, and safe behavior for incomplete provider data is contractual. This permits work under [plan.en.md](plan.en.md) and its dependencies. The full MVP is accepted only after all ACs, live conformance for mandatory products and operational gates; SDD Ready does not mean a running or deployed product.

## Family amendment

| ID | Decision |
| --- | --- |
| D-18 | User, household, membership and roles are separate. MVP has two members with a configured limit; no fixed member fields, public registration, exit or replacement. |
| D-19 | Both see everything and edit all transactions. Personal goals/plan portions are owner-editable; joint ones are editable by either member. One authorized approval suffices and the other is notified. |
| D-20 | Personal and household accounts. Account ownership, external owner, payer, actor and expense beneficiary differ; reconnecting one source does not duplicate it. |
| D-21 | One monthly household plan with individual views; all income and available funds are pooled. A personal allowance may be funded by the other member’s money. |
| D-22 | Personal/joint attribution by transaction and item; joint shares are 50/50 with plan-line/purchase overrides. AI applies rules and clarifies unknown attribution. |
| D-23 | Household transfers are not income/expense; inter-member debts and settlements are explicit and do not increase household wealth. |
| D-24 | Personal and joint goals receive explicit reservations; joint goals appear in a separate shared block without personal shares. No automatic allocation to goals. |
| D-25 | The household cap is calculated once per currency; individual and category limits partition it instead of repeating it. Goals, holds and obligations are deducted once. |
| D-26 | One shared AI chat; message and decision authors are retained. Text cannot replace principal; personal plan/goal changes require the authorized owner. |
| D-27 | Revisions and permissions are checked for edits, clarifications and reversal. Concurrent conflict requires refresh; the other member’s later edit is preserved. |
| D-28 | Both manage connections. The external-account owner supplies password/MFA; partner and AI receive no secrets. Platforms remain read-only. |
| D-29 | Goal/plan changes notify the other member; read state and push subscriptions are individual. Recovering one sign-in does not reset the other. |

## Desktop design and interaction

| ID | Decision |
| --- | --- |
| D-30 | macOS laptop Chrome/Arc only, 1280×720/1440×900 and zoom; dark #1A1A1A / #5F4EF5, Pixelify Sans + Manrope, shadcn on Base UI. Reference sign-in, 35 question/answer/action screens, clear states and accessibility. Mobile scope excluded, push retained. |
| D-31 | Contextual pixel animations: rocket for adding an accounting account, top-up sparkles, confetti/soft disco for achievement, calm overspend signal. Confirmed events only, presentation dedup, off and reduced motion; no payment execution or bank-product opening. |

## Ozon coverage refinement, 2026-09-07

| ID | Decision |
| --- | --- |
| D-32 | The current MVP supports the available Ozon debit card and linked main account. Credit cards, savings and deposits belong to a future contract extension; their absence does not block Ozon/the MVP. This explicitly changes the earlier matrix, rather than claiming those products do not exist at the bank. Debit-data reading quality/automation, household identity and shared accounting rules remain. Other providers are unchanged. |

## Aifory coverage refinement, 2026-09-07

| ID | Decision |
| --- | --- |
| D-33 | Current Aifory scope: RUB accounts, USDT, ETH and the existing crypto card with its actual USD balance. Other products/currency wallets, other cards, standalone P2P/referral products and service catalogs are deferred and do not block the MVP. Retain all included-wallet movements even when the related service is deferred. ETH joins accounting and selectable valuation currencies. Selected-product reliability/automation remain mandatory; expansion needs new verified contracts. Other providers and shared functions are not reduced. |

REQ-002/REQ-003/REQ-046, their ACs and downstream tasks were updated with IDs preserved. This is target-contract version 4; no application exists yet, so no runtime/data migration is required. [Research outcome](evidence/aifory.en.md).

## EMCD refinement, 2026-09-07

| ID | Decision |
| --- | --- |
| D-34 | Current EMCD scope: used crypto cards, Coinhold/Grow, the USDT wallet and historical P2P orders. Mining has never been used; neither its current data nor history is required. Mining and other unused products/wallets are deferred without blocking the MVP; expansion requires a new decision and verified contracts. Retain all included-wallet movements. Automation and quality for selected products, shared features and other providers are not reduced. |

## Raiffeisen refinement, 2026-09-07

| ID | Decision |
| --- | --- |
| D-35 | The owner explicitly confirmed the current Raif scope as the individual entrepreneur current account only: balances, incoming/outgoing movements, fees and history through RBO API. Raif personal cards, credit, savings accounts and deposits are excluded from the current MVP and do not block readiness. Other providers and shared features are unchanged. |

REQ-043/AC-043 and task-0.2/task-4.2 were refined with IDs preserved; the Raif link to AC-070 on credit/savings terms was removed. This changes the target contract; no runtime schemas or data are migrated. [Research outcome](evidence/raiffeisen.en.md).

## Bybit refinement, 2026-09-07

| ID | Decision |
| --- | --- |
| D-36 | Funding USDT/USDC/ETH/BTC, used Easy Earn and P2P. Official APIs take priority over Playwright. Spot/UTA trading, futures, options, card, On-Chain/Advanced Earn and other unused products are deferred without blocking; retain included-wallet movements. USDC is a distinct accounting and selectable valuation asset without assumed USD/USDT/USDC parity. Expansion requires a new decision and verified contracts. |

REQ-002/REQ-003/REQ-039/REQ-045 and task IDs remain; AC-071 and task-6.5 no longer require unused Bybit products. Shared financial functions remain. Target-contract version 6; task-1.1 foundation exists, financial API/database do not and no migration is needed. [Research](evidence/bybit.en.md). Authenticated RSA reads closed BYBIT-B02/B05 access; D-39 and the task-4.4 provider gate resolve the remaining questions without a global SDD block. See [private API evidence](evidence/bybit-api.en.md).

## Ready-gate decisions, 2026-09-07

| ID | Decision |
| --- | --- |
| D-37 | Current Alfa scope: debit card, current and savings accounts, deposits and cashback. The Alfa credit card is deferred; shared credit-card modeling and manual accounting remain. |
| D-38 | `Ready for development` means a decision-complete SDD with safe unknown/partial/ambiguous states. Operator permission, second-account, revocation, production reachability and executable conformance remain task-4.x/task-8.x entry/deployment gates. |
| D-39 | Source identity is `household + provider + stable external account + product/log namespace + provider record ID`. Connection/session ID is provenance; amount, time and text are not identity. A missing ID permits only a documented provider-specific immutable composite. A collision retains evidence as `source_ambiguous`, creates clarification and posts no money. |
| D-40 | CBR is primary USD/RUB, Frankfurter `providers=CBR` is fallback/cross-check, and CoinGecko Demo supplies current and up-to-365-day crypto history. Older history yields `valuation_unavailable`. A platform quote without direction, amount, time and known fee/spread yields `quote_unavailable`; a reference rate never substitutes for it. |
| D-41 | Terminal command detail is retained for 90 days after outcome; unresolved commands remain through reconciliation plus 90 days. A tombstone with `commandId`, scope, key/hash and outcome lives throughout unresolved state and for 400 days after terminal/reconciled outcome. `/commands/recent` returns 30 days of terminal commands and all unresolved commands. Expired detail yields `command_expired`; a live tombstone prevents a repeated effect. Financial audit is retained independently. |
| D-42 | XIRR uses Actual/365, same-date aggregation, both signs and exactly one sign transition; fractional powers use decimal ln/exp with at least 50 digits, HALF_EVEN and an NPV error bound of `1e-24`; bisection runs from `-1 + 1e-12` through `1,000,000` with `1e-12` solver tolerance and at most 512 iterations. Ambiguous, numerically unproven or non-convergent results are explained `unavailable`. |
| D-43 | Provider sync requires server-owned admission for the exact environment + adapter/collector build + contract + allowlist + non-secret configuration + operator-permission binding. The aggregate/repository and atomic combine/invalidate transition belong to the `backend/internal/connections/admission/` application boundary with a task-1.3 storage adapter. task-4.x and task-8.x provide the two evidence parts; the application admission service combines them. Jobs/results carry monotonic `admissionRevision`; pre-IO and commit-time revalidation prevent a stale result from creating a source record/posting. A binding change closes new sync with `provider_not_admitted`; cancellation of an already started read is best effort. |

D-45, 2026-09-07: task-1.4 implements closed bootstrap, personal passkeys, 12-hour sessions / 30-minute idle expiry and recovery that atomically revokes old access only after a new key is registered. Full rules and proven scope are in the [contract](contracts.en.md) and [report](evidence/task-1.4-identity.en.md).

D-46, 2026-09-07: task-1.5 uses existing membership/sessions without blocking on task-1.6. Secrets and files are encrypted separately; authorized accepted receipts are available to both members and restricted AI input. Key failure blocks dependent functions while preserving sign-in and ordinary accounting. Rules: [contract](contracts.en.md), [evidence](evidence/task-1.5-privacy.en.md).

## Alfa coverage update, 2026-10-28

D-48, 2026-10-29: by owner decision, task-4.1 delivers the read connector, accounts/aliases, unknown source records and encrypted evidence. Financial mapping and remaining live checks move to [task-4.7 / #94](https://github.com/andrey-krasheninnikov/want-keep/issues/94). Until completion, bank operations and balances have no confirmed monetary effect; production admission remains disabled. Full D-47 scope and D-39/D-43 remain. Task-9.1 acceptance depends on task-4.7.

D-47: by owner decision, the current Alfa connector covers current accounts, debit cards, transactions and cashback. Savings accounts, deposits and credit products are excluded from this connector and its mandatory acceptance. This decision supersedes the D-37 product scope; other providers and shared financial features remain in scope. A confirmed cash cashback payout is recognized as separate income on its payout date; an expected accrual creates no posting. Reauthentication and a second independent account remain mandatory live checks.


D-49, 2026-10-29: by direct owner clarification, issue #97 adopts the complete want-desyst theme, including Days One/Manrope fonts, palette, status roles, radii, focus and motion. This supersedes the visual portion of D-30 and previous local overrides. Keep the name, logo, desktop platform, financial and authentication contracts. Foundation/react are pinned to 0.2.1; want-desyst PR #9 contains the calendar fix. Preserve task-7.10/task-7.11 history; task-7.16 tracks the current migration.
