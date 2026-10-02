# Integrations and provider gates

[Русский](integrations.md)

Evidence snapshot: 2026-09-07. The SDD is **Ready for development** under D-38. This does not prove running connectors: every provider deployment stays disabled until D-43 admission from task-4.x and task-8.x evidence for the exact binding.

## Mandatory coverage

| Platform | MVP scope | Design evidence | Entry/deployment gate |
| --- | --- | --- | --- |
| Alfa-Bank | Debit card, current and savings accounts, deposits and cashback (D-37) | [UI/API research](evidence/alfa.en.md): product sections and history are available; only synthetic facts are public. An Alfa credit card is not required. | task-4.1: authorized structured read path, allowlist, stable IDs, coverage/revisions/statuses/fees/cashback, reauthentication, two accounts and target-host Alfa route. |
| Raiffeisenbank Russia | Individual entrepreneur current account through RBO API (D-35) | [API/CAMT evidence](evidence/raiffeisen.en.md): Account, CAMT.053, OPBD/CLBD and `no-statements`; synthetic JSON/XML. | task-4.2: OAuth lifecycle, CAMT corrections/reversals, full coverage, two accounts, unknown balances/fees and live conformance. |
| Ozon Bank | Debit card and linked main account (D-32) | [Sanitized HAR projection](evidence/ozon.en.md): read routes, pagination, fee relation and synthetic fixtures. | task-4.3: session-transport permission, stable account identity, end of history, lifecycle/revisions, second account and reauthentication. |
| Bybit | Funding USDT/USDC/ETH/BTC, used Flexible Easy Earn and P2P (D-36) | [Research](evidence/bybit.en.md) and [read-only RSA API](evidence/bybit-api.en.md); official APIs take priority. | task-4.4: route-specific identities, hourly collision, precision/history gaps, two accounts, rotation/revocation and live conformance. |
| Aifory Pro | RUB accounts, USDT, ETH and existing USD card (D-33) | [UI research](evidence/aifory.en.md): selected areas are available; Flutter UI is not an API schema. | task-4.5: authorized structured fixture, allowlist, account/log IDs, card lifecycle/fees/FX, coverage, reauthentication and two accounts. |
| EMCD | USDT wallet, Grow/Coinhold, existing crypto cards and P2P history (D-34) | [UI research](evidence/emcd.en.md) and synthetic scenarios; the mining API does not cover selected products. | task-4.6: structured fixtures for every log, allowlist, balance composition, lifecycle/fees, coverage, reauthentication and two accounts. |

Unused products excluded by D-32–D-37 require a separate expansion decision and do not block current scope. Shared manual credit-card, savings and trading/mining domain features remain where they are not tied to an excluded provider product.

## Shared read contract

The read allowlist is route/action specific. Payment, transfer, trade, product-open, P2P create/pay/release, stake/redeem and every other external mutation are forbidden. APIs take priority over a browser collector; a collector is allowed only for a proven API gap and owner permission. The external owner supplies password/MFA; secrets never enter Git, AI, logs or fixtures.

D-43 binds admission to environment, adapter/collector build, contract, allowlist, non-secret configuration and operator permission. The `backend/internal/connections/admission/` application boundary and task-1.3 storage adapter own the aggregate/repository and atomic combine/invalidate transition. task-4.x proves provider evidence and task-8.x proves host/deployment evidence; only the admission service combines both passes. Jobs/results carry the exact binding and monotonic `admissionRevision`: a stale/missing binding yields `provider_not_admitted` before a new job or collector IO, while commit-time revalidation sends an already started stale result to quarantine without a source record/posting. Cancellation of a started read is best effort. Pre-admission conformance also runs in quarantine.

D-39 defines identity as `household + provider + stable external account + product/log namespace + provider record ID`. Amount, time, text, connection/session and UI labels are not identity. A provider-specific immutable fallback must be documented; a collision yields `source_ambiguous`, retains evidence and posts no money.

Every adapter retains raw revision/hash reference, fetched/occurred time, status, native amounts, fee known/unknown, pagination cursor, requested/observed coverage and reauthentication state. A gap yields `source_partial`; unknown balance/fee is not zero.

## Provider-specific decisions

- **Bybit:** IDs belong to a specific route namespace. Cross-log amount/time is a candidate only. Hourly fallback `(coin, productId, hourlyDate)` is allowed in the hourly namespace; differing payload is a collision. A short page with a cursor does not end an import.
- **Raiffeisen:** Account UUID and number/accountKeys differ. A CAMT entry permits 1:N details. A sufficient versioned `camtCrossReportFingerprint` is always the canonical providerRecordId across camt.052/camt.053; optional NtryRef/AcctSvcrRef/EndToEndId and statement/report ID are atomically registered aliases/provenance. Amount/time are excluded. An insufficient fingerprint or ambiguous alias mapping yields `source_ambiguous`, retains evidence and creates no new posting. Corrections/reversals create revisions. When camt.052 is unavailable, expose the last confirmed CLBD with `asOf`; available/locked/fee remain unknown.
- **Ozon:** accountToken/connection and groupID are not identity; a parent relation links a fee without merging financial effects.
- **Alfa/Aifory/EMCD:** signed-in UI observations confirm scope, but task-4.x must obtain stable source fields from a structured fixture. Until then, neither UI text nor a DOM selector creates a posting.

## Rates

D-40: CBR is primary USD/RUB; Frankfurter `providers=cbr` (CBR source code) is fallback/cross-check; CoinGecko Demo supplies current and up-to-365-day BTC/ETH/USDT/USDC prices in USD. Older history yields `valuation_unavailable` while retaining native amount. A platform quote without direction/amount/time/known fee-spread yields `quote_unavailable`; a reference rate never substitutes for it. task-6.1 verifies Demo key/quota/attribution before runtime.
