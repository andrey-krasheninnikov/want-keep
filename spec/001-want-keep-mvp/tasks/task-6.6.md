<!-- want-keep-task: task-6.6 -->
# task-6.6 — Планировать месячный бюджет и доходы / Plan monthly budgets and income

## RU

Разделить утверждённый план, фактические платежи и ожидаемые поступления.

**Состояние:** Открыта; ready-for-agent. Описание готово к реализации; начало зависит от незакрытых задач и условий ниже. Метка не подтверждает выполнение, live-допуск или разрешение на production; SDD Ready не является блокером.

**Зависимости:** `task-2.7`, `task-2.6`, `task-6.1`, `task-6.2`, `task-2.8`.

**Тип:** `implementation`.

### Изменение и контракты

### Текущее поведение

Точные денежные типы, журнал, исправления, распределения и валютная оценка включены в базу. Полнота внешних данных и приёмка расчётов этой задачи проверяются отдельно.

### Требуемое поведение

Реализовать calendar-month планы по валютам, dated obligations, recurrence и flexible category envelopes, ожидаемые доходы с частичным исполнением. Сопоставление факта уменьшает незакрытое обязательство без double reserve; план не создаёт проводок. Копирование переносит определения, не факт/остатки; версия утверждения защищает предложения AI. Late refund пересчитывает исходный месяц и текущий cash flow раздельно. Один семейный план: личные строки редактирует их владелец, общие строки и семейный прогноз доходов — любой. Активация общего плана не утверждает чужие личные черновики; личная строка включается по решению её владельца. Общие расходы и доходы не дублировать в семейном и персональных видах.

### Ключевые контракты

План календарного месяца, обязательства и ожидаемые доходы, category envelopes, версии утверждения и личная/совместная принадлежность.

### Вне объёма

Не исполнять банковские платежи, обмены или сделки. Не подменять неизвестные данные нулём и не переносить финансовые расчёты в UI; платформенное покрытие расширяется отдельным решением.

### Блокирующие зависимости

- [#23](https://github.com/andrey-krasheninnikov/want-keep/issues/23) (`task-2.7`): выполнена.
- [#22](https://github.com/andrey-krasheninnikov/want-keep/issues/22) (`task-2.6`): выполнена.
- [#40](https://github.com/andrey-krasheninnikov/want-keep/issues/40) (`task-6.1`): выполнена.
- [#41](https://github.com/andrey-krasheninnikov/want-keep/issues/41) (`task-6.2`): открыта; блокирует выполнение.
- [#24](https://github.com/andrey-krasheninnikov/want-keep/issues/24) (`task-2.8`): выполнена.

### Границы изменений

- `backend/internal/budget/`

Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-010:** Возврат уменьшает расходы исходного месяца покупки, сохраняя дату реального поступления денег.
- **REQ-011:** Расход признаётся целиком по фактической оплате, включая годовые подписки.
- **REQ-016:** Позиции чека распределяют одну оплаченную сумму по категориям без дублирования итога.
- **REQ-021:** AI меняет утверждённый бюджет, прогноз доходов или цели только по явному решению участника с правом на изменение.
- **REQ-023:** Единый семейный бюджет составляется на календарный месяц с суммами в исходных валютах и индивидуальными разрезами.
- **REQ-024:** План поддерживает обязательные расходы по датам и повторяемые платежи.
- **REQ-025:** Гибкие категории ограничивают траты за месяц и показывают остаток и перерасход.
- **REQ-026:** Прогнозируемые доходы имеют сумму, валюту, дату и отдельное состояние исполнения.
- **REQ-027:** План можно копировать; остатки и перерасход прошлого месяца автоматически не переносятся.
- **REQ-030:** Дневные лимиты показывают семейный и индивидуальный доступный/прогнозный остаток, по категориям и с отдельным обеспечением каждой валютой.
- **REQ-031:** Кредитные карты показывают задолженность, собственные средства, лимит, минимальный платёж и дату по данным источника.
- **REQ-064:** Оба участника видят все финансовые данные и изменяют операции; личные цели и части плана изменяет только их владелец.
- **REQ-066:** Все доходы и доступные средства входят в семейный пул; общий бюджет и личные разрезы используют один финансовый факт.
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

#### AC-021

- **Дано:** Есть утверждённый бюджет и предложение перераспределения.
- **Когда:** Приходит новый расход, затем уполномоченный участник подтверждает предложенное изменение.
- **Тогда:** До подтверждения план неизменен; подтверждение применяет показанную версию предложения один раз; устаревшее предложение пересогласуется.
- **Уровень:** `integration`.

#### AC-023

- **Дано:** Задан часовой пояс бюджета и планы RUB 30 000 и USD 100.
- **Когда:** Операции приходят на границе месяцев.
- **Тогда:** Отнесение к месяцу следует выбранному часовому поясу; суммы разных валют не складываются без явного пересчёта.
- **Уровень:** `unit`.

#### AC-024

- **Дано:** Аренда запланирована на 5-е число, подписка повторяется ежемесячно.
- **Когда:** Наступает дата платежа и импортируется фактическое списание.
- **Тогда:** Плановая строка сама не создаёт расход; сопоставленный факт погашает обязательство без двойного резервирования.
- **Уровень:** `integration`.

#### AC-025

- **Дано:** На еду выделено RUB 6 000, потрачено RUB 6 200.
- **Когда:** Открывается бюджет.
- **Тогда:** Показан перерасход RUB 200; факт не скрыт и лимит другой категории не изменён автоматически.
- **Уровень:** `unit`.

#### AC-026

- **Дано:** Зарплата RUB 50 000 ожидается 20-го числа.
- **Когда:** Доход ещё не поступил, затем приходит RUB 45 000.
- **Тогда:** До поступления это только прогноз; факт RUB 45 000 сопоставляется отдельно, разница RUB 5 000 не становится доходом.
- **Уровень:** `integration`.

#### AC-027

- **Дано:** В августе осталось RUB 1 000 в одной категории и перерасход RUB 500 в другой.
- **Когда:** Владелец копирует план на сентябрь.
- **Тогда:** Копируются плановые значения и правила; августовские остатки, факт и перерасход не меняют сентябрьские лимиты.
- **Уровень:** `unit`.

#### AC-031

- **Дано:** Покупка RUB 1 000 сделана с кредитки, затем долг погашен с собственного счёта.
- **Когда:** Формируется бюджет и сводка кредитки.
- **Тогда:** Покупка учтена один раз; погашение не второй расход; проценты и комиссии — отдельные расходы; неизвестный минимальный платёж не вычисляется догадкой.
- **Уровень:** `integration`.

#### AC-066

- **Дано:** До аренды нет достаточных RUB; будущая зарплата запланирована после даты аренды; есть кредитный лимит.
- **Когда:** Рассчитывается прогноз и доход переносится на более позднюю дату.
- **Тогда:** Показан кассовый разрыв на дату аренды; кредитный лимит не выдаётся за доступные собственные деньги; основной план без решения владельца не меняется.
- **Уровень:** `unit+end-to-end`.

#### AC-078

- **Дано:** У A есть личная цель и статья плана; у семьи общая статья и операции обоих.
- **Когда:** B читает все данные, исправляет операцию A и общий план, затем пытается изменить личную цель/план A через API и AI.
- **Тогда:** Чтение, операции и общее изменение разрешены; личные план/цель A защищены сервером. Одного уполномоченного подтверждения достаточно, второй уведомлён.
- **Уровень:** `end-to-end`.

#### AC-080

- **Дано:** Зарплата поступила на счёт A, общая аренда оплачена B, у A нет доступного остатка.
- **Когда:** Строятся семейный бюджет, персональные расходы и обеспеченность по валютам.
- **Тогда:** Доход общий с сохранением получателя; аренда учтена в семье один раз и в личных видах по долям. Доступность семьи включает средства обоих без автоматического обмена валют и без кредитного лимита.
- **Уровень:** `integration`.

#### AC-081

- **Дано:** Чек RUB 1000 содержит общие продукты 600 и личные покупки A 100 и B 300.
- **Когда:** Чек заносит любой участник; AI применяет правила или уточняет неизвестное назначение.
- **Тогда:** Факт семьи 1000, A 400, B 600; доли суммируются точно. Исключение покупки приоритетнее статьи, затем 50/50; неоднозначная трата сохранена без вымышленной принадлежности.
- **Уровень:** `integration`.

#### AC-086

- **Дано:** A и B открыли одну версию операции или уточнения.
- **Когда:** Оба отправляют несовместимые изменения и повторяют один запрос.
- **Тогда:** Один результат применяется; второй получает конфликт с необходимостью перечитать состояние. Повтор не дублирует эффект; отмена создаёт новую проверенную revision и не стирает чужую последующую правку.
- **Уровень:** `integration`.

#### AC-091

- **Дано:** Январский чек: общие товары 600 (50/50), личные A 100, B 300; в феврале доли статьи стали 60/40.
- **Когда:** В феврале возвращаются общие товары на 200.
- **Тогда:** Январь уменьшается семье на 200, каждому на 100 с исходной исторической оценкой; февральская пропорция не меняет январь, cash date возврата остаётся февральской.
- **Уровень:** `integration`.

### Проверка результата

```sh
make test-go PKG=./internal/budget/... && make test-integration AREA=budget
```

Границы месяца, recurrence, частичное поступление, перерасход, возврат и copying сохраняют согласованный plan/actual.

Команды Make являются интерфейсами проверки; наличие target не доказывает реализацию соответствующей suite. Если требуемой suite ещё нет, реализовать её в рамках задачи; отсутствие suite не считается успешной проверкой. Фиксировать фактический результат каждого прогона. make docs-check проверяет документацию и исследовательские инструменты, но не финансовый runtime, браузеры или production. Live/manual-проверки, доступ и расходы фиксируются отдельно.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Separate approved plans, actual payments and expected receipts.

**Status:** Open; ready-for-agent. The description is ready for implementation; open prerequisites and the entry conditions below govern execution. The label does not establish completion, live admission or production authorization; SDD Ready is not a blocker.

**Dependencies:** `task-2.7`, `task-2.6`, `task-6.1`, `task-6.2`, `task-2.8`.

**Kind:** `implementation`.

### Change and contracts

### Current behavior

Exact money types, the ledger, corrections, allocations and currency valuation are included in the base. External-data coverage and acceptance of this task's calculations require separate verification.

### Desired behavior

Implement per-currency calendar-month plans, dated obligations, recurrence, flexible category envelopes and partially fulfilled expected income. Actual matching reduces outstanding obligations without double reserve; plans never create postings. Copy definitions, not actuals/remaining amounts; approval versions protect AI proposals. Late refunds separately recalculate the original month and current cash flow. One household plan: personal lines are edited by their owner; joint lines and household income forecasts by either member. Activating the joint plan does not approve another member’s personal drafts; a personal line is included on its owner’s decision. Do not duplicate joint expenses or income across household and individual views.

### Key interfaces

Calendar-month plan, obligations and expected income, category envelopes, approval revisions and personal/shared ownership.

### Out of scope

Do not execute bank payments, exchanges or trades. Do not replace unknown data with zero or move financial calculations into the UI; platform coverage requires a separate decision.

### Blocked by

- [#23](https://github.com/andrey-krasheninnikov/want-keep/issues/23) (`task-2.7`): completed.
- [#22](https://github.com/andrey-krasheninnikov/want-keep/issues/22) (`task-2.6`): completed.
- [#40](https://github.com/andrey-krasheninnikov/want-keep/issues/40) (`task-6.1`): completed.
- [#41](https://github.com/andrey-krasheninnikov/want-keep/issues/41) (`task-6.2`): open; blocks execution.
- [#24](https://github.com/andrey-krasheninnikov/want-keep/issues/24) (`task-2.8`): completed.

### Change boundaries

- `backend/internal/budget/`

Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-010:** A refund reduces expenses in the purchase month while preserving the actual cash receipt date.
- **REQ-011:** An expense is recognized in full on payment, including annual subscriptions.
- **REQ-016:** Receipt items allocate one paid amount across categories without duplicating the total.
- **REQ-021:** AI changes an approved budget, income forecast or goals only on an explicit decision by a member authorized for the change.
- **REQ-023:** One household budget covers a calendar month in native currencies with individual views.
- **REQ-024:** The plan supports dated obligations and recurring payments.
- **REQ-025:** Flexible categories limit monthly spending and show remaining allowance and overspend.
- **REQ-026:** Forecast income has an amount, currency, date and separate fulfillment state.
- **REQ-027:** Plans can be copied; prior-month remaining allowances and overspend do not roll over automatically.
- **REQ-030:** Daily limits show household and individual available/forecast allowances, by category and with separate funding in each currency.
- **REQ-031:** Credit cards show debt, own funds, credit limit, minimum payment and due date from source data.
- **REQ-064:** Both members see all financial data and edit transactions; only the owner edits personal goals and plan portions.
- **REQ-066:** All income and available funds enter the household pool; household and individual budget views share one financial fact.
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

#### AC-021

- **Given:** An approved budget and a reallocation proposal exist.
- **When:** A new expense arrives and an authorized member later confirms the proposal.
- **Then:** The plan stays unchanged until confirmation; confirmation applies the displayed proposal version once; a stale proposal must be reconfirmed.
- **Level:** `integration`.

#### AC-023

- **Given:** A budget timezone and RUB 30,000 and USD 100 plans are configured.
- **When:** Transactions arrive around the month boundary.
- **Then:** Month attribution follows the configured timezone; amounts in different currencies are not added without explicit conversion.
- **Level:** `unit`.

#### AC-024

- **Given:** Rent is planned for the 5th and a subscription recurs monthly.
- **When:** The due date arrives and the actual charge is imported.
- **Then:** The planned row does not itself create an expense; matched actual payment settles the obligation without double reservation.
- **Level:** `integration`.

#### AC-025

- **Given:** Food has a RUB 6,000 allocation and RUB 6,200 spent.
- **When:** The budget is opened.
- **Then:** RUB 200 overspend is visible; actual spending is not hidden and another category's limit is not changed automatically.
- **Level:** `unit`.

#### AC-026

- **Given:** RUB 50,000 salary is expected on the 20th.
- **When:** Income has not arrived, then RUB 45,000 is received.
- **Then:** Before receipt it remains a forecast; RUB 45,000 actual income is matched separately and the RUB 5,000 difference is not booked as income.
- **Level:** `integration`.

#### AC-027

- **Given:** August has RUB 1,000 left in one category and RUB 500 overspend in another.
- **When:** The owner copies the plan to September.
- **Then:** Planned values and rules are copied; August remaining amounts, actuals and overspend do not change September limits.
- **Level:** `unit`.

#### AC-031

- **Given:** A RUB 1,000 credit-card purchase is followed by repayment from an owned account.
- **When:** The budget and card summary are built.
- **Then:** The purchase is counted once; repayment is not another expense; interest and fees are separate expenses; an unknown minimum payment is not guessed.
- **Level:** `integration`.

#### AC-066

- **Given:** Available RUB cannot cover rent; salary is scheduled after rent; a credit limit exists.
- **When:** The forecast is calculated and the income date is moved later.
- **Then:** A rent-date cash shortfall is shown; credit is not presented as owned cash and the plan is not changed without an owner decision.
- **Level:** `unit+end-to-end`.

#### AC-078

- **Given:** A has a personal goal and plan line; the household has a joint line and both members’ transactions.
- **When:** B reads all data, edits A’s transaction and the joint plan, then attempts to change A’s personal goal/plan through API and AI.
- **Then:** Reads, transaction edits and joint changes succeed; A’s personal plan/goal are protected server-side. One authorized confirmation suffices and the other member is notified.
- **Level:** `end-to-end`.

#### AC-080

- **Given:** Salary arrived in A’s account, B paid joint rent and A has no available balance.
- **When:** The household budget, individual expenses and currency funding are calculated.
- **Then:** Income is pooled with recipient retained; rent appears once for the household and by shares in individual views. Household availability includes both members’ funds without automatic currency exchange or credit limits.
- **Level:** `integration`.

#### AC-081

- **Given:** A RUB 1,000 receipt contains joint groceries of 600 and personal purchases of A 100 and B 300.
- **When:** Either member enters the receipt; AI applies rules or clarifies unknown attribution.
- **Then:** Household actual is 1,000, A 400, B 600; shares sum exactly. Purchase override takes precedence over plan line, then 50/50; ambiguous spending persists without invented attribution.
- **Level:** `integration`.

#### AC-086

- **Given:** A and B opened the same transaction or clarification revision.
- **When:** Both submit conflicting edits and replay one request.
- **Then:** One result applies; the other receives a conflict requiring refresh. Replay does not duplicate effects; reversal creates a checked new revision without erasing the other member’s later edit.
- **Level:** `integration`.

#### AC-091

- **Given:** January receipt: joint items 600 (50/50), personal A 100, B 300; February plan shares became 60/40.
- **When:** Joint items worth 200 are returned in February.
- **Then:** January falls by 200 for the household and 100 for each member using original historical valuation; February shares do not rewrite January and refund cash date stays in February.
- **Level:** `integration`.

### Verification

```sh
make test-go PKG=./internal/budget/... && make test-integration AREA=budget
```

Month boundaries, recurrence, partial receipt, overspend, refunds and copying preserve coherent plan/actuals.

Make commands are verification interfaces; the existence of a target does not establish an implemented suite. If a required suite is missing, implement it as part of the task; a missing suite is not a passing check. Record the actual outcome of each run. make docs-check validates documentation and research tooling, not financial runtime, browsers or production. Record live/manual checks, access and costs separately.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
