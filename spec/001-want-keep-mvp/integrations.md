# Интеграции и provider gates

[English](integrations.en.md)

Срез evidence: 2026-09-07. SDD **Ready for development** по D-38. Это не доказательство работающих коннекторов: каждый provider deployment выключен до D-43 admission по результатам task-4.x и task-8.x для точного binding.

## Обязательное покрытие

| Платформа | Scope MVP | Доказательство для дизайна | Entry/deployment gate |
| --- | --- | --- | --- |
| Альфа-Банк | Дебетовая карта, текущий и накопительные счета, вклады, кэшбэк (D-37) | [UI/API research](evidence/alfa.md): продуктовые разделы и история доступны; опубликованы только синтетические сведения. Кредитка Alfa не требуется. | task-4.1: разрешённый structured read path, allowlist, stable IDs, coverage/revisions/statuses/fees/cashback, reauth, два аккаунта и target-host Alfa route. |
| Райффайзенбанк РФ | Расчётный счёт ИП через RBO API (D-35) | [API/CAMT evidence](evidence/raiffeisen.md): Account, CAMT.053, OPBD/CLBD и `no-statements`; синтетические JSON/XML. | task-4.2: OAuth lifecycle, CAMT corrections/reversals, full coverage, два аккаунта, unknown balances/fees и live conformance. |
| Ozon Банк | Дебетовая карта и связанный основной счёт (D-32) | [Sanitized HAR projection](evidence/ozon.md): read routes, pagination, fee relation и synthetic fixtures. | task-4.3: session-transport permission, stable account identity, end-of-history, lifecycle/revisions, второй аккаунт и reauth. |
| Bybit | Funding USDT/USDC/ETH/BTC, используемый Flexible Easy Earn и P2P (D-36) | [Research](evidence/bybit.md) и [read-only RSA API](evidence/bybit-api.md); официальный API приоритетен. | task-4.4: route-specific identities, hourly collision, precision/history gaps, два аккаунта, rotation/revocation и live conformance. |
| Aifory Pro | RUB-счета, USDT, ETH и используемая карта USD (D-33) | [UI research](evidence/aifory.md): выбранные области доступны; Flutter UI не является схемой API. | task-4.5: разрешённый structured fixture, allowlist, account/log IDs, card lifecycle/fees/FX, coverage, reauth и два аккаунта. |
| EMCD | Кошелёк USDT, Grow/Coinhold, используемые криптокарты и P2P history (D-34) | [UI research](evidence/emcd.md) и синтетические сценарии; mining API не заменяет выбранные продукты. | task-4.6: structured fixtures каждого журнала, allowlist, balance composition, lifecycle/fees, coverage, reauth и два аккаунта. |

Неиспользуемые продукты, исключённые D-32–D-37, расширяются отдельным решением и не блокируют текущий scope. Общие ручные кредитки, накопления, trading/mining domain features сохраняются там, где они не привязаны к исключённому provider product.

## Общий read-контракт

Read allowlist задаётся на route/action уровне. Любые payment, transfer, trade, product-open, P2P create/pay/release, stake/redeem и другие внешние изменения запрещены. API предпочтительнее browser collector; collector допускается только при доказанном API-пробеле и разрешении владельца. Пароль/MFA вводит внешний владелец, секреты не попадают в Git, AI, логи или fixtures.

D-43 связывает admission с environment, adapter/collector build, contract, allowlist, non-secret config и operator permission. Application boundary `backend/internal/connections/admission/` и storage adapter task-1.3 владеют aggregate/repository и атомарным combine/invalidate. task-4.x подтверждает provider evidence, task-8.x — host/deployment evidence; только admission service объединяет оба pass. Job/result несут exact binding и monotonic `admissionRevision`: stale/missing binding даёт `provider_not_admitted` до нового job/collector IO, а commit-time revalidation переводит уже начатый stale result в quarantine без source record/проводки. Cancel начатого read — best effort. Pre-admission conformance также идёт в quarantine.

D-39 задаёт identity: `household + provider + stable external account + product/log namespace + provider record ID`. Сумма, время, текст, connection/session и UI-label не являются identity. Provider-specific immutable fallback должен быть документирован; collision даёт `source_ambiguous`, сохраняет evidence и не проводит деньги.

Каждый адаптер сохраняет raw revision/hash reference, fetched/occurred time, status, native amounts, fee known/unknown, pagination cursor, requested/observed coverage и reauth state. Gap даёт `source_partial`; unknown balance/fee не равен нулю.

## Provider-specific решения

- **Bybit:** IDs относятся к конкретному route namespace. Cross-log amount/time — только кандидат. Hourly fallback `(coin, productId, hourlyDate)` разрешён в hourly namespace; differing payload — collision. Короткая страница с cursor не завершает импорт.
- **Raiffeisen:** Account UUID и number/accountKeys различаются. CAMT entry допускает 1:N details. Достаточный versioned `camtCrossReportFingerprint` всегда является canonical providerRecordId между camt.052/camt.053; optional NtryRef/AcctSvcrRef/EndToEndId и statement/report ID регистрируются атомарно как aliases/provenance. Amount/time исключены. Недостаточный fingerprint или неоднозначное alias mapping даёт `source_ambiguous`, сохраняет evidence и не создаёт новую проводку. Corrections/reversals создают revisions. Если camt.052 недоступен, показывается последний подтверждённый CLBD с `asOf`; available/locked/fee остаются unknown.
- **Ozon:** accountToken/connection и groupID не являются identity; parent relation связывает комиссию, не объединяя финансовые эффекты.
- **Alfa/Aifory/EMCD:** авторизованные UI-наблюдения подтверждают scope, но stable source fields получает только structured fixture в task-4.x. До этого ни UI-текст, ни DOM-selector не создаёт проводку.

## Курсы

D-40: CBR — основной USD/RUB; Frankfurter `providers=cbr` (код источника CBR) — fallback/cross-check; CoinGecko Demo — current и история BTC/ETH/USDT/USDC в USD до 365 дней. Старше 365 дней — `valuation_unavailable` с сохранением native amount. Platform quote без direction/amount/time/known fee-spread — `quote_unavailable`; reference rate его не заменяет. Demo key/quota/attribution проверяет task-6.1 перед runtime.
