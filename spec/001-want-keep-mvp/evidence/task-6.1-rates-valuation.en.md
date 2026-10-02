# Task-6.1 — rates and currency valuation

The backend and `/api/v1/rates`, `/api/v1/reports/valuation` retain native monetary amounts and show reference equivalents separately. CBR USD/RUB is fetched directly; Frankfurter v2 is queried only with `providers=cbr` for fallback and cross-checking. BTC, ETH, USDT and USDC each use a distinct CoinGecko Demo USD price. Similar asset labels and stablecoin parity are never assumed. Cross-rates use exact rational arithmetic; decimal representation is bounded at the response boundary, not in the ledger.

Migration `020_rates_valuation.sql` adds immutable source observations, historical valuation revisions and legs, request quota, and validated platform quotes. A quote requires an exact pair, direction, amount, time, and confirmed fee and spread coverage; a reference rate cannot substitute for it. A historical transaction component snapshot pins its source observations. The report separates revaluation from actual income and spending. Owned, available, and debt equivalents appear by asset and as household totals; unknown inputs make a total unknown, while a known amount can still show a source discrepancy. Missing prices retain native data and an incomplete-data reason. A partial refund can calculate its share of the original valuation from cumulative boundaries; linking a refund to its purchase remains task-2.7.

The `transaction.changed` background job prepares valuations for each new revision. Existing operations without such a job appear as `valuation_pending`; report reads do not create hidden records. Provider adapters do not yet submit executable platform quotes. Product screens and production were not changed.

## Checks

- Local unit tests cover six assets, separate stablecoins, cross-rates, precision, source legs, unavailable prices and partial refunds.
- Contract tests cover CBR XML, the Frankfurter filter, CoinGecko current/history, private-file credentials and invalid provider responses.
- The PostgreSQL suite covers observation revisions and immutability, exact values, quota, isolation and quote amount, pinned snapshots and repeated writes.
- Read-only probes on 2026-10-02: CBR XML and CoinGecko Demo current/history returned HTTP 200; Frankfurter v2 with a lowercase path and `providers=cbr` returned HTTP 200. The initial uppercase URL returned HTTP 403 and was replaced. The key was never placed in this report, command arguments or logs.

Source references: [CBR XML](https://www.cbr.ru/development/sxml/), [Frankfurter CBR](https://frankfurter.dev/providers/cbr/), [CoinGecko history](https://docs.coingecko.com/reference/coins-id-history), [CoinGecko attribution](https://brand.coingecko.com/resources/attribution-guide). These checks do not establish a full browser scenario or a production service. SDD remains **Ready for development**.
