<!-- want-keep-task: task-6.4 -->
# task-6.4 — Сравнивать доходность денежных потоков / Compare dated cash-flow returns

## RU

Сопоставлять вложения без ложной доходности от пополнения или FX.

**Состояние:** Открыта; ready-for-agent. Описание готово к реализации; начало зависит от незакрытых задач и условий ниже. Метка не подтверждает выполнение, live-допуск или разрешение на production; SDD Ready не является блокером.

**Зависимости:** `task-6.3`.

**Тип:** `implementation`.

### Изменение и контракты

### Текущее поведение

Точные денежные типы, журнал, исправления, распределения и валютная оценка включены в базу. Полнота внешних данных и приёмка расчётов этой задачи проверяются отдельно.

### Требуемое поведение

Реализовать XIRR Actual/365 по объединённым потокам одной календарной даты. Cash-flow decimal остаются точными; fractional power считать как `exp((days/365) × ln(1+r))` в decimal context минимум 50 значащих цифр, ROUND_HALF_EVEN, с доказанной общей погрешностью NPV `≤ 1e-24 × max(1, Σ|CF|)`. Требуются положительный и отрицательный потоки и ровно одна смена знака. Решать NPV=0 bisection от `rLow=-1+1e-12` до `rHigh=1 000 000`; bracket требует разные знаки boundary NPV либо границу в tolerance. Остановка: `|NPV| ≤ 1e-12 × max(1, Σ|CF|)` или ширина `≤ 1e-12 × max(1, |rMid|)`, максимум 512 итераций; результат — rMid, HALF_EVEN до 12 знаков после запятой. Если error bound/знак не доказан, нет bracket, несколько смен знака, нулевой период, missing valuation или convergence — объяснённый `unavailable`, не 0%. Проверить irregular 182-day и boundary vectors; показывать native/reporting currency, доход и метод отдельно от FX.

### Ключевые контракты

Dated cash flows, valuation и XIRR Actual/365 с точным decimal-контекстом, заданными границами и объяснённым unavailable.

### Вне объёма

Не исполнять банковские платежи, обмены или сделки. Не подменять неизвестные данные нулём и не переносить финансовые расчёты в UI; платформенное покрытие расширяется отдельным решением.

### Блокирующие зависимости

- [#42](https://github.com/andrey-krasheninnikov/want-keep/issues/42) (`task-6.3`): открыта; блокирует выполнение.

### Границы изменений

- `backend/internal/returns/`

Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-033:** Накопления показывают фактические начисления и прогноз по ставкам, срокам, капитализации и денежным потокам.
- **REQ-034:** Доходность вложений сравнивается с учётом дат денежных потоков и валюты оценки.
- **REQ-037:** Исторические расходы используют зафиксированную оценку на дату операции, текущий капитал — актуальную оценку.
- **REQ-059:** Денежные расчёты используют точную арифметику и явные правила округления на границах.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-033

- **Дано:** Есть вклад или Earn с подтверждёнными условиями, пополнением и выводом.
- **Когда:** Рассчитывается доход за период и прогноз.
- **Тогда:** Факт отделён от прогноза и переоценки; смена ставки и капитализация учитываются по условиям; неизвестные условия блокируют точный прогноз.
- **Уровень:** `integration`.

#### AC-034

- **Дано:** Два вложения имеют разные даты пополнений и одинаковый конечный остаток.
- **Когда:** Строится сравнение доходности.
- **Тогда:** Показаны фактический доход и годовая денежно-взвешенная доходность с датами/методом; некорректные или неоднозначные расчёты обозначены недоступными, не нулём.
- **Уровень:** `unit`.

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

### Проверка результата

```sh
make test-go PKG=./internal/returns/...
```

`-1000/+1100` за 365 дней даёт 10%; `-1000/+1050` за 182 дня даёт `0.102795595422`. rLow/root-above-rHigh, отсутствие или несколько смен знака, same-day net zero, missing valuation, недоказанный numeric error и no convergence дают заданный boundary outcome либо `unavailable`.

Команды Make являются интерфейсами проверки; наличие target не доказывает реализацию соответствующей suite. Если требуемой suite ещё нет, реализовать её в рамках задачи; отсутствие suite не считается успешной проверкой. Фиксировать фактический результат каждого прогона. make docs-check проверяет документацию и исследовательские инструменты, но не финансовый runtime, браузеры или production. Live/manual-проверки, доступ и расходы фиксируются отдельно.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Compare investments without treating contributions or FX as yield.

**Status:** Open; ready-for-agent. The description is ready for implementation; open prerequisites and the entry conditions below govern execution. The label does not establish completion, live admission or production authorization; SDD Ready is not a blocker.

**Dependencies:** `task-6.3`.

**Kind:** `implementation`.

### Change and contracts

### Current behavior

Exact money types, the ledger, corrections, allocations and currency valuation are included in the base. External-data coverage and acceptance of this task's calculations require separate verification.

### Desired behavior

Implement Actual/365 XIRR after aggregating flows on the same calendar date. Cash-flow decimals remain exact; evaluate fractional powers as `exp((days/365) × ln(1+r))` in a decimal context of at least 50 significant digits, ROUND_HALF_EVEN, with a proven total NPV error `≤ 1e-24 × max(1, Σ|CF|)`. Inputs require positive and negative flows and exactly one sign transition. Solve NPV=0 by bisection from `rLow=-1+1e-12` through `rHigh=1,000,000`; a bracket requires opposite boundary NPV signs or a boundary within tolerance. Stop at `|NPV| ≤ 1e-12 × max(1, Σ|CF|)` or interval width `≤ 1e-12 × max(1, |rMid|)`, with at most 512 iterations; return rMid rounded HALF_EVEN to 12 decimal places. If the error bound/sign is not established, no bracket exists, signs change more than once, the period is zero, valuation is missing or convergence fails, return explained `unavailable`, never 0%. Test irregular 182-day and boundary vectors; show native/reporting currency, income and method separately from FX.

### Key interfaces

Dated cash flows, valuation and XIRR Actual/365 with an exact decimal context, specified boundaries and explained unavailable results.

### Out of scope

Do not execute bank payments, exchanges or trades. Do not replace unknown data with zero or move financial calculations into the UI; platform coverage requires a separate decision.

### Blocked by

- [#42](https://github.com/andrey-krasheninnikov/want-keep/issues/42) (`task-6.3`): open; blocks execution.

### Change boundaries

- `backend/internal/returns/`

Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-033:** Savings show actual accruals and forecasts using rates, terms, compounding and cash flows.
- **REQ-034:** Investment returns are compared using dated cash flows and valuation currency.
- **REQ-037:** Historical expenses use a fixed transaction-date valuation; current wealth uses a current valuation.
- **REQ-059:** Money calculations use exact arithmetic and explicit boundary rounding rules.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-033

- **Given:** A deposit or Earn product has confirmed terms, a top-up and a withdrawal.
- **When:** Period income and forecast are calculated.
- **Then:** Actual income is separate from forecast and revaluation; rate changes and compounding follow the terms; unknown terms prevent an exact forecast.
- **Level:** `integration`.

#### AC-034

- **Given:** Two investments have different top-up dates and the same ending balance.
- **When:** A return comparison is built.
- **Then:** Actual income and annualized money-weighted return show dates/method; invalid or ambiguous calculations are unavailable rather than zero.
- **Level:** `unit`.

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

### Verification

```sh
make test-go PKG=./internal/returns/...
```

`-1000/+1100` over 365 days returns 10%; `-1000/+1050` over 182 days returns `0.102795595422`. rLow/root-above-rHigh, no or multiple sign transitions, same-day net zero, missing valuation, unproven numeric error and non-convergence yield the specified boundary outcome or `unavailable`.

Make commands are verification interfaces; the existence of a target does not establish an implemented suite. If a required suite is missing, implement it as part of the task; a missing suite is not a passing check. Record the actual outcome of each run. make docs-check validates documentation and research tooling, not financial runtime, browsers or production. Record live/manual checks, access and costs separately.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
