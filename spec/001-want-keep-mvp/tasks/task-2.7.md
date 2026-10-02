<!-- want-keep-task: task-2.7 -->
# task-2.7 — Реализовать возвраты и распределение расходов / Implement refunds and expense allocation

## RU

Корректно пересчитывать исходные расходы без искажения движения денег.

**Состояние:** Реализованы backend/API возвратов, версия связи с покупкой и позициями, исторический аналитический эффект и защита от повторного денежного движения. UI, получение курсов, отчёты и production остаются профильным задачам.

**Зависимости:** `task-2.2`, `task-2.6`, `task-2.8`.

**Тип:** `implementation`.

### Изменение и контракты

Ручной возврат создаёт одну posted refund-операцию: деньги поступают в фактическую дату без дохода, а аналитика уменьшает расход исходного месяца. Версионная связь сохраняет revisions покупки и возврата, точные позиции, исходное распределение, зафиксированную историческую оценку и отдельный review request. Совокупный возврат ограничен невозвращённой суммой покупки и каждой позиции под блокировкой покупки; лимит 1000 учитывает только активные связи. Двухэтапное largest-remainder распределение сохраняет frozen reporting total всех частичных возвратов в пределах 1000 весов. Импортированный возврат связывается без второго денежного эффекта. Correction, exclusion, reversal и undo пакетно пересчитывают и сохраняют связи через тот же ledger Writer; история ledger выбирает связи на момент его revision, а отдельная постраничная история refund-связи показывает последующие изменения. Неоднозначная позиция остаётся clarification без вымышленного распределения.

### Границы изменений

- `backend/internal/ledger/`
- `backend/internal/expenses/`
- `backend/internal/storage/`
- `backend/internal/delivery/ledger/`
- `backend/migrations/021_refunds.sql`
- `backend/test/integration/refunds/`
- `api/`

Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-010:** Возврат уменьшает расходы исходного месяца покупки, сохраняя дату реального поступления денег.
- **REQ-011:** Расход признаётся целиком по фактической оплате, включая годовые подписки.
- **REQ-016:** Позиции чека распределяют одну оплаченную сумму по категориям без дублирования итога.
- **REQ-037:** Исторические расходы используют зафиксированную оценку на дату операции, текущий капитал — актуальную оценку.
- **REQ-059:** Денежные расчёты используют точную арифметику и явные правила округления на границах.
- **REQ-067:** Расходы и позиции чеков имеют личное или совместное назначение; общая доля по умолчанию 50/50 с исключениями статьи или покупки.
- **REQ-072:** Конкурирующие изменения, ответы на уточнения и отмены проверяют версию и текущие права, сохраняя обоих авторов.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-010

- **Дано:** В августе оплачен расход RUB 1 000.
- **Когда:** В сентябре получен связанный частичный возврат RUB 400.
- **Тогда:** Расход августа становится RUB 600; движение RUB +400 остаётся в сентябре; пересчёт и связь доступны в истории.
- **Уровень:** `integration`.

#### AC-011

- **Дано:** Годовая подписка стоит RUB 12 000.
- **Когда:** Она оплачена в августе.
- **Тогда:** Факт августа содержит RUB 12 000; автоматического распределения по 12 месяцам нет.
- **Уровень:** `unit`.

#### AC-016

- **Дано:** Оплачено RUB 900 за позиции RUB 600 и RUB 400 со скидкой RUB 100.
- **Когда:** AI извлекает и категоризирует позиции.
- **Тогда:** Сумма распределений точно RUB 900; скидка сохраняется; расхождение суммы направляется на уточнение, а не исправляется выдуманной позицией.
- **Уровень:** `integration`.

#### AC-037

- **Дано:** Расход USD 10 оценён в RUB 900; текущий курс стал RUB 100/USD.
- **Когда:** Обновляются котировки и дашборд.
- **Тогда:** Исторический расход остаётся RUB 900, USD-остаток переоценивается; видны источник, время курса и отдельное курсовое изменение.
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

#### AC-081

- **Дано:** Чек RUB 1000 содержит общие продукты 600 и личные покупки A 100 и B 300.
- **Когда:** Чек заносит любой участник; AI применяет правила или уточняет неизвестное назначение.
- **Тогда:** Факт семьи 1000, A 400, B 600; доли суммируются точно. Исключение покупки приоритетнее статьи, затем 50/50; неоднозначная трата сохранена без вымышленной принадлежности.
- **Уровень:** `integration`.

#### AC-091

- **Дано:** Январский чек: общие товары 600 (50/50), личные A 100, B 300; в феврале доли статьи стали 60/40.
- **Когда:** В феврале возвращаются общие товары на 200.
- **Тогда:** Январь уменьшается семье на 200, каждому на 100 с исходной исторической оценкой; февральская пропорция не меняет январь, cash date возврата остаётся февральской.
- **Уровень:** `integration`.

### Проверка результата

```sh
make check && make test-integration AREA=refunds && make test-refunds-race
```

Шесть активов, частичные/повторные/конкурентные возвраты, позиции и скидки, точная совокупная историческая оценка без накопления округления, исторические revisions связей, комиссия третьего актива, correction/exclusion/undo/reversal, review requests, replay и связь импортного возврата сохраняют точную сумму, исходный месяц и единственный денежный эффект.

Зависимости task-2.2, task-2.6 и task-2.8 включены в базу. Доказательства и границы: evidence/task-2.7-refunds.md. Обязательны make check, refunds и затронутые integration/race/privacy suites. Это не подтверждает эксплуатационную готовность.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Recalculate original expenses without distorting cash movements.

**Status:** Refund backend/API, versioned purchase/item links, historical analytical effects and duplicate cash-effect protection are implemented. UI, rate acquisition, reports and production remain with their owning tasks.

**Dependencies:** `task-2.2`, `task-2.6`, `task-2.8`.

**Kind:** `implementation`.

### Change and contracts

A manual refund creates one posted refund transaction: cash arrives on its actual date without income, while analytics reduces the original month's expense. The versioned link retains purchase and refund revisions, exact items, the original allocation, a frozen historical valuation and a separate review request. Cumulative refunds are capped by the remaining purchase and item amounts while the purchase is locked; the 1000 limit counts active links only. Two-stage largest-remainder allocation preserves the frozen reporting total across all partial refunds within the 1000-weight limit. An imported refund is linked without a second cash effect. Correction, exclusion, reversal and undo recalculate and persist links in a batch through the same ledger Writer; ledger history selects links at its revision, while separate paginated refund-link history shows later changes. Ambiguous items remain clarification without invented allocation.

### Change boundaries

- `backend/internal/ledger/`
- `backend/internal/expenses/`
- `backend/internal/storage/`
- `backend/internal/delivery/ledger/`
- `backend/migrations/021_refunds.sql`
- `backend/test/integration/refunds/`
- `api/`

Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-010:** A refund reduces expenses in the purchase month while preserving the actual cash receipt date.
- **REQ-011:** An expense is recognized in full on payment, including annual subscriptions.
- **REQ-016:** Receipt items allocate one paid amount across categories without duplicating the total.
- **REQ-037:** Historical expenses use a fixed transaction-date valuation; current wealth uses a current valuation.
- **REQ-059:** Money calculations use exact arithmetic and explicit boundary rounding rules.
- **REQ-067:** Expenses and receipt items have personal or joint attribution; joint shares default to 50/50 with line or purchase overrides.
- **REQ-072:** Competing edits, clarification answers and reversals check revision and current permissions while retaining both authors.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-010

- **Given:** A RUB 1,000 expense was paid in August.
- **When:** A linked RUB 400 partial refund is received in September.
- **Then:** August expense becomes RUB 600; the RUB +400 cash movement stays in September; the recalculation and link are auditable.
- **Level:** `integration`.

#### AC-011

- **Given:** An annual subscription costs RUB 12,000.
- **When:** It is paid in August.
- **Then:** August actual expense includes RUB 12,000; it is not automatically amortized over 12 months.
- **Level:** `unit`.

#### AC-016

- **Given:** RUB 900 was paid for RUB 600 and RUB 400 items with a RUB 100 discount.
- **When:** AI extracts and categorizes the items.
- **Then:** Allocations total exactly RUB 900 and retain the discount; a mismatch is clarified rather than patched with an invented item.
- **Level:** `integration`.

#### AC-037

- **Given:** A USD 10 expense was valued at RUB 900; the current rate becomes RUB 100/USD.
- **When:** Quotes and dashboard refresh.
- **Then:** Historical expense remains RUB 900 and the USD balance is revalued; rate source/time and separate FX change are visible.
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

#### AC-081

- **Given:** A RUB 1,000 receipt contains joint groceries of 600 and personal purchases of A 100 and B 300.
- **When:** Either member enters the receipt; AI applies rules or clarifies unknown attribution.
- **Then:** Household actual is 1,000, A 400, B 600; shares sum exactly. Purchase override takes precedence over plan line, then 50/50; ambiguous spending persists without invented attribution.
- **Level:** `integration`.

#### AC-091

- **Given:** January receipt: joint items 600 (50/50), personal A 100, B 300; February plan shares became 60/40.
- **When:** Joint items worth 200 are returned in February.
- **Then:** January falls by 200 for the household and 100 for each member using original historical valuation; February shares do not rewrite January and refund cash date stays in February.
- **Level:** `integration`.

### Verification

```sh
make check && make test-integration AREA=refunds && make test-refunds-race
```

Six assets, partial/replayed/concurrent refunds, items and discounts, exact cumulative historical valuation without rounding drift, historical link revisions, a third-asset fee, correction/exclusion/undo/reversal, review requests, replay and imported-refund linking preserve exact totals, the original month and one cash effect.

Dependencies task-2.2, task-2.6 and task-2.8 are included in the base. Evidence and boundaries: evidence/task-2.7-refunds.en.md. Require make check, refunds and affected integration/race/privacy suites. This does not confirm operational readiness.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
