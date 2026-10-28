# Want Keep MVP requirements

Rendered from [catalog.json](catalog.json). Edit the catalog, then run `python3 spec/001-want-keep-mvp/tools/spec_tool.py render`.

Context, goals, exclusions and decisions are in [proposal.en.md](proposal.en.md); contracts/errors in [contracts.en.md](contracts.en.md); blockers in [verification.en.md](verification.en.md). Every requirement is mandatory for the complete MVP.

## REQ-001

The family pilot serves two members with separate sign-in and restricted joining; public registration is unavailable.

Source: `D-01`. Acceptance: [AC-001](acceptance_criteria.en.md#ac-001).

## REQ-002

Accounting supports RUB, USD, USDT, USDC, BTC and ETH; cash, bank money and platform wallets are separate accounts.

Source: `D-36`. Acceptance: [AC-002](acceptance_criteria.en.md#ac-002).

## REQ-003

The reporting currency can switch among RUB, USD, USDT, USDC, BTC and ETH.

Source: `D-36`. Acceptance: [AC-003](acceptance_criteria.en.md#ac-003).

## REQ-004

Accounting starts on a selected date; opening balances are separate from income and expenses.

Source: `D-03`. Acceptance: [AC-004](acceptance_criteria.en.md#ac-004).

## REQ-005

Accounts distinguish owned, available, locked and borrowed amounts where the source provides them.

Source: `D-02`. Acceptance: [AC-005](acceptance_criteria.en.md#ac-005).

## REQ-006

Transfers between household accounts, including different members’ accounts, change balances without principal income or expense.

Source: `D-04`. Acceptance: [AC-006](acceptance_criteria.en.md#ac-006).

## REQ-007

Exchange and P2P conversion of owned money preserve both currency amounts, the actual rate and fees.

Source: `D-04`. Acceptance: [AC-007](acceptance_criteria.en.md#ac-007).

## REQ-008

Repeated imports, receipts and chat entries combine evidence of one transaction without double counting.

Source: `D-04`. Acceptance: [AC-008](acceptance_criteria.en.md#ac-008).

## REQ-009

Pending, posted, cancelled and refunded transaction states are explicit.

Source: `D-04`. Acceptance: [AC-009](acceptance_criteria.en.md#ac-009).

## REQ-010

A refund reduces expenses in the purchase month while preserving the actual cash receipt date.

Source: `D-05`. Acceptance: [AC-010](acceptance_criteria.en.md#ac-010).

## REQ-011

An expense is recognized in full on payment, including annual subscriptions.

Source: `D-05`. Acceptance: [AC-011](acceptance_criteria.en.md#ac-011).

## REQ-012

Accounting corrections preserve the original, actor, reason, version and ability to undo a decision.

Source: `D-04`. Acceptance: [AC-012](acceptance_criteria.en.md#ac-012).

## REQ-013

Ledger/source balance discrepancies are investigated without hidden automatic balancing.

Source: `D-04`. Acceptance: [AC-013](acceptance_criteria.en.md#ac-013).

## REQ-014

Category, subcategory, merchant and receipt item are separate analytical dimensions.

Source: `D-06`. Acceptance: [AC-014](acceptance_criteria.en.md#ac-014).

## REQ-015

Receipt submission requires a debit account; the photo/PDF stays linked to the processing result.

Source: `D-07`. Acceptance: [AC-015](acceptance_criteria.en.md#ac-015).

## REQ-016

Receipt items allocate one paid amount across categories without duplicating the total.

Source: `D-07`. Acceptance: [AC-016](acceptance_criteria.en.md#ac-016).

## REQ-017

Chat records an established transaction, clarifies missing data and explicitly explains skipped irrelevant documents.

Source: `D-07`. Acceptance: [AC-017](acceptance_criteria.en.md#ac-017).

## REQ-018

Every new or materially changed transaction receives AI review of its version.

Source: `D-08`. Acceptance: [AC-018](acceptance_criteria.en.md#ac-018).

## REQ-019

AI automates internal accounting through validated commands; uncertainty remains explicit.

Source: `D-08`. Acceptance: [AC-019](acceptance_criteria.en.md#ac-019).

## REQ-020

AI income/expense insights reference verifiable data and separate forecasts from facts.

Source: `D-08`. Acceptance: [AC-020](acceptance_criteria.en.md#ac-020).

## REQ-021

AI changes an approved budget, income forecast or goals only on an explicit decision by a member authorized for the change.

Source: `D-09`. Acceptance: [AC-021](acceptance_criteria.en.md#ac-021).

## REQ-022

Full transactions and receipts may be sent to OpenAI; access secrets and unrelated private data are excluded.

Source: `D-08`. Acceptance: [AC-022](acceptance_criteria.en.md#ac-022).

## REQ-023

One household budget covers a calendar month in native currencies with individual views.

Source: `D-09`. Acceptance: [AC-023](acceptance_criteria.en.md#ac-023).

## REQ-024

The plan supports dated obligations and recurring payments.

Source: `D-09`. Acceptance: [AC-024](acceptance_criteria.en.md#ac-024).

## REQ-025

Flexible categories limit monthly spending and show remaining allowance and overspend.

Source: `D-09`. Acceptance: [AC-025](acceptance_criteria.en.md#ac-025).

## REQ-026

Forecast income has an amount, currency, date and separate fulfillment state.

Source: `D-09`. Acceptance: [AC-026](acceptance_criteria.en.md#ac-026).

## REQ-027

Plans can be copied; prior-month remaining allowances and overspend do not roll over automatically.

Source: `D-09`. Acceptance: [AC-027](acceptance_criteria.en.md#ac-027).

## REQ-028

A personal or joint goal has an amount, currency, deadline and funding mode: an explicit reservation or dedicated account.

Source: `D-10`. Acceptance: [AC-028](acceptance_criteria.en.md#ac-028).

## REQ-029

The same money cannot be reserved for multiple goals or counted again through a dedicated account.

Source: `D-10`. Acceptance: [AC-029](acceptance_criteria.en.md#ac-029).

## REQ-030

Daily limits show household and individual available/forecast allowances, by category and with separate funding in each currency.

Source: `D-10`. Acceptance: [AC-030](acceptance_criteria.en.md#ac-030).

## REQ-031

Credit cards show debt, own funds, credit limit, minimum payment and due date from source data.

Source: `D-11`. Acceptance: [AC-031](acceptance_criteria.en.md#ac-031).

## REQ-032

Grace-period tracking uses the specific card's terms and shows the amount and deadline needed to preserve the benefit.

Source: `D-11`. Acceptance: [AC-032](acceptance_criteria.en.md#ac-032).

## REQ-033

Savings show actual accruals and forecasts using rates, terms, compounding and cash flows.

Source: `D-11`. Acceptance: [AC-033](acceptance_criteria.en.md#ac-033).

## REQ-034

Investment returns are compared using dated cash flows and valuation currency.

Source: `D-11`. Acceptance: [AC-034](acceptance_criteria.en.md#ac-034).

## REQ-035

Trading analytics separates realized P&L, unrealized P&L, fees and funding.

Source: `D-11`. Acceptance: [AC-035](acceptance_criteria.en.md#ac-035).

## REQ-036

Mining rewards are separate from transfers between owned wallets.

Source: `D-11`. Acceptance: [AC-036](acceptance_criteria.en.md#ac-036).

## REQ-037

Historical expenses use a fixed transaction-date valuation; current wealth uses a current valuation.

Source: `D-12`. Acceptance: [AC-037](acceptance_criteria.en.md#ac-037).

## REQ-038

Exchange quotes include direction, provider, timestamp, applicable amount and known fees.

Source: `D-12`. Acceptance: [AC-038](acceptance_criteria.en.md#ac-038).

## REQ-039

Missing rates and unsupported assets never become zero amounts or assumed USD/USDT/USDC parity.

Source: `D-36`. Acceptance: [AC-039](acceptance_criteria.en.md#ac-039).

## REQ-040

Each source refreshes hourly and on demand with a visible last-success timestamp.

Source: `D-03`. Acceptance: [AC-040](acceptance_criteria.en.md#ac-040).

## REQ-041

History retains coverage boundaries, cursors, gaps and source status.

Source: `D-03`. Acceptance: [AC-041](acceptance_criteria.en.md#ac-041).

## REQ-042

Alfa automatically reads current accounts, debit cards, transactions and cashback under a verified contract.

Source: `D-47`. Acceptance: [AC-042](acceptance_criteria.en.md#ac-042).

## REQ-043

Raiffeisen RBO API reads only the entrepreneur current account: balances, receipts, debits, fees and history (D-35).

Source: `D-35`. Acceptance: [AC-043](acceptance_criteria.en.md#ac-043).

## REQ-044

The Ozon Bank integration automatically reads the debit card and linked main account: balances, transactions and available details under a verified contract. Other Ozon products are deferred until a contract extension.

Source: `D-32`. Acceptance: [AC-044](acceptance_criteria.en.md#ac-044).

## REQ-045

Bybit automatically reads Funding USDT/USDC/ETH/BTC, used Easy Earn and P2P; the official API is preferred. Other products are deferred without blocking under D-36.

Source: `D-36`. Acceptance: [AC-045](acceptance_criteria.en.md#ac-045).

## REQ-046

Aifory Pro automatically reads RUB accounts, USDT, ETH and the existing crypto card, including these products’ movements and fees. Other products are deferred and do not block the MVP.

Source: `D-33`. Acceptance: [AC-046](acceptance_criteria.en.md#ac-046).

## REQ-047

The EMCD integration automatically reads used crypto cards, Coinhold/Grow, the USDT wallet and historical P2P orders under D-34; mining has never been used and is deferred with other unused products without blocking readiness.

Source: `D-34`. Acceptance: [AC-047](acceptance_criteria.en.md#ac-047).

## REQ-048

Integrations and the browser collector perform authorized read operations only.

Source: `D-13`. Acceptance: [AC-048](acceptance_criteria.en.md#ac-048).

## REQ-049

Each member signs in with their own passkeys and one-time recovery codes; partner-assisted reset is unavailable.

Source: `D-14`. Acceptance: [AC-049](acceptance_criteria.en.md#ac-049).

## REQ-050

Files, source keys, sessions and financial records are protected against unauthorized access.

Source: `D-14`. Acceptance: [AC-050](acceptance_criteria.en.md#ac-050).

## REQ-051

AI is limited to $50/month and degrades to a waiting queue without stopping ordinary accounting.

Source: `D-15`. Acceptance: [AC-051](acceptance_criteria.en.md#ac-051).

## REQ-052

The dashboard combines accounts, plan/actuals, income, expenses, goals and daily limits with drill-down.

Source: `D-16`. Acceptance: [AC-052](acceptance_criteria.en.md#ac-052).

## REQ-053

Reminders and summaries are available in-app and through authorized web push.

Source: `D-16`. Acceptance: [AC-053](acceptance_criteria.en.md#ac-053).

## REQ-054

UI, chat and documentation support RU/EN without changing financial semantics.

Source: `D-16`. Acceptance: [AC-054](acceptance_criteria.en.md#ac-054).

## REQ-055

The web app targets macOS laptops in Chrome and Arc; window resizing and zoom preserve daily accounting access.

Source: `D-01`. Acceptance: [AC-055](acceptance_criteria.en.md#ac-055).

## REQ-056

Deployment fits $40/month for a server in DE/NL/BG; no separately paid data sources are used.

Source: `D-15`. Acceptance: [AC-056](acceptance_criteria.en.md#ac-056).

## REQ-057

An encrypted backup is pulled to the MacBook hourly while reachable; recovery is tested.

Source: `D-15`. Acceptance: [AC-057](acceptance_criteria.en.md#ac-057).

## REQ-058

Operational status exposes import, AI, FX, backup failures and spend without leaking financial content.

Source: `D-15`. Acceptance: [AC-058](acceptance_criteria.en.md#ac-058).

## REQ-059

Money calculations use exact arithmetic and explicit boundary rounding rules.

Source: `D-17`. Acceptance: [AC-059](acceptance_criteria.en.md#ac-059).

## REQ-060

Receipt text, bank descriptions and AI outputs cannot expand agent authority.

Source: `D-08`. Acceptance: [AC-060](acceptance_criteria.en.md#ac-060).

## REQ-061

Repeated jobs, restarts and concurrent changes cannot create duplicate financial effects.

Source: `D-17`. Acceptance: [AC-061](acceptance_criteria.en.md#ac-061).

## REQ-062

Architecture uses Go/PostgreSQL, React/TypeScript/Vite and a separate Playwright collector with dependencies pointing toward the domain.

Source: `D-17`. Acceptance: [AC-062](acceptance_criteria.en.md#ac-062).

## REQ-063

User, household and membership are separate models; the two-member limit is configured.

Source: `D-18`. Acceptance: [AC-077](acceptance_criteria.en.md#ac-077).

## REQ-064

Both members see all financial data and edit transactions; only the owner edits personal goals and plan portions.

Source: `D-19`. Acceptance: [AC-078](acceptance_criteria.en.md#ac-078).

## REQ-065

Account ownership, external-account owner, record author and expense attribution are distinct dimensions.

Source: `D-20`. Acceptance: [AC-079](acceptance_criteria.en.md#ac-079).

## REQ-066

All income and available funds enter the household pool; household and individual budget views share one financial fact.

Source: `D-21`. Acceptance: [AC-080](acceptance_criteria.en.md#ac-080).

## REQ-067

Expenses and receipt items have personal or joint attribution; joint shares default to 50/50 with line or purchase overrides.

Source: `D-22`. Acceptance: [AC-081](acceptance_criteria.en.md#ac-081).

## REQ-068

An inter-member debt is recorded only explicitly and does not increase household assets or expenses.

Source: `D-23`. Acceptance: [AC-082](acceptance_criteria.en.md#ac-082).

## REQ-069

Personal and joint goal reservations are explicit; joint goals appear in a separate shared block without personal shares.

Source: `D-24`. Acceptance: [AC-083](acceptance_criteria.en.md#ac-083).

## REQ-070

Individual daily allowances sum to no more than the household ceiling in one currency; the payer’s account does not change expense shares.

Source: `D-25`. Acceptance: [AC-084](acceptance_criteria.en.md#ac-084).

## REQ-071

One shared chat retains message authors and checks the AI command initiator’s authority at execution.

Source: `D-26`. Acceptance: [AC-085](acceptance_criteria.en.md#ac-085).

## REQ-072

Competing edits, clarification answers and reversals check revision and current permissions while retaining both authors.

Source: `D-27`. Acceptance: [AC-086](acceptance_criteria.en.md#ac-086).

## REQ-073

Both manage connections; the external-account owner performs bank authentication without exposing secrets to the partner or AI.

Source: `D-28`. Acceptance: [AC-087](acceptance_criteria.en.md#ac-087).

## REQ-074

Plan and goal changes notify the other member; read state and push subscriptions belong to the individual user.

Source: `D-29`. Acceptance: [AC-088](acceptance_criteria.en.md#ac-088).

## REQ-075

Data recovery preserves users, memberships, ownership, roles, history and shared household accounting.

Source: `D-18`. Acceptance: [AC-089](acceptance_criteria.en.md#ac-089).

## REQ-076

Household scope is checked for APIs, files, AI, jobs and external IDs independently of supplied actor/owner fields.

Source: `D-18`. Acceptance: [AC-090](acceptance_criteria.en.md#ac-090).

## REQ-077

Dark Want Keep tokens: restrained cyberpunk and Middle Eastern architectural rhythm.

Source: `D-30`. Acceptance: [AC-094](acceptance_criteria.en.md#ac-094).

## REQ-078

Pixelify Sans serves branding and large accents; Manrope serves everyday UI.

Source: `D-30`. Acceptance: [AC-095](acceptance_criteria.en.md#ac-095).

## REQ-079

Project-owned shadcn/ui components on Base UI use custom semantic tokens.

Source: `D-30`. Acceptance: [AC-096](acceptance_criteria.en.md#ac-096).

## REQ-080

Each screen answers a user question and leads to a useful next action.

Source: `D-30`. Acceptance: [AC-097](acceptance_criteria.en.md#ac-097).

## REQ-081

Desktop navigation preserves context and changing household views never changes authority.

Source: `D-30`. Acceptance: [AC-098](acceptance_criteria.en.md#ac-098).

## REQ-082

Screen states explain consequences and a safe next step without losing input.

Source: `D-30`. Acceptance: [AC-099](acceptance_criteria.en.md#ac-099).

## REQ-083

Sign-in preserves the supplied reference composition and personal access recovery.

Source: `D-30`. Acceptance: [AC-100](acceptance_criteria.en.md#ac-100).

## REQ-084

Accessibility is checked in actual Chrome and Arc, including keyboard, focus, contrast, zoom and reduced motion.

Source: `D-30`. Acceptance: [AC-101](acceptance_criteria.en.md#ac-101).

## REQ-085

Seven core user tasks are completed without developer help with explainable financial outcomes.

Source: `D-30`. Acceptance: [AC-102](acceptance_criteria.en.md#ac-102).

## REQ-086

Account details depend on the product and show availability before technical details.

Source: `D-30`. Acceptance: [AC-103](acceptance_criteria.en.md#ac-103).

## REQ-087

Contextual pixel animations acknowledge milestones and warn about limits while preserving accessibility and truthful outcomes.

Source: `D-31`. Acceptance: [AC-104](acceptance_criteria.en.md#ac-104).

## REQ-088

Provider sync is allowed only by a current server-side admission bound to verified adapter, contract, allowlist, configuration, operator-permission and environment revisions.

Source: `D-43`. Acceptance: [AC-106](acceptance_criteria.en.md#ac-106).
