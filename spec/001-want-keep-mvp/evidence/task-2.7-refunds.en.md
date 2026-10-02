# Task-2.7 — refunds and expense allocation

The refund backend/API is implemented on top of task-2.2, task-2.6 and task-2.8. The `expenses` domain calculates a versioned link between a purchase and refund transaction; ledger remains the sole owner of postings and lifecycle, while the purchase allocation snapshot owns historical household allocation. Production and UI were not changed.

A manual refund creates one posted refund transaction and one link through the shared matching ingress. A later matching imported receipt awaits resolution without a second cash effect; after linking, only the carrier reduces purchase expense. Two distinct manual refunds of the same amount are not merged automatically. Principal arrives on the actual date without income, while the analytical effect reduces the original expense month. An existing imported refund transaction can be linked without a second cash movement. Principal asset matches the purchase; fees remain separate postings, including a third asset.

Migration 020 stores the current link, immutable revisions, exact item portions, member/category effects, frozen historical valuation and a separate review request for every material link revision. Compound household foreign keys bind both operations, purchase/refund revisions, receipt items, memberships and categories. The total and per-item remaining amounts are checked in the household transaction under the existing lock. Clarification retains a confirmed receipt without inventing an item or shares.

The shared ledger Writer recalculates links after correction, exclusion, reversal, cancellation and undo. Related refund revisions are loaded in one batch; cumulative caps and historical valuation are calculated in one pass. An excluded refund does not consume the 1000-active-link capacity. A nonfinancial purchase correction retains its historical basis when date and principal match; an incompatible correction fails atomically. An unchanged calculation creates no revision or event. Two-stage largest-remainder allocation preserves the frozen reporting total within the 1000-weight limit. Changed links use set-based SQL persistence; current lists and history pages hydrate links in batches at the relevant ledger revision. Separate paginated refund-link history exposes a later imported-refund link and the actor for each change. A missing basis remains `historical_basis_unavailable`.

## Verification matrix

Required candidate commands are `make check`, `make check-contracts`, `make test-integration AREA=refunds`, `make test-refunds-race`, affected integration/race/privacy suites and `git diff --check`. PostgreSQL 17.11 is required; the refund suite fails when the database is unavailable. Exact published-candidate results are recorded in the PR and Issue.

Coverage includes a RUB 1000 August purchase with a RUB 400 September refund; a USD 10 purchase frozen at RUB 900 with a USD 4 refund producing RUB 360; six partial refunds preserving a frozen RUB 1 total exactly; a historical refund revision retaining its prior amount; six assets; third-asset fees; a discounted receipt with exact item portions; clarification without allocation; total and item caps; concurrent commands; idempotent replay; purchase/refund correction, exclusion, undo and reversal; review requests; and linking an existing refund transaction without a second movement.

## Acceptance boundaries

| Criteria | Proven by task-2.7 | Downstream verification |
| --- | --- | --- |
| AC-010/011/016 | Actual receipt and original month are separate; partial/item caps and absence of a current-rate fallback are verifiable | Historical quote acquisition in task-6.1 and reports |
| AC-037/059 | Exact native item/allocation effects and frozen valuation remain separate by asset | Current revaluation and FX analytics in task-6.1 |
| AC-065/081 | The purchase snapshot and discounts yield exact member/category effects without retroactive new rules | Budget views and browser FORM-08/SCR-010 |
| AC-091 | Both members, command replay and imported linking preserve one financial fact | Live receipt/bank import and end-to-end acceptance |

The SDD remains **Ready for development**. UI, live provider IO, rate acquisition, budget reports and production are not verified by this task.
