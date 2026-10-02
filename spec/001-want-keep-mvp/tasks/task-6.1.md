<!-- want-keep-task: task-6.1 -->
# task-6.1 — Реализовать курсы и валютную оценку / Implement rates and currency valuation

## RU

Сохранять историческую оценку и показывать текущие эквиваленты/котировки.

**Состояние:** Реализованы backend/API курсов, семейных валютных итогов и версионной исторической оценки; UI, обратное восстановление старых оценок и production остаются отдельной работой.

**Зависимости:** `task-0.7`, `task-2.2`, `task-3.2`.

**Тип:** `implementation`.

### Изменение и контракты

Использовать CBR как основной USD/RUB, Frankfurter `providers=cbr` (код источника CBR) как fallback/cross-check и CoinGecko Demo для текущих и исторических ≤365 дней BTC/ETH/USDT/USDC в USD. Для более старой криптоистории возвращать `valuation_unavailable`, сохраняя native facts. Текущая цена не переписывает историческую оценку. Platform executable quote хранится только с направлением, amount, временем и известными fee/spread; иначе `quote_unavailable`. Reference rate никогда не подменяет platform quote. Missing/stale/unsupported не равны нулю или peg. Free-key quota, attribution и live probe являются task-6.1 configuration/runtime gate.

### Границы изменений

- `backend/internal/valuation/`
- `backend/internal/integrations/rates/`
- `backend/internal/delivery/valuation/`
- `backend/internal/storage/rates.go`
- `backend/migrations/020_rates_valuation.sql`
- `api/paths/reports.yaml`

Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-002:** Учёт поддерживает RUB, USD, USDT, USDC, BTC и ETH; наличные, банковские деньги и платформенные кошельки различаются счетами.
- **REQ-003:** Общую валюту отображения можно переключать между RUB, USD, USDT, USDC, BTC и ETH.
- **REQ-007:** Обмен и P2P-конвертация собственных денег сохраняют обе валютные суммы, фактический курс и комиссии.
- **REQ-010:** Возврат уменьшает расходы исходного месяца покупки, сохраняя дату реального поступления денег.
- **REQ-016:** Позиции чека распределяют одну оплаченную сумму по категориям без дублирования итога.
- **REQ-037:** Исторические расходы используют зафиксированную оценку на дату операции, текущий капитал — актуальную оценку.
- **REQ-038:** Курсы обмена учитывают направление, сервис, время, сумму применимости и известные комиссии.
- **REQ-039:** Отсутствующие курсы и неподдерживаемые активы не превращаются в нулевые суммы или условный паритет USD/USDT/USDC.
- **REQ-059:** Денежные расчёты используют точную арифметику и явные правила округления на границах.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-003

- **Дано:** Для всех необходимых пар есть актуальная оценка.
- **Когда:** Участник переключает RUB на USD, USDT, USDC, BTC и ETH.
- **Тогда:** Меняется эквивалент итогов, исходные суммы операций и счетов сохраняются.
- **Уровень:** `end-to-end`.

#### AC-007

- **Дано:** Собственные RUB 9 000 обменены на USDT 100, комиссия RUB 50.
- **Когда:** Приходят банковская и криптовалютная стороны подтверждённого обмена.
- **Тогда:** Основные суммы не становятся расходом/доходом; курс RUB 90/USDT отделён от комиссии; неоднозначная связь требует уточнения.
- **Уровень:** `integration`.

#### AC-010

- **Дано:** В августе оплачен расход RUB 1 000.
- **Когда:** В сентябре получен связанный частичный возврат RUB 400.
- **Тогда:** Расход августа становится RUB 600; движение RUB +400 остаётся в сентябре; пересчёт и связь доступны в истории.
- **Уровень:** `integration`.

#### AC-037

- **Дано:** Расход USD 10 оценён в RUB 900; текущий курс стал RUB 100/USD.
- **Когда:** Обновляются котировки и дашборд.
- **Тогда:** Исторический расход остаётся RUB 900, USD-остаток переоценивается; видны источник, время курса и отдельное курсовое изменение.
- **Уровень:** `integration`.

#### AC-038

- **Дано:** Два сервиса дают разные bid/ask и один не раскрывает комиссию.
- **Когда:** Владелец сравнивает RUB → USDT.
- **Тогда:** Показаны сопоставимые направления и свежесть; неизвестная комиссия не считается нулевой; недоступная котировка не заменяется обещанием рыночного курса.
- **Уровень:** `integration`.

#### AC-039

- **Дано:** В источнике есть неподдерживаемый USDC.E; для USDT/USD и USDC/USD отсутствуют курсы.
- **Когда:** Строится общая оценка.
- **Тогда:** Исходные данные сохранены, покрытие оценки обозначено неполным; нет скрытого нуля или автоматического курса 1:1. USDC.E не объединён с USDC по похожему символу.
- **Уровень:** `integration`.

#### AC-059

- **Дано:** Есть дробные BTC, USDT, процентное начисление и распределение чека.
- **Когда:** Данные проходят API, базу и повторный расчёт.
- **Тогда:** Исходная точность не теряется; JSON-суммы не проходят binary float; распределения сходятся точно, округление отображения не меняет журнал.
- **Уровень:** `unit+contract`.

#### AC-065

- **Дано:** Покупка USD 10 распределена по двум категориям и оценена в RUB 900.
- **Когда:** Позже возвращено USD 4 за известную позицию; текущий курс иной.
- **Тогда:** Историческая категория уменьшается на исходную стоимость возвращённой части RUB 360; реальные поступления и валютная разница сохраняются отдельно; превышение суммы возвратов блокируется.
- **Уровень:** `unit+integration`.

#### AC-074

- **Дано:** RUB-сумма известна, отсутствует исторический BTC-кросс.
- **Когда:** Владелец выбирает отчёт в BTC.
- **Тогда:** Нативный факт сохранён и доступен; эквивалент/общий итог обозначен неполным; текущий курс не подставлен вместо исторического.
- **Уровень:** `unit+end-to-end`.

### Проверка результата

```sh
make test-go PKG=./internal/valuation/... && make test-contract PROVIDER=rates && make test-integration AREA=rates
```

RUB, USD, USDT, USDC, BTC и ETH, cross-rates без условного паритета, partial refund, остатки точнее UI и отсутствие исторической цены проверены без изменения native ledger.

Результат backend/API и границы: evidence/task-6.1-rates-valuation.md. Доказанные части AC-003/037/038/039/059/074; AC-065 остаётся частичным до связывания возврата task-2.7. Старые операции требуют отдельного восстановления оценок. Платформенные котировки не поступают до реализации соответствующих адаптеров; UI и production не проверены.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Preserve historical valuation and show current equivalents/quotes.

**Status:** Rates API, household valuation totals and versioned historical valuation are implemented; UI, backfilling older valuations and production remain separate work.

**Dependencies:** `task-0.7`, `task-2.2`, `task-3.2`.

**Kind:** `implementation`.

### Change and contracts

Use CBR as the primary USD/RUB source, Frankfurter `providers=cbr` (CBR source code) as fallback/cross-check, and CoinGecko Demo for current and ≤365-day historical BTC/ETH/USDT/USDC prices in USD. Older crypto history returns `valuation_unavailable` while preserving native facts. A current price never rewrites historical valuation. Retain a platform executable quote only with direction, amount, time and known fee/spread; otherwise return `quote_unavailable`. A reference rate never substitutes for a platform quote. Missing/stale/unsupported is neither zero nor a peg. Free-key quota, attribution and a live probe are task-6.1 configuration/runtime gates.

### Change boundaries

- `backend/internal/valuation/`
- `backend/internal/integrations/rates/`
- `backend/internal/delivery/valuation/`
- `backend/internal/storage/rates.go`
- `backend/migrations/020_rates_valuation.sql`
- `api/paths/reports.yaml`

Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-002:** Accounting supports RUB, USD, USDT, USDC, BTC and ETH; cash, bank money and platform wallets are separate accounts.
- **REQ-003:** The reporting currency can switch among RUB, USD, USDT, USDC, BTC and ETH.
- **REQ-007:** Exchange and P2P conversion of owned money preserve both currency amounts, the actual rate and fees.
- **REQ-010:** A refund reduces expenses in the purchase month while preserving the actual cash receipt date.
- **REQ-016:** Receipt items allocate one paid amount across categories without duplicating the total.
- **REQ-037:** Historical expenses use a fixed transaction-date valuation; current wealth uses a current valuation.
- **REQ-038:** Exchange quotes include direction, provider, timestamp, applicable amount and known fees.
- **REQ-039:** Missing rates and unsupported assets never become zero amounts or assumed USD/USDT/USDC parity.
- **REQ-059:** Money calculations use exact arithmetic and explicit boundary rounding rules.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-003

- **Given:** A current valuation exists for every required pair.
- **When:** The member switches RUB to USD, USDT, USDC, BTC and ETH.
- **Then:** Equivalent totals change while original account and transaction amounts remain unchanged.
- **Level:** `end-to-end`.

#### AC-007

- **Given:** Owned RUB 9,000 is exchanged for USDT 100 with a RUB 50 fee.
- **When:** The bank and crypto legs of a confirmed exchange arrive.
- **Then:** Principal is not income/expense; the RUB 90/USDT rate is separate from the fee; an ambiguous match requires clarification.
- **Level:** `integration`.

#### AC-010

- **Given:** A RUB 1,000 expense was paid in August.
- **When:** A linked RUB 400 partial refund is received in September.
- **Then:** August expense becomes RUB 600; the RUB +400 cash movement stays in September; the recalculation and link are auditable.
- **Level:** `integration`.

#### AC-037

- **Given:** A USD 10 expense was valued at RUB 900; the current rate becomes RUB 100/USD.
- **When:** Quotes and dashboard refresh.
- **Then:** Historical expense remains RUB 900 and the USD balance is revalued; rate source/time and separate FX change are visible.
- **Level:** `integration`.

#### AC-038

- **Given:** Two providers have different bid/ask quotes and one omits its fee.
- **When:** The owner compares RUB → USDT.
- **Then:** Directions and freshness are comparable; an unknown fee is not treated as zero; an unavailable quote is not replaced by a promise of market execution.
- **Level:** `integration`.

#### AC-039

- **Given:** A source contains unsupported USDC.E; USDT/USD and USDC/USD rates are unavailable.
- **When:** A total valuation is built.
- **Then:** Raw data is retained and valuation coverage is incomplete; no hidden zero or automatic 1:1 rate is used. USDC.E is not merged into USDC by symbol similarity.
- **Level:** `integration`.

#### AC-059

- **Given:** Fractional BTC, USDT, interest accrual and receipt allocation exist.
- **When:** Data traverses API, storage and recalculation.
- **Then:** Original precision survives; JSON money never traverses binary floats; allocations reconcile exactly and display rounding does not alter the ledger.
- **Level:** `unit+contract`.

#### AC-065

- **Given:** A USD 10 purchase is split across two categories and valued at RUB 900.
- **When:** USD 4 is later refunded for a known item at a different current rate.
- **Then:** The historical category decreases by the refunded original value RUB 360; actual cash receipts and FX difference stay separate; excess cumulative refunds are rejected.
- **Level:** `unit+integration`.

#### AC-074

- **Given:** A RUB amount is known but its historical BTC cross-rate is missing.
- **When:** The owner selects BTC reporting.
- **Then:** Native actuals remain available; equivalent/total is marked incomplete; a current rate is not substituted for the historical rate.
- **Level:** `unit+end-to-end`.

### Verification

```sh
make test-go PKG=./internal/valuation/... && make test-contract PROVIDER=rates && make test-integration AREA=rates
```

RUB, USD, USDT, USDC, BTC and ETH, cross-rates without assumed parity, partial refund, residuals beyond UI precision and missing historical prices pass without changing the native ledger.

Backend/API result and limits: evidence/task-6.1-rates-valuation.en.md. Backend portions of AC-003/037/038/039/059/074 are covered; AC-065 remains partial until task-2.7 links refunds. Existing operations require a separate valuation backfill. Platform quotes need their provider adapters; UI and production are not verified.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
