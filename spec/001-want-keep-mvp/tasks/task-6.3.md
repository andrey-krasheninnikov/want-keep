<!-- want-keep-task: task-6.3 -->
# task-6.3 — Считать начисления и прогноз накоплений / Calculate accruals and savings forecasts

## RU

Показывать доход накопительных продуктов на проверенных условиях.

**Состояние:** Открыта; ready-for-agent. Описание готово к реализации; начало зависит от незакрытых задач и условий ниже. Метка не подтверждает выполнение, live-допуск или разрешение на production; SDD Ready не является блокером.

**Зависимости:** `task-4.1`, `task-4.2`, `task-4.3`, `task-4.4`, `task-4.6`, `task-6.1`.

**Тип:** `implementation`.

### Изменение и контракты

### Текущее поведение

Точные денежные типы, журнал, исправления, распределения и валютная оценка включены в базу. Полнота внешних данных и приёмка расчётов этой задачи проверяются отдельно.

### Требуемое поведение

Сохранить тип продукта, дату действия ставки, basis/day-count, payout/compounding, срок/lock, пополнения и выводы из подтверждённого контракта. Рассчитывать прогноз с воспроизводимыми предпосылками и отдельно фактические начисления; future income не зачислять в ledger. Ограниченные/неизвестные условия показывать как таковые.

### Ключевые контракты

Условия продукта и dated cash flows, rate/basis/day-count, payout/compounding; отдельные actual и forecast с предпосылками.

### Вне объёма

Не исполнять банковские платежи, обмены или сделки. Не подменять неизвестные данные нулём и не переносить финансовые расчёты в UI; платформенное покрытие расширяется отдельным решением.

### Блокирующие зависимости

- [#29](https://github.com/andrey-krasheninnikov/want-keep/issues/29) (`task-4.1`): выполнена.
- [#30](https://github.com/andrey-krasheninnikov/want-keep/issues/30) (`task-4.2`): выполнена.
- [#31](https://github.com/andrey-krasheninnikov/want-keep/issues/31) (`task-4.3`): открыта; блокирует выполнение.
- [#32](https://github.com/andrey-krasheninnikov/want-keep/issues/32) (`task-4.4`): открыта; блокирует выполнение.
- [#34](https://github.com/andrey-krasheninnikov/want-keep/issues/34) (`task-4.6`): открыта; блокирует выполнение.
- [#40](https://github.com/andrey-krasheninnikov/want-keep/issues/40) (`task-6.1`): выполнена.

### Границы изменений

- `backend/internal/savings/`

Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-028:** Личная или совместная цель содержит сумму, валюту, срок и способ накопления: явный резерв либо выделенный счёт.
- **REQ-031:** Кредитные карты показывают задолженность, собственные средства, лимит, минимальный платёж и дату по данным источника.
- **REQ-032:** Грейс-период опирается на условия конкретной карты и показывает сумму и срок сохранения льготы.
- **REQ-033:** Накопления показывают фактические начисления и прогноз по ставкам, срокам, капитализации и денежным потокам.
- **REQ-037:** Исторические расходы используют зафиксированную оценку на дату операции, текущий капитал — актуальную оценку.
- **REQ-039:** Отсутствующие курсы и неподдерживаемые активы не превращаются в нулевые суммы или условный паритет USD/USDT/USDC.
- **REQ-059:** Денежные расчёты используют точную арифметику и явные правила округления на границах.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-028

- **Дано:** Созданы цель покупки и цель накопления в USD.
- **Когда:** Одна цель получает резерв, другая связывается с выделенным счётом.
- **Тогда:** Показаны прогресс, остаток до цели и срок; перемещение на собственный накопительный счёт не становится потребительским расходом.
- **Уровень:** `integration`.

#### AC-033

- **Дано:** Есть вклад или Earn с подтверждёнными условиями, пополнением и выводом.
- **Когда:** Рассчитывается доход за период и прогноз.
- **Тогда:** Факт отделён от прогноза и переоценки; смена ставки и капитализация учитываются по условиям; неизвестные условия блокируют точный прогноз.
- **Уровень:** `integration`.

#### AC-037

- **Дано:** Расход USD 10 оценён в RUB 900; текущий курс стал RUB 100/USD.
- **Когда:** Обновляются котировки и дашборд.
- **Тогда:** Исторический расход остаётся RUB 900, USD-остаток переоценивается; видны источник, время курса и отдельное курсовое изменение.
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

#### AC-070

- **Дано:** Банк передаёт баланс, но не условия грейса; ставка Earn имеет неизвестную базу начисления.
- **Когда:** Открываются прогнозы.
- **Тогда:** Баланс отображается; льгота и точный прогноз имеют причину недоступности; AI не извлекает гарантированную бизнес-логику из рекламной формулировки.
- **Уровень:** `contract+end-to-end`.

### Проверка результата

```sh
make test-go PKG=./internal/savings/... && make test-contract PROVIDER=savings
```

Смена ставки, капитализация, досрочный вывод, неполные условия и границы периода имеют ожидаемый факт/прогноз.

Команды Make являются интерфейсами проверки; наличие target не доказывает реализацию соответствующей suite. Если требуемой suite ещё нет, реализовать её в рамках задачи; отсутствие suite не считается успешной проверкой. Фиксировать фактический результат каждого прогона. make docs-check проверяет документацию и исследовательские инструменты, но не финансовый runtime, браузеры или production. Live/manual-проверки, доступ и расходы фиксируются отдельно.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Show savings-product income under verified terms.

**Status:** Open; ready-for-agent. The description is ready for implementation; open prerequisites and the entry conditions below govern execution. The label does not establish completion, live admission or production authorization; SDD Ready is not a blocker.

**Dependencies:** `task-4.1`, `task-4.2`, `task-4.3`, `task-4.4`, `task-4.6`, `task-6.1`.

**Kind:** `implementation`.

### Change and contracts

### Current behavior

Exact money types, the ledger, corrections, allocations and currency valuation are included in the base. External-data coverage and acceptance of this task's calculations require separate verification.

### Desired behavior

Retain product type, effective rate date, basis/day-count, payout/compounding, term/lock, top-ups and withdrawals from verified contracts. Calculate forecasts with reproducible assumptions separately from actual accruals; never post future income to the ledger. Expose constrained/unknown terms explicitly.

### Key interfaces

Product terms and dated cash flows, rate/basis/day-count, payout/compounding; separate actual and forecast values with assumptions.

### Out of scope

Do not execute bank payments, exchanges or trades. Do not replace unknown data with zero or move financial calculations into the UI; platform coverage requires a separate decision.

### Blocked by

- [#29](https://github.com/andrey-krasheninnikov/want-keep/issues/29) (`task-4.1`): completed.
- [#30](https://github.com/andrey-krasheninnikov/want-keep/issues/30) (`task-4.2`): completed.
- [#31](https://github.com/andrey-krasheninnikov/want-keep/issues/31) (`task-4.3`): open; blocks execution.
- [#32](https://github.com/andrey-krasheninnikov/want-keep/issues/32) (`task-4.4`): open; blocks execution.
- [#34](https://github.com/andrey-krasheninnikov/want-keep/issues/34) (`task-4.6`): open; blocks execution.
- [#40](https://github.com/andrey-krasheninnikov/want-keep/issues/40) (`task-6.1`): completed.

### Change boundaries

- `backend/internal/savings/`

Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-028:** A personal or joint goal has an amount, currency, deadline and funding mode: an explicit reservation or dedicated account.
- **REQ-031:** Credit cards show debt, own funds, credit limit, minimum payment and due date from source data.
- **REQ-032:** Grace-period tracking uses the specific card's terms and shows the amount and deadline needed to preserve the benefit.
- **REQ-033:** Savings show actual accruals and forecasts using rates, terms, compounding and cash flows.
- **REQ-037:** Historical expenses use a fixed transaction-date valuation; current wealth uses a current valuation.
- **REQ-039:** Missing rates and unsupported assets never become zero amounts or assumed USD/USDT/USDC parity.
- **REQ-059:** Money calculations use exact arithmetic and explicit boundary rounding rules.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-028

- **Given:** A purchase goal and a USD savings goal exist.
- **When:** One goal receives a reservation and the other is linked to a dedicated account.
- **Then:** Progress, remaining target and deadline are shown; moving money to an owned savings account is not a consumer expense.
- **Level:** `integration`.

#### AC-033

- **Given:** A deposit or Earn product has confirmed terms, a top-up and a withdrawal.
- **When:** Period income and forecast are calculated.
- **Then:** Actual income is separate from forecast and revaluation; rate changes and compounding follow the terms; unknown terms prevent an exact forecast.
- **Level:** `integration`.

#### AC-037

- **Given:** A USD 10 expense was valued at RUB 900; the current rate becomes RUB 100/USD.
- **When:** Quotes and dashboard refresh.
- **Then:** Historical expense remains RUB 900 and the USD balance is revalued; rate source/time and separate FX change are visible.
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

#### AC-070

- **Given:** A bank exposes balance but no grace terms; an Earn rate has an unknown accrual basis.
- **When:** Forecasts are opened.
- **Then:** Balance is shown; grace eligibility and exact forecasts explain unavailability; AI does not turn marketing wording into guaranteed business rules.
- **Level:** `contract+end-to-end`.

### Verification

```sh
make test-go PKG=./internal/savings/... && make test-contract PROVIDER=savings
```

Rate changes, compounding, early withdrawal, incomplete terms and period boundaries yield expected actuals/forecasts.

Make commands are verification interfaces; the existence of a target does not establish an implemented suite. If a required suite is missing, implement it as part of the task; a missing suite is not a passing check. Record the actual outcome of each run. make docs-check validates documentation and research tooling, not financial runtime, browsers or production. Record live/manual checks, access and costs separately.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
