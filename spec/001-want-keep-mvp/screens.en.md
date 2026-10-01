# Want Keep screens, forms and states

Rendered from [catalog.json](catalog.json). Edit the catalog, then run `python3 spec/001-want-keep-mvp/tools/spec_tool.py render`.

[Design](design.en.md) · [Navigation](navigation.en.md)

Routes are future UI contracts. Listed states are tested under the relevant failure/action; omitted states are inapplicable to that screen. Shared states apply per section, preserving working parts. Forms inherit saving/unknown/conflict/permission/session/error/offline/success according to command type. Resizing/zoom preserves desktop navigation. Exact API payloads follow contracts and Ready; UI never invents provider fields.

## State catalog

### UISTATE-01 — Loading

Structural skeleton and loading label; amounts are never replaced by zero.

### UISTATE-02 — Refreshing

Keep previous data/context and last-success time; block only conflicting actions.

### UISTATE-03 — Empty

Explain the useful outcome and offer a first account, receipt, plan or goal action.

### UISTATE-04 — No matches

Keep filters, explain no results and offer to clear conditions.

### UISTATE-05 — Partial data

Name the missing source/period and its effect on the amount; available sections work and unknowns stay explicit.

### UISTATE-06 — Stale data

Show last-success date and impact on the decision; offer refresh or connection details.

### UISTATE-07 — Error

Plain cause and next step beside the affected section; preserve input/healthy data and expand diagnostics separately.

### UISTATE-08 — Offline

Show missing connectivity and do not promise saved data. Sensitive drafts remain only in current-tab memory, without a new offline queue.

### UISTATE-09 — Saving

Immediately show current-action progress and prevent duplicate command submission.

### UISTATE-10 — Unknown outcome

Keep command ID/input and query its result; never blindly create another financial command. After reload reconcile the server list of recent commands.

### UISTATE-11 — Version conflict

Show authors/differences and keep my input; load current version and allow chosen changes to be reapplied after validation.

### UISTATE-12 — Insufficient permission

Household can read financial data; forbidden edits explain ownership. Server rejects the command regardless of button visibility.

### UISTATE-13 — Session expired

Hide protected contents; require the same member to sign in and return through a safe internal route. Another identity never receives the prior draft.

### UISTATE-14 — AI waiting

Distinguish queued, processing, clarification and budget/API pause; ordinary accounting remains available and results are not invented.

### UISTATE-15 — Bank sign-in needed

Name connection and owner, offer safe owner sign-in; partner sees waiting without secret access.

### UISTATE-16 — Confirmed

After server-confirmed outcome show what changed, an object link and available correction; do not rely on a disappearing toast.

### UISTATE-17 — Cancelled

Explain that no new outcome was confirmed and offer explicit retry; cancelled system passkey prompts are not a malfunction.

## Screen catalog

### SCR-001 — Sign in

`/login`

**Question:** How do I sign in?

**Primary answer:** One sign-in with a personal passkey.

**Top-down structure:** Center: logo, Want Keep, whitespace, primary button; subtle language/help.

**Next action:** Normal sign-in → SCR-006; same-member reauthentication after expiry restores an allowed internal route. Help → SCR-002.

**Explanation and details:** System prompt, local error reason and retry; do not copy export dimensions.

**Permissions:** Before sign-in only own authentication; financial data hidden.

Forms: FORM-01.

States: UISTATE-01, UISTATE-07, UISTATE-08, UISTATE-09, UISTATE-10, UISTATE-16, UISTATE-17.

REQ: REQ-049, REQ-083. AC: AC-049, AC-100.

Task: [task-7.1](tasks/task-7.1.md).

### SCR-002 — Recovery

`/recovery`

**Question:** How do I regain my access?

**Primary answer:** Personal single-use code and a new passkey.

**Top-down structure:** Explanation → code → verification → new passkey → new codes/outcome.

**Next action:** Recover own access, then SCR-006; cancel → SCR-001.

**Explanation and details:** This user’s old sessions revoked; invalid/used/expired code without exposing another identity.

**Permissions:** Before sign-in only own authentication; financial data hidden.

Forms: FORM-01.

States: UISTATE-01, UISTATE-07, UISTATE-08, UISTATE-09, UISTATE-10, UISTATE-16, UISTATE-17.

REQ: REQ-049, REQ-076. AC: AC-049, AC-090.

Task: [task-7.1](tasks/task-7.1.md).

### SCR-003 — Initial setup

`/setup`

**Question:** How do we start our household?

**Primary answer:** Restricted first-member setup.

**Top-down structure:** Operator token, name/household, locale, timezone and asset → own passkey → one-time recovery codes → SCR-005.

**Next action:** Create household → SCR-005; after an unknown response check /me, then sign in with the created passkey.

**Explanation and details:** Repeat bootstrap is closed; no public signup.

**Permissions:** Before sign-in only own authentication; financial data hidden.

Forms: FORM-02.

States: UISTATE-01, UISTATE-07, UISTATE-08, UISTATE-09, UISTATE-10, UISTATE-12, UISTATE-16, UISTATE-17.

REQ: REQ-001, REQ-063. AC: AC-001, AC-077.

Task: [task-7.1](tasks/task-7.1.md).

### SCR-004 — Invitation

`/invite`

**Question:** How does my partner join?

**Primary answer:** Verified private invitation to a specific household.

**Top-down structure:** Invitation check → household/inviter → name/language/currency → own passkey → own recovery codes.

**Next action:** Accept → SCR-005; expired invitation explains obtaining a new one from the member.

**Explanation and details:** Expiry, use, revocation and a full household are distinct. An already signed-in user signs out first. After a lost registration response, sign in with the new passkey and regenerate codes after fresh own authentication if needed.

**Permissions:** Before sign-in only own authentication; financial data hidden.

Forms: FORM-02.

States: UISTATE-01, UISTATE-07, UISTATE-08, UISTATE-09, UISTATE-10, UISTATE-12, UISTATE-16, UISTATE-17.

REQ: REQ-001, REQ-063, REQ-076. AC: AC-001, AC-077, AC-090.

Task: [task-7.1](tasks/task-7.1.md).

### SCR-005 — Onboarding

`/onboarding`

**Question:** How do I get a useful first overview?

**Primary answer:** Start with cash and accounts already available.

**Top-down structure:** Household members and available accounts → cash account with exact opening funds/date → partner invitation or overview. Connections and the first plan are added by their owning tasks.

**Next action:** Add FORM-03/13 or continue with available data → SCR-006.

**Explanation and details:** Onboarding remains accessible without a completion flag. Invitation and future connections do not block continuation. Account timeout reconciles the original UUID; reload checks recent commands.

**Permissions:** Both members can read; server checks membership and resource ownership for actions.

Forms: FORM-03, FORM-13.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-14, UISTATE-15.

REQ: REQ-004, REQ-040, REQ-041. AC: AC-004, AC-040, AC-041.

Task: [task-7.1](tasks/task-7.1.md).

### SCR-006 — Overview

`/overview`

**Question:** How much can I spend and cover obligations?

**Primary answer:** Available daily allowance by currency and next shortfall risk.

**Top-down structure:** Allowance/next payments → plan/actual → attention → goals → funds. Household/member, month and currency stay visible.

**Next action:** Explain amount → SCR-017; obligations → SCR-016; clarification → SCR-025.

**Explanation and details:** Formula/sources/period/history → transactions; forecast separate from available, current revaluation separate.

**Permissions:** Both members can read; server checks membership and resource ownership for actions.

Forms: —.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-14, UISTATE-15, UISTATE-16.

REQ: REQ-030, REQ-052, REQ-066, REQ-070, REQ-080. AC: AC-030, AC-052, AC-080, AC-084, AC-097.

Task: [task-7.6](tasks/task-7.6.md).

### SCR-007 — Money and accounts

`/accounts`

**Question:** Where is the money and how much is available?

**Primary answer:** Own funds, available, reserved and debt by currency.

**Top-down structure:** Summary → personal/household account groups → available/reserved/blocked/debt → freshness.

**Next action:** Open SCR-008, add account FORM-03 or connect SCR-029; transactions → SCR-009.

**Explanation and details:** Equivalent does not fund another currency; accounts outside a view filter remain in the household pool.

**Permissions:** Both members read/correct facts for any household account; actor from session.

Forms: FORM-03.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-04, UISTATE-15.

REQ: REQ-002, REQ-003, REQ-005, REQ-065, REQ-087. AC: AC-002, AC-003, AC-005, AC-079, AC-104, AC-105.

Task: [task-7.2](tasks/task-7.2.md), [task-7.15](tasks/task-7.15.md).

### SCR-008 — Account details

`/accounts/:id`

**Question:** What is available in this account?

**Primary answer:** The answer depends on product; availability and obligations come first.

**Top-down structure:** Current: available/reserved/blocked. Credit: debt, required payment/date, grace-preserving amount, own funds. Deposit/Earn/Coinhold: actual/forecast, maturity/withdrawal terms. Crypto: assets/positions, available/margin/blocked, P&L/fees/funding/mining. Then transactions.

**Next action:** Resolve balance → SCR-012; returns → SCR-021/022; record movement FORM-05.

**Explanation and details:** Terms, source, valuation date and history expand; never invent unknown grace/withdrawal terms.

**Permissions:** Both members read/correct facts for any household account; actor from session.

Forms: FORM-03, FORM-05, FORM-06.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-04, UISTATE-15.

REQ: REQ-005, REQ-031, REQ-033, REQ-035, REQ-086, REQ-087. AC: AC-005, AC-031, AC-033, AC-035, AC-103, AC-104, AC-105.

Task: [task-7.2](tasks/task-7.2.md), [task-7.7](tasks/task-7.7.md), [task-7.15](tasks/task-7.15.md).

### SCR-009 — Transactions

`/transactions`

**Question:** Where did money go and is everything recorded?

**Primary answer:** Transaction purpose and a meaningful selected-period total.

**Top-down structure:** Search/month/account/payer/allocation/category/merchant/item/status → totals → date/merchant/amount/share rows.

**Next action:** Open SCR-010; enter FORM-04 or receipt in SCR-024.

**Explanation and details:** Category, merchant and item filters are independent; uncategorized expense remains visible. Items form one payment without a duplicate total. Member amounts and unallocated are a MembershipID view of one household fact; payer, owner and actor are separate. Transfers/exchanges are marked as movements excluded from income/expense; filters never change accounting semantics.

**Permissions:** Both members read/correct facts for any household account; actor from session.

Forms: FORM-04, FORM-05.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-04, UISTATE-14.

REQ: REQ-006, REQ-008, REQ-014, REQ-067. AC: AC-006, AC-008, AC-014, AC-081.

Task: [task-7.2](tasks/task-7.2.md), [task-2.6](tasks/task-2.6.md), [task-2.8](tasks/task-2.8.md).

### SCR-010 — Transaction details

`/transactions/:id`

**Question:** Is this purchase accounted for correctly?

**Primary answer:** Amount, category, merchant, items, allocation and payer for one transaction.

**Top-down structure:** Accounting outcome → account/date/status → category/merchant → item gross/discount/net → shares → receipt → correct/undo/refund.

**Next action:** Correct FORM-06/07, refund FORM-08, explicit debt FORM-09; receipt → SCR-011.

**Explanation and details:** Before/after history exposes actor, decisionId, source and protected fields. Allocation shows personal/shared purpose, exact member amounts, unallocated, applied rule revisions and items. Explicit item value wins over purchase, then merchant/category rule and equal for an explicitly shared expense. A source update cannot erase the user choice; changing an amount-based allocation requires a consistent correction while share-based allocation recalculates. Matching retains one household and member effect carrier; undo checks every participant revision.

**Permissions:** Both members read/correct facts for any household account; actor from session.

Forms: FORM-06, FORM-07, FORM-08, FORM-09, FORM-05.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-14.

REQ: REQ-010, REQ-012, REQ-067, REQ-072, REQ-006, REQ-007, REQ-008, REQ-014, REQ-016. AC: AC-010, AC-012, AC-081, AC-086, AC-006, AC-007, AC-008, AC-093, AC-014, AC-016.

Task: [task-7.2](tasks/task-7.2.md), [task-2.3](tasks/task-2.3.md), [task-2.4](tasks/task-2.4.md), [task-2.6](tasks/task-2.6.md), [task-2.8](tasks/task-2.8.md).

### SCR-011 — Receipt

`/receipts/:id`

**Question:** What was extracted and which purchase does it belong to?

**Primary answer:** One transaction link or a specific clarification.

**Top-down structure:** Created/found/question outcome → account → original beside items → amounts/shares.

**Next action:** Clarify/allocate FORM-07/12; open SCR-010.

**Explanation and details:** Item gross, discount and net explain one payment; a receipt-wide discount is allocated deterministically. Incomplete discounts or a mismatch require clarification without an invented item. Uncertain fields are labelled, unsuitable documents are explained and unsafe content is never executed. OCR/PDF remains task-5.3.

**Permissions:** Both members read/correct facts for any household account; actor from session.

Forms: FORM-07, FORM-12.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-14.

REQ: REQ-015, REQ-016, REQ-017, REQ-060, REQ-067. AC: AC-015, AC-016, AC-017, AC-060, AC-081.

Task: [task-7.3](tasks/task-7.3.md), [task-2.6](tasks/task-2.6.md).

### SCR-012 — Balance reconciliation

`/accounts/:id/reconciliation`

**Question:** Why does the balance differ?

**Primary answer:** Source and ledger owned, available, locked and debt at sourceAsOf, their exact differences, data quality and evidence-backed explanations.

**Top-down structure:** Lifecycle/result and freshness → four source/ledger/difference components → explanations and related transactions → replay → resolution.

**Next action:** Start/wait for bounded replay, reauthenticate the source, open related movements in SCR-010 or, after completed/unavailable replay, explicitly adjust owned/debt.

**Explanation and details:** Unknown is not zero; balanced stale does not become fresh. Available/locked cannot be adjusted directly. An explicit adjustment is not income/expense, never changes the source observation and previews the server-derived effect.

**Permissions:** Both members read/correct facts for any household account; actor from session.

Forms: FORM-06.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-15.

REQ: REQ-004, REQ-013, REQ-041. AC: AC-004, AC-013, AC-041.

Task: [task-2.5](tasks/task-2.5.md), [task-7.2](tasks/task-7.2.md).

### SCR-013 — Reimbursements

`/reimbursements`

**Question:** Are any reimbursements outstanding?

**Primary answer:** Explicit debts, outstanding amounts and linked-transfer state.

**Top-down structure:** Open and attention-required debts → debtor/creditor/currency/outstanding → reason and source expense → active or stale settlements → decision history.

**Next action:** Create debt, link a posted transfer, correct data or undo a selected decision in FORM-09; open the source transaction in SCR-010.

**Explanation and details:** Allocation, payer and transfer never create debt by themselves. Fees do not reduce outstanding. Cross-asset settlement shows both native amounts. A changed expense or transfer requires an explicit decision.

**Permissions:** Both members read/correct facts for any household account; actor from session.

Forms: FORM-09.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-04.

REQ: REQ-006, REQ-068. AC: AC-006, AC-082.

Task: [task-7.2](tasks/task-7.2.md).

### SCR-014 — Monthly plan

`/plan`

**Question:** How should we allocate money this month?

**Primary answer:** Plan/actual and funding for each currency.

**Top-down structure:** Month/household/member → expected income → obligations → flexible spending → personal/shared → shortfall/remainder.

**Next action:** Edit SCR-015, copy month, calendar SCR-016, limits SCR-017.

**Explanation and details:** Expected income is not received; copying never carries old remainder, currency equivalent never covers a shortfall.

**Permissions:** Both read; only the owner edits personal resources, either member edits shared resources.

Forms: FORM-10.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-14.

REQ: REQ-023, REQ-026, REQ-027, REQ-064, REQ-066. AC: AC-023, AC-026, AC-027, AC-078, AC-080.

Task: [task-7.4](tasks/task-7.4.md).

### SCR-015 — Plan editor

`/plan/edit`

**Question:** What changes after my decision?

**Primary answer:** Allowance and funding preview before saving.

**Top-down structure:** Current revision → lines/income/dates/shares → comparison → confirm.

**Next action:** Add/edit FORM-10; save → SCR-014, cancel with unsaved-input protection.

**Explanation and details:** Partner personal lines are read-only; shared plan changes notify the partner.

**Permissions:** Both read; only the owner edits personal resources, either member edits shared resources.

Forms: FORM-10, FORM-12.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-14, UISTATE-17.

REQ: REQ-021, REQ-024, REQ-025, REQ-064, REQ-072. AC: AC-021, AC-024, AC-025, AC-078, AC-086.

Task: [task-7.4](tasks/task-7.4.md).

### SCR-016 — Budget calendar

`/plan/calendar`

**Question:** When is money due or expected?

**Primary answer:** Upcoming dates with currency shortfall risk.

**Top-down structure:** Month → calendar and accessible agenda list → income/payments → fulfillment/link.

**Next action:** Open line FORM-10 or linked SCR-010, view SCR-017.

**Explanation and details:** Future date creates no actual transaction; overdue/fulfilled remain distinct.

**Permissions:** Both read; only the owner edits personal resources, either member edits shared resources.

Forms: FORM-10.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-04.

REQ: REQ-024, REQ-026, REQ-031. AC: AC-024, AC-026, AC-031.

Task: [task-7.4](tasks/task-7.4.md).

### SCR-017 — Daily allowances

`/plan/limits`

**Question:** How much today and why?

**Primary answer:** Available and forecast variants by currency/category/member.

**Top-down structure:** Available today → separate forecast → remaining days → obligations/reserves/actual → personal allocation.

**Next action:** Expand formula, change own plan SCR-015 or reserve SCR-019.

**Explanation and details:** Personal allowances sum to no more than household cap; shortfall explicit and missing inputs never become zero.

**Permissions:** Both members can read; server checks membership and resource ownership for actions.

Forms: —.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-14.

REQ: REQ-030, REQ-069, REQ-070, REQ-087. AC: AC-030, AC-083, AC-084, AC-104.

Task: [task-7.4](tasks/task-7.4.md), [task-7.6](tasks/task-7.6.md), [task-7.15](tasks/task-7.15.md).

### SCR-018 — Goals

`/goals`

**Question:** How are our goals progressing?

**Primary answer:** Personal goal progress and a separate joint-goal block.

**Top-down structure:** Joint goals → personal groups → deadline/progress/reserve → available to reserve by currency.

**Next action:** Create FORM-11 or open SCR-019.

**Explanation and details:** Joint reserve counted once, joint goals have no personal shares.

**Permissions:** Both read; only the owner edits personal resources, either member edits shared resources.

Forms: FORM-11.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-04.

REQ: REQ-028, REQ-029, REQ-069. AC: AC-028, AC-029, AC-083.

Task: [task-7.5](tasks/task-7.5.md).

### SCR-019 — Goal details

`/goals/:id`

**Question:** How much can we reserve without risking the budget?

**Primary answer:** Progress and proposed reserve impact on available funds.

**Top-down structure:** Goal/deadline → saved and forecast → reserve/dedicated account → change preview → history.

**Next action:** Change reserve FORM-11, open dedicated SCR-008 or budget SCR-014.

**Explanation and details:** Dedicated account never doubles reserve; achievement uses actual data, not forecast.

**Permissions:** Both read; only the owner edits personal resources, either member edits shared resources.

Forms: FORM-11, FORM-12.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-14.

REQ: REQ-028, REQ-029, REQ-069, REQ-072, REQ-087. AC: AC-028, AC-029, AC-083, AC-086, AC-104.

Task: [task-7.5](tasks/task-7.5.md), [task-7.15](tasks/task-7.15.md).

### SCR-020 — Income and expenses

`/analytics`

**Question:** What changed in spending and why?

**Primary answer:** Main deviations and comparable-period comparison.

**Top-down structure:** Conclusion/period/currency/household → plan vs actual and change → categories/merchants/items → chart and table.

**Next action:** Select bar/category → SCR-009 with the same filters.

**Explanation and details:** Refunds restate purchase; transfer principal excluded; incomplete-history periods labelled.

**Permissions:** Both members can read; server checks membership and resource ownership for actions.

Forms: —.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-04.

REQ: REQ-010, REQ-014, REQ-037, REQ-052. AC: AC-010, AC-014, AC-037, AC-052.

Task: [task-7.6](tasks/task-7.6.md).

### SCR-021 — Returns

`/analytics/returns`

**Question:** Which savings earn returns accounting for time?

**Primary answer:** Actual and forecast returns on a common comparison basis.

**Top-down structure:** Period/currency → income/XIRR and limitation → products/cash flows → actual vs forecast.

**Next action:** Compare products, open SCR-008 or specific flows SCR-010.

**Explanation and details:** Dates, fees, FX and solver assumptions expand; undefined/multiple XIRR roots are not zero.

**Permissions:** Both members can read; server checks membership and resource ownership for actions.

Forms: —.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-04.

REQ: REQ-033, REQ-034. AC: AC-033, AC-034.

Task: [task-7.7](tasks/task-7.7.md).

### SCR-022 — Crypto analytics

`/analytics/crypto`

**Question:** What produced the crypto result?

**Primary answer:** Realized/unrealized results separate from fees and mining.

**Top-down structure:** Period/currency/account → results → fees/funding/mining → positions/trades → movements.

**Next action:** Open account SCR-008, explanatory transactions SCR-009/010.

**Explanation and details:** Portfolio valuation/FX separate from trading P&L; unknown cost basis/history explicitly limits result.

**Permissions:** Both members can read; server checks membership and resource ownership for actions.

Forms: —.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-04.

REQ: REQ-035, REQ-036, REQ-037. AC: AC-035, AC-036, AC-037.

Task: [task-7.7](tasks/task-7.7.md).

### SCR-023 — Rates and valuation

`/analytics/rates`

**Question:** Which rate valued the money?

**Primary answer:** Rate source/date/direction and applicability.

**Top-down structure:** Pair/date → reference rate → available provider buy/sell quotes → known fees/limitations.

**Next action:** Inspect historical valuation or linked accounts/transactions; refresh reference data.

**Explanation and details:** No executable-exchange promise; missing quote is not USDT/USD parity; actual transaction rate separate.

**Permissions:** Both members can read; server checks membership and resource ownership for actions.

Forms: —.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-04.

REQ: REQ-003, REQ-037, REQ-038, REQ-039. AC: AC-003, AC-037, AC-038, AC-039.

Task: [task-7.6](tasks/task-7.6.md).

### SCR-024 — Shared chat

`/chat`

**Question:** Help me record and understand

**Primary answer:** Shared conversation and a clear outcome for each input.

**Top-down structure:** Messages with authors → context/clarifications → text/file input and account → linked outcome.

**Next action:** Send text/receipt FORM-07; answer FORM-12; open SCR-010/011/025.

**Explanation and details:** Created/found/waiting/skipped distinct; secrets do not belong here; AI waiting never implies success.

**Permissions:** Both members read/correct facts for any household account; actor from session.

Forms: FORM-07, FORM-12.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-14, UISTATE-17.

REQ: REQ-015, REQ-017, REQ-019, REQ-051, REQ-071. AC: AC-015, AC-017, AC-019, AC-051, AC-085.

Task: [task-7.3](tasks/task-7.3.md).

### SCR-025 — Clarifications

`/chat/clarifications`

**Question:** What needs clarification for correct accounting?

**Primary answer:** A specific question with safe options and context.

**Top-down structure:** Awaiting answer → reason/purchase → choices and free text → preview → status.

**Next action:** Answer FORM-12, open original SCR-010/011; show answered items with author.

**Explanation and details:** Answer conflict preserves input; only owner may answer for a personal goal.

**Permissions:** Both read; only the owner edits personal resources, either member edits shared resources.

Forms: FORM-12.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-04, UISTATE-14.

REQ: REQ-017, REQ-019, REQ-071, REQ-072. AC: AC-017, AC-019, AC-085, AC-086.

Task: [task-7.3](tasks/task-7.3.md).

### SCR-026 — Insights

`/insights`

**Question:** What should change and what supports the suggestion?

**Primary answer:** Brief observation with impact magnitude and verifiable data.

**Top-down structure:** Important findings → reason/period/comparison → proposal → evidence.

**Next action:** Inspect transactions SCR-009; accept authorized change FORM-12 or dismiss.

**Explanation and details:** Incomplete data/limitations labelled; forecast is no promise and AI never silently edits an approved plan.

**Permissions:** Both read; only the owner edits personal resources, either member edits shared resources.

Forms: FORM-12.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-14.

REQ: REQ-020, REQ-021, REQ-051. AC: AC-020, AC-021, AC-051.

Task: [task-7.6](tasks/task-7.6.md).

### SCR-027 — Connections

`/connections`

**Question:** Are data updating and where is action needed?

**Primary answer:** Last successful read and issues per connection.

**Top-down structure:** Needs attention → six-platform accounts/owners → freshness/coverage → add.

**Next action:** Connect SCR-029, open SCR-028, refresh a specific source.

**Explanation and details:** Unconfirmed capability never appears operational; same platform may have distinct accounts.

**Permissions:** Both members can read; server checks membership and resource ownership for actions.

Forms: FORM-13.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-04, UISTATE-15.

REQ: REQ-040, REQ-041, REQ-073. AC: AC-040, AC-041, AC-087.

Task: [task-7.13](tasks/task-7.13.md).

### SCR-028 — Connection status

`/connections/:id`

**Question:** Why are this service’s data incomplete?

**Primary answer:** Plain cause, available period and required action.

**Top-down structure:** Status/owner → last success/next sync → products/history boundaries → actions.

**Next action:** Refresh, reauthorize SCR-029, disconnect with history-preservation preview.

**Explanation and details:** Errors/cursor/diagnostic code expand; partial import never implies complete history.

**Permissions:** Both members can read; server checks membership and resource ownership for actions.

Forms: FORM-13.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-15.

REQ: REQ-040, REQ-041, REQ-048, REQ-073. AC: AC-040, AC-041, AC-048, AC-087.

Task: [task-7.13](tasks/task-7.13.md).

### SCR-029 — Platform authorization

`/connections/new; /connections/:id/reauth`

**Question:** How do I safely connect my account?

**Primary answer:** Read-only connection with credentials supplied by its owner.

**Top-down structure:** Platform/owner → read-access explanation → isolated input → MFA → outcome/history date.

**Next action:** Continue FORM-13 → SCR-028; partner sees waiting for owner.

**Explanation and details:** Expiry/cancel preserves existing ledger; secret never enters chat, logs or shared screen.

**Permissions:** Both manage; only external-account owner enters secrets.

Forms: FORM-13.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-15, UISTATE-17.

REQ: REQ-048, REQ-050, REQ-073. AC: AC-048, AC-050, AC-087.

Task: [task-7.13](tasks/task-7.13.md).

### SCR-030 — Notifications

`/notifications`

**Question:** What matters to me right now?

**Primary answer:** Prioritized events with a concrete action.

**Top-down structure:** Unread/all → shortfall/payment/clarification/change/source → date/author → destination.

**Next action:** Open object; mark read; push settings SCR-031.

**Explanation and details:** Reading is personal; push denial never hides in-app and push omits sensitive details by default.

**Permissions:** Both members can read; server checks membership and resource ownership for actions.

Forms: —.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-04.

REQ: REQ-053, REQ-074. AC: AC-053, AC-088.

Task: [task-7.8](tasks/task-7.8.md).

### SCR-031 — Settings

`/settings`

**Question:** How do I configure comfortable accounting?

**Primary answer:** Personal preferences and clear settings groups.

**Top-down structure:** Language/currency → notifications → household → security → categories → system health.

**Next action:** Save FORM-14; open SCR-032/033/034/035.

**Explanation and details:** Changing language/currency changes neither financial fact nor authority; no theme toggle.

**Permissions:** Both members can read; server checks membership and resource ownership for actions.

Forms: FORM-14.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-17.

REQ: REQ-003, REQ-053, REQ-054, REQ-087. AC: AC-003, AC-053, AC-054, AC-104.

Task: [task-7.14](tasks/task-7.14.md), [task-7.15](tasks/task-7.15.md).

### SCR-032 — Household

`/settings/household`

**Question:** Who belongs and what can each do?

**Primary answer:** Two distinct members with transparent permissions.

**Top-down structure:** Members/invitation → shared-access explanation → personal/shared resources and rules.

**Next action:** When capacity exists, issue/reissue after own passkey confirmation within 5 minutes; revoke with an active session. Show metadata/revision first; never reread the secret.

**Explanation and details:** No role changing, member exit/replacement or partner recovery in MVP.

**Permissions:** Both members can read; server checks membership and resource ownership for actions.

Forms: FORM-02.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16.

REQ: REQ-001, REQ-063, REQ-064. AC: AC-001, AC-077, AC-078.

Task: [task-7.9](tasks/task-7.9.md), [task-7.14](tasks/task-7.14.md).

### SCR-033 — Security

`/settings/security`

**Question:** How do I preserve my access?

**Primary answer:** Own passkeys, recovery and active devices.

**Top-down structure:** Access methods → backup access → devices/sessions → revocation consequences.

**Next action:** Add passkey, rotate codes, revoke own device FORM-14.

**Explanation and details:** Recovery codes appear only in protected own flow, unavailable to partner.

**Permissions:** Own credentials and sessions only.

Forms: FORM-14.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-17.

REQ: REQ-049, REQ-050, REQ-076. AC: AC-049, AC-050, AC-090.

Task: [task-7.14](tasks/task-7.14.md).

### SCR-034 — Categories and rules

`/settings/categories`

**Question:** How do we reduce manual clarifications?

**Primary answer:** Clear categories, merchants and expense-allocation rules.

**Top-down structure:** Categories/subcategories and archive → merchants/confirmed aliases → rules/priority → example preview.

**Next action:** Create/correct FORM-15; inspect affected transactions SCR-009.

**Explanation and details:** Starter labels use stable RU/EN keys and a custom name is not translated. A merchant never becomes a subcategory; archival preserves history. Rules use merchant/category AND, priority and active-member shares; preview explains selected revisions or rule_conflict. A new rule affects only new facts. An AI proposal requires user confirmation; a personal/shared rule never edits a partner personal plan.

**Permissions:** Both members read/correct facts for any household account; actor from session.

Forms: FORM-15.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-04.

REQ: REQ-014, REQ-019, REQ-067. AC: AC-014, AC-019, AC-081.

Task: [task-7.14](tasks/task-7.14.md), [task-2.6](tasks/task-2.6.md), [task-2.8](tasks/task-2.8.md).

### SCR-035 — Accounting and backup health

`/system`

**Question:** Is accounting working and data preserved?

**Primary answer:** What works, how fresh the backup is and what needs action.

**Top-down structure:** Critical issues → last successful backup/age → AI/spend limit → sources/rates → diagnostics.

**Next action:** Open affected SCR-028/025, follow Mac connection runbook instructions; refresh status.

**Explanation and details:** Conditional RPO follows backup age; offline Mac never implies a fresh copy. No database-restore button; separate operator procedure.

**Permissions:** Both members can read; server checks membership and resource ownership for actions.

Forms: —.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-14, UISTATE-15.

REQ: REQ-051, REQ-056, REQ-057, REQ-058. AC: AC-051, AC-056, AC-057, AC-058.

Task: [task-7.14](tasks/task-7.14.md).

## Form catalog

#### FORM-01 — Passkey and personal recovery

**Fields:** System passkey prompt; recovery: personal code followed by new passkey enrollment.

**Validation and permissions:** Verified challenge/origin/RP; code is hidden and single-use. Partner cannot recover access. No email/password fallback.

**Outcome:** Current-member session; recovery revokes only their old sessions; cancellation returns to sign-in.

#### FORM-02 — Household setup and invitation

**Fields:** Member/household name, language, timezone/currency; private invitation, joining member name and passkey.

**Validation and permissions:** Operator bootstrap is one-time. One random invitation lasts 24 hours; issue/reissue requires own authentication within 5 minutes and expectedRevision. Revocation uses CSRF without fresh auth. Acceptance checks browser/purpose/revision, inviter membership and capacity atomically with a new passkey/session/codes.

**Outcome:** Membership created, personal recovery codes shown; proceed to onboarding. Secrets excluded from URL logs/analytics.

#### FORM-03 — Account and opening balance

**Fields:** Name, asset, personal/household cash account, start date in household timezone and exact opening balance. Imported products have separate owned/available/locked/debt, confirmation and provenance; cards are balance-free aliases.

**Validation and permissions:** Both members create accounts and correct accounting facts. Changing owner or personal/household scope of an existing personal account requires its current owner; either member may change a household account. This never changes verified external ownership or transaction history. Exact decimals and currency required. Imported fields change through correction; opening balance is not income. Manual creation is cash only; a personal account is created for oneself. Moving the date keeps earlier operations in history; the new opening replaces the previous calculation input. The date cannot be in the future. Unknown outcomes use the original Idempotency-Key via /commands; partial/unknown are not zero.

**Outcome:** Accounting account created/corrected with audit; this does not open a bank product.

#### FORM-04 — Income or expense

**Fields:** Type, account, date/time, amount/currency, category/subcategory, merchant, purpose/shares, note/receipt.

**Validation and permissions:** Either member records on any household account; actor comes from session and payer is independent. Amount >0 and asset matches the account. An expense may contain active category/merchant references and typed allocation by current MembershipID; unknown purpose remains unallocated. Income rejects allocation. Confirmed account/amount/date produce posted and one household fact even without classification.

**Outcome:** One transaction, visible allocation and AI status, linked receipt; confirmed outcome and link.

#### FORM-05 — Record transfer or exchange

**Fields:** From/to accounts, dates, both amounts/currencies, fees/fee account, existing movements.

**Validation and permissions:** Either member; distinct household accounts; one outgoing and one incoming side. Principal is excluded from income/expenses; separate fees may use a third asset. Empty existingTransactions creates new movement. A nonempty list supplies all participant IDs/revisions (up to 100); amounts, fees and primary date are checked without creating missing sides. Transfer/exchange/payment links require version checks; ambiguity remains in matching; separate-purchase confirmation releases waiting once.

**Outcome:** Ledger movements linked; no actual transfer or asset purchase.

#### FORM-06 — Correction, matching and undo

**Fields:** Transaction, expectedRevision and reason; complete principal and fees, date, payer, raw merchant/note; category and merchant identity via set|clear, items via replace|clear. Undo uses decisionId and expectedRevisions; compare before/after and source.

**Validation and permissions:** Both members correct household facts. The server retains actor, principal accounts/assets and provenance, validates active categories/merchants and replaces the complete item set atomically. Items and discounts equal principal exactly; top-level category with items is forbidden. Undo preserves later independent fields. Matching and shares remain with their owning tasks.

**Outcome:** New decision and financial revisions with history, or no_change/conflict without effect or lost input. Exclusion does not change bank state; undo recomputes the current effect.

#### FORM-07 — Receipt and item allocation

**Fields:** Photo/PDF, required debit account including cash; items, discounts, categories, personal/shared and percentage or amount shares.

**Validation and permissions:** Either member; file limits follow the contract. Task-2.6 atomically validates items, assets and discounts against one payment, allocates a known receipt-wide discount deterministically and requires clarification for incomplete data. Task-5.3/2.4/2.8 own OCR/PDF, matching and personal/shared shares.

**Outcome:** Created/linked to existing/awaiting clarification/document unsuitable with reason. One confirmed debit.

#### FORM-08 — Purchase refund

**Fields:** Original purchase, returned items/shares/amount, receiving account and actual date.

**Validation and permissions:** Either member; cumulative refund cannot exceed purchase; original historical FX and refunded-part allocation are retained.

**Outcome:** Original purchase month recalculated; cash arrives on actual date; FX separate.

#### FORM-09 — Explicit debt and reimbursement

**Fields:** Creditor and debtor as household members, exact amount/currency, reason and optional expense. Settlement selects an existing transfer and supplies its revision, transferAmount and settledAmount.

**Validation and permissions:** Debt is created explicitly only. Members are distinct and active. Transfer runs between their personal accounts; fees do not settle debt. Principal cannot be reused beyond its remainder. Cross-asset settlement requires both amounts. Changed references surface attention_required or stale instead of silently reducing debt.

**Outcome:** The debt is created, corrected, settled or restored with a new revision and history; the household cash fact is not duplicated.

#### FORM-10 — Plan line and income forecast

**Fields:** Month, obligation/flexible expense/income type, amount/currency, date/repeat, category, personal/shared, owner/shares.

**Validation and permissions:** Owner edits personal, any member shared. Currency funding/allowance preview, expectedRevision; AI changes approved plans only after a decision.

**Outcome:** New plan/forecast revision and partner notification; copying never rolls balances over.

#### FORM-11 — Goal and reserve

**Fields:** Name, personal/shared, personal owner if applicable, amount/currency/deadline; explicit reserve or dedicated account; reserve delta.

**Validation and permissions:** Owner edits personal, any member joint. Preview free money/daily allowance in native currency; no double reserve; joint goals have no personal shares.

**Outcome:** Goal/reserve updated once, partner notification and audit.

#### FORM-12 — AI response and proposal

**Fields:** Answer to a specific clarification or explicit decision on proposed changes; object/question revision.

**Validation and permissions:** Either member for transactions, owner only for personal plan/goal; text cannot change actor. Concurrent response checks revision; preview before financial change.

**Outcome:** Command confirmed, rejected, stale or pending; response/reason and author retained.

#### FORM-13 — Connection and reauth

**Fields:** Platform, external-account owner, history start, available products; secret only in isolated authorization flow.

**Validation and permissions:** Both manage; external owner alone supplies key/password/MFA. Adapter-confirmed read-only scopes, identity and coverage; no MFA/CAPTCHA bypass.

**Outcome:** Connected/syncing/awaiting owner/error; disconnect preserves history and invalidates generation.

#### FORM-14 — Personal preferences and security

**Fields:** Language, display currency, push; passkey name, revoke own device, regenerate own recovery codes.

**Validation and permissions:** Own security only; sensitive changes require auth-contract reauthentication. Do not remove the last access path without replacement. Push denial never blocks in-app.

**Outcome:** Personal preferences saved; revoked session/subscription stops working and codes never enter chat.

#### FORM-15 — Categories and rules

**Fields:** Category/subcategory name, parent and state; merchant name, state and confirmed aliases; merchant/category rule conditions, priority 1–1000, state and exact member shares; preview without persistence.

**Validation and permissions:** Either member manages the household catalog and rules. Conditions within one rule use AND and lower priority wins. Equal-priority identical outcomes are compatible; different outcomes yield rule_conflict. Shares total exactly 100% across active members; expectedRevision, CSRF and the session actor are mandatory. A rule applies only to new facts and never rewrites history.

**Outcome:** A versioned category, merchant or rule is stored; preview exposes applied revisions or a safe unresolved reason. no_change/conflict creates no effect and historical references remain available.
