# Task-4.1: Alfa read qualification

[Русский](task-4.1-alfa.md)

Date: 2026-10-28. Task: [#29](https://github.com/andrey-krasheninnikov/want-keep/issues/29).

## Scope and status

D-47 supersedes the D-37 product scope: current accounts, debit cards, transactions and cashback. By owner decision, Alfa savings accounts, deposits and credit products are excluded. Shared financial features and other providers remain in scope.

The first live structured-read qualification stage covered one account. The connector, normalization, background worker and `PROVIDER=alfa` / `AREA=alfa` suites are not implemented yet. No financial records or deployment admission were created. This research does not establish acceptance of AC-042 or AC-106.

## Observed routes

Every route below belongs to the authenticated `https://web.alfabank.ru` portal, not the official Alfa API. The owner completed authentication and MFA in separate Chrome with a nonpersistent context. Personal browser profiles were not used.

| ID | Read | Verified shape and limitation |
| --- | --- | --- |
| ALFA-E21 | `GET /api/v1/account/` | `accounts[]`: `number`, `type`, `openDate`, `properties` and separate `amount`, `total`, `holds`, `currentBalance`. Reads with `accountNumber`/`type` and without filters were verified. `currentBalance` is not declared spendable owned funds without verified semantics. |
| ALFA-E22 | `GET /api/v1/cards/masked-cards` | `cards[]`: `id`, `number`, `isCredit`, `isClosed`, `account.number`, lifecycle/status. This supplies a card alias, not a separate card balance. A mask does not replace a stable card ID. |
| ALFA-E23 | `POST /api/v1/operations-history/operations` | The body contains `page`, `size`, `forced`, `filters`. Both an empty filter list and a `type=accounts` filter with string references were observed. The response contains `operations[]` and `pagesInfo.page/size`. Pages 1 and 2, each with 20 records, were read separately. |
| ALFA-E24 | `GET /api/v1/operations-history/operations/{id}` | Details use an ID from the list. Fields include `id`, `operationId`, `dateTime`, `direction`, `amount`, `status`, `reference`, `fields`, `loyaltyDetails`. A retrieved income detail had `status=null`; it must not automatically become `posted`. |
| ALFA-E25 | `GET /api/v1/loyalty/accounts` and `GET /api/v1/loyalty/accounts/{id}` | Loyalty list and summary: balance, forecast, monthly accruals and total cashback. The summary does not establish bank-payout identity or create income on its own. |

Observed RUB amounts use `currency=RUR`, an integer `value` and `minorUnits=100`. Original numeric tokens must be handled exactly: inspecting JSON shapes does not prove future Go/TypeScript or database normalization precision.

Observed history states include `HOLD`, `SUCCESS`, `FAILED`, and missing or `null` status. An absent fee field is not zero. The account response contains no separately verified balance-state timestamp. `dateTime`, HTTP response time and fetch time must not replace unknown financial dates without evidence.

## Transport checks and read limitations

A repeated first-page request returned the same IDs in the same order. First-page IDs were unique, and the retrieved second page did not overlap the first. These are observations of one range in one session, not proof of complete history, stable offset pagination or lifecycle revisions.

The history request was also repeated with a minimal set of previously observed headers: `accept`, `content-type`, `x-xsrf-token`, `x-security-request`, `device-app-id`, `ao-origin-application-id`, `zone-offset`. Values remain secret and are omitted here. That replay did not require forwarding `x-gib-*` antifraud headers; this is not a claim that they are generally optional.

A new BrowserContext with a temporary in-memory cookie copy, page scripts disabled and service workers blocked successfully read history, the account list and cashback summary. Its network filter allowed only verified GET requests and the history-read POST. PUT was not allowed. Sessions and bank responses remained in research-process memory; they were not saved in the repository, application or database.

The full portal was observed sending an automatic `PUT /api/v1/loyalty/accounts/{id}/last-seen-balance`. It is outside the read allowlist. Automatic review rejected a subsequent full cashback-page navigation; the summary was then read directly through previously observed GET requests in the context without page scripts. Full portal navigation is not a verified safe collector workflow.

A separate Playwright APIRequestContext request failed certificate verification; the exact trust-chain cause is not established. TLS verification was not disabled. Browser-read success does not prove Node HTTP transport or deployment-host reachability.

## Open contract questions

- Verify identity across reauthentication and a second independent account. The owner signed in again, but IDs were not compared across those sign-ins.
- Verify the selected-date filter, history end, retention, gaps and pagination behavior during new or late transaction changes.
- Establish balance units/composition, known/unknown fees, financial dates and `status=null` handling without invented facts.
- Establish stable-ID links from accrual to bank payout and cashback correction/reversal. An expected accrual does not increase funds; a confirmed payout becomes one income entry under D-47.
- Implement and verify normalization, vault enrollment, fences, worker, PostgreSQL integration and synthetic contract suites after their relevant questions are closed.
- Obtain provider/host evidence for the exact D-43 binding. Production and admission remain unchanged.

A second independent account will be available later. The issue stays open until mandatory evidence is complete. SDD status is **Ready for development**; connector acceptance and operational readiness are not established.
