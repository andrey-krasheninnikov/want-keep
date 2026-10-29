<!-- want-keep-task: task-6.2 -->
# task-6.2 — Учитывать кредитки и грейс-период / Account for credit cards and grace periods

## RU

Показывать долг и условия сохранения льготы по подтверждённым данным.

**Состояние:** Открыта; ready-for-agent. Описание готово к реализации; начало зависит от незакрытых задач и условий ниже. Метка не подтверждает выполнение, live-допуск или разрешение на production; SDD Ready не является блокером.

**Зависимости:** `task-4.1`, `task-4.2`, `task-4.3`, `task-2.2`.

**Тип:** `implementation`.

### Изменение и контракты

### Текущее поведение

Точные денежные типы, журнал, исправления, распределения и валютная оценка включены в базу. Полнота внешних данных и приёмка расчётов этой задачи проверяются отдельно.

### Требуемое поведение

Маппить statement cycle, задолженность, own funds, minimum/due date и grace eligibility из проверенных контрактов конкретных карт. Доменные правила должны учитывать исключения, частичные платежи и изменение условий. Не строить сроки из рекламных текстов или универсального graceDays. Missing terms дают unknown; purchase/repayment/interest/fee имеют различную семантику.

### Ключевые контракты

Statement cycle, debt/own funds, minimum payment/due date и grace eligibility с происхождением и подтверждёнными условиями.

### Вне объёма

Не исполнять банковские платежи, обмены или сделки. Не подменять неизвестные данные нулём и не переносить финансовые расчёты в UI; платформенное покрытие расширяется отдельным решением.

### Блокирующие зависимости

- [#29](https://github.com/andrey-krasheninnikov/want-keep/issues/29) (`task-4.1`): выполнена.
- [#30](https://github.com/andrey-krasheninnikov/want-keep/issues/30) (`task-4.2`): выполнена.
- [#31](https://github.com/andrey-krasheninnikov/want-keep/issues/31) (`task-4.3`): открыта; блокирует выполнение.
- [#18](https://github.com/andrey-krasheninnikov/want-keep/issues/18) (`task-2.2`): выполнена.

### Границы изменений

- `backend/internal/credit/`

Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-005:** Счета показывают собственные, доступные, заблокированные и заёмные средства в пределах данных источника.
- **REQ-031:** Кредитные карты показывают задолженность, собственные средства, лимит, минимальный платёж и дату по данным источника.
- **REQ-032:** Грейс-период опирается на условия конкретной карты и показывает сумму и срок сохранения льготы.
- **REQ-033:** Накопления показывают фактические начисления и прогноз по ставкам, срокам, капитализации и денежным потокам.
- **REQ-039:** Отсутствующие курсы и неподдерживаемые активы не превращаются в нулевые суммы или условный паритет USD/USDT/USDC.
- **REQ-048:** Интеграции и браузерный сборщик выполняют только разрешённые операции чтения.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-005

- **Дано:** Источник сообщает собственные RUB 100, долг RUB 300 и кредитный лимит RUB 1 000.
- **Когда:** Строится сводка денег.
- **Тогда:** Кредитный лимит не увеличивает собственный капитал или доступный бюджет; отсутствующее поле отображается как неизвестное.
- **Уровень:** `integration`.

#### AC-031

- **Дано:** Покупка RUB 1 000 сделана с кредитки, затем долг погашен с собственного счёта.
- **Когда:** Формируется бюджет и сводка кредитки.
- **Тогда:** Покупка учтена один раз; погашение не второй расход; проценты и комиссии — отдельные расходы; неизвестный минимальный платёж не вычисляется догадкой.
- **Уровень:** `integration`.

#### AC-032

- **Дано:** Для карты подтверждены условия, выписка, исключения и крайняя дата.
- **Когда:** Совершаются покупка, частичное погашение и операция, исключённая из льготы.
- **Тогда:** Сумма и срок согласованы с подтверждёнными условиями; при нехватке условий отображается неизвестность, а не обещание сохранения льготы.
- **Уровень:** `contract`.

#### AC-048

- **Дано:** Сборщик имеет сессию личного кабинета с более широкими внешними правами.
- **Когда:** Возникают запрос на платёж, неподтверждённый маршрут или MFA/CAPTCHA.
- **Тогда:** Платёж и неизвестный маршрут блокируются; MFA/CAPTCHA передаётся владельцу, источник приостанавливается; остальные источники продолжают работать.
- **Уровень:** `integration`.

#### AC-070

- **Дано:** Банк передаёт баланс, но не условия грейса; ставка Earn имеет неизвестную базу начисления.
- **Когда:** Открываются прогнозы.
- **Тогда:** Баланс отображается; льгота и точный прогноз имеют причину недоступности; AI не извлекает гарантированную бизнес-логику из рекламной формулировки.
- **Уровень:** `contract+end-to-end`.

### Проверка результата

```sh
make test-go PKG=./internal/credit/... && make test-contract PROVIDER=credit
```

Синтетические контракты каждого банка покрывают грейс/исключения/частичное погашение; нет обещания льготы при неизвестных условиях.

Команды Make являются интерфейсами проверки; наличие target не доказывает реализацию соответствующей suite. Если требуемой suite ещё нет, реализовать её в рамках задачи; отсутствие suite не считается успешной проверкой. Фиксировать фактический результат каждого прогона. make docs-check проверяет документацию и исследовательские инструменты, но не финансовый runtime, браузеры или production. Live/manual-проверки, доступ и расходы фиксируются отдельно.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Show debt and grace eligibility from confirmed data.

**Status:** Open; ready-for-agent. The description is ready for implementation; open prerequisites and the entry conditions below govern execution. The label does not establish completion, live admission or production authorization; SDD Ready is not a blocker.

**Dependencies:** `task-4.1`, `task-4.2`, `task-4.3`, `task-2.2`.

**Kind:** `implementation`.

### Change and contracts

### Current behavior

Exact money types, the ledger, corrections, allocations and currency valuation are included in the base. External-data coverage and acceptance of this task's calculations require separate verification.

### Desired behavior

Map statement cycles, debt, own funds, minimum/due dates and grace eligibility from verified card-specific contracts. Domain rules handle exclusions, partial payments and term changes. Do not derive deadlines from marketing text or a universal graceDays value. Missing terms yield unknown; purchases/repayments/interest/fees retain distinct semantics.

### Key interfaces

Statement cycle, debt/own funds, minimum payment/due date and grace eligibility with provenance and verified terms.

### Out of scope

Do not execute bank payments, exchanges or trades. Do not replace unknown data with zero or move financial calculations into the UI; platform coverage requires a separate decision.

### Blocked by

- [#29](https://github.com/andrey-krasheninnikov/want-keep/issues/29) (`task-4.1`): completed.
- [#30](https://github.com/andrey-krasheninnikov/want-keep/issues/30) (`task-4.2`): completed.
- [#31](https://github.com/andrey-krasheninnikov/want-keep/issues/31) (`task-4.3`): open; blocks execution.
- [#18](https://github.com/andrey-krasheninnikov/want-keep/issues/18) (`task-2.2`): completed.

### Change boundaries

- `backend/internal/credit/`

Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-005:** Accounts distinguish owned, available, locked and borrowed amounts where the source provides them.
- **REQ-031:** Credit cards show debt, own funds, credit limit, minimum payment and due date from source data.
- **REQ-032:** Grace-period tracking uses the specific card's terms and shows the amount and deadline needed to preserve the benefit.
- **REQ-033:** Savings show actual accruals and forecasts using rates, terms, compounding and cash flows.
- **REQ-039:** Missing rates and unsupported assets never become zero amounts or assumed USD/USDT/USDC parity.
- **REQ-048:** Integrations and the browser collector perform authorized read operations only.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-005

- **Given:** The source reports RUB 100 owned, RUB 300 debt and a RUB 1,000 credit limit.
- **When:** A money summary is built.
- **Then:** The credit limit does not increase net worth or the spendable budget; missing fields are shown as unknown.
- **Level:** `integration`.

#### AC-031

- **Given:** A RUB 1,000 credit-card purchase is followed by repayment from an owned account.
- **When:** The budget and card summary are built.
- **Then:** The purchase is counted once; repayment is not another expense; interest and fees are separate expenses; an unknown minimum payment is not guessed.
- **Level:** `integration`.

#### AC-032

- **Given:** Card terms, statement, exclusions and deadline are confirmed.
- **When:** A purchase, partial repayment and grace-excluded transaction occur.
- **Then:** Amount and deadline follow confirmed terms; missing terms produce an unknown state rather than a promise of grace eligibility.
- **Level:** `contract`.

#### AC-048

- **Given:** The collector has a personal-account session with broader provider permissions.
- **When:** A payment request, unapproved route or MFA/CAPTCHA appears.
- **Then:** Payments and unknown routes are blocked; MFA/CAPTCHA is handed to the owner and that source pauses; other sources continue.
- **Level:** `integration`.

#### AC-070

- **Given:** A bank exposes balance but no grace terms; an Earn rate has an unknown accrual basis.
- **When:** Forecasts are opened.
- **Then:** Balance is shown; grace eligibility and exact forecasts explain unavailability; AI does not turn marketing wording into guaranteed business rules.
- **Level:** `contract+end-to-end`.

### Verification

```sh
make test-go PKG=./internal/credit/... && make test-contract PROVIDER=credit
```

Each bank's synthetic contracts cover grace/exclusions/partial repayment; unknown terms never promise eligibility.

Make commands are verification interfaces; the existence of a target does not establish an implemented suite. If a required suite is missing, implement it as part of the task; a missing suite is not a passing check. Record the actual outcome of each run. make docs-check validates documentation and research tooling, not financial runtime, browsers or production. Record live/manual checks, access and costs separately.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
