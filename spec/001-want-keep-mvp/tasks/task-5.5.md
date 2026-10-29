<!-- want-keep-task: task-5.5 -->
# task-5.5 — Формировать обоснованные AI-инсайты / Generate grounded AI insights

## RU

Давать проверяемые объяснения расходов, доходов и выполнения плана.

**Состояние:** Открыта; ready-for-agent. Описание готово к реализации; начало зависит от незакрытых задач и условий ниже. Метка не подтверждает выполнение, live-допуск или разрешение на production; SDD Ready не является блокером.

**Зависимости:** `task-5.1`, `task-6.8`.

**Тип:** `implementation`.

### Изменение и контракты

### Текущее поведение

Приватные вложения, журнал операций, проверка версий и gateway с контролем расходов включены в базу. Обработка чеков, общий чат и продуктовые инсайты имеют отдельные критерии приёмки.

### Требуемое поведение

Передавать AI рассчитанные доменом срезы, coverage и ссылки на операции; суммы/выводы валидировать против этих данных. Генерировать дневные/месячные сводки и ответы по запросу, помечать прогнозы и неполноту. Предложение нового бюджета остаётся proposal до решения владельца. Перегенерация учитывает версии данных и бюджет API, не запускается на каждый render. Применить лимиты/модель из evidence/openai.md; объяснения проверять на противоречия источникам, forecast/partial и релевантность следующего действия. Числовой score eval не доказывает качество объяснения; добавить ручной протокол проверки. Исследование task-0.8 завершено: использовать gpt-5.6-terra xhigh и финальную strict-схему из evidence/openai.prompts.json; Luna/Sol/MiniMax/DeepSeek автоматически не подключать. Финальный xhigh eval 206/206 не заменяет runtime/locale проверки. reasoning.effort=xhigh; никаких автоматических downgrade при лимите $50. Статус записи формирует приложение, не объяснение модели.

### Ключевые контракты

Доменные срезы отчётов и прогнозов с coverage, ссылки на операции, budget-controlled gateway и предложения плана без автоматического утверждения.

### Вне объёма

Не менять согласованную модель и лимит расходов, не предоставлять модели секреты или платёжные инструменты. Пользовательские экраны и соседние сценарии реализуются своими задачами.

### Блокирующие зависимости

- [#35](https://github.com/andrey-krasheninnikov/want-keep/issues/35) (`task-5.1`): выполнена.
- [#47](https://github.com/andrey-krasheninnikov/want-keep/issues/47) (`task-6.8`): открыта; блокирует выполнение.

### Границы изменений

- `backend/internal/insights/`

Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-020:** AI-инсайты по доходам и расходам ссылаются на проверяемые данные и отделяют прогноз от факта.
- **REQ-021:** AI меняет утверждённый бюджет, прогноз доходов или цели только по явному решению участника с правом на изменение.
- **REQ-030:** Дневные лимиты показывают семейный и индивидуальный доступный/прогнозный остаток, по категориям и с отдельным обеспечением каждой валютой.
- **REQ-037:** Исторические расходы используют зафиксированную оценку на дату операции, текущий капитал — актуальную оценку.
- **REQ-051:** AI ограничен бюджетом $50/месяц и деградирует в очередь ожидания без остановки обычного учёта.
- **REQ-052:** Дашборд объединяет счета, план/факт, доходы, расходы, цели и дневные лимиты с детализацией.
- **REQ-055:** Веб-приложение предназначено для ноутбука macOS в Chrome и Arc; изменение окна и масштаба сохраняет доступность ежедневного учёта.
- **REQ-056:** Развёртывание укладывается в $40/месяц на сервер в DE/NL/BG; отдельные платные источники не используются.
- **REQ-066:** Все доходы и доступные средства входят в семейный пул; общий бюджет и личные разрезы используют один финансовый факт.
- **REQ-069:** Резервы личных и совместных целей задаются явно; совместные цели отображаются отдельным общим блоком без персональных долей.
- **REQ-070:** Сумма индивидуальных дневных лимитов не превышает семейный предел одной валюты; счёт плательщика не меняет долю расходов.
- **REQ-071:** Один общий чат сохраняет автора сообщения и проверяет полномочия инициатора AI-команды при исполнении.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-020

- **Дано:** В двух месяцах известны расходы по категориям; один источник устарел.
- **Когда:** AI формирует месячный инсайт.
- **Тогда:** Числа воспроизводятся отчётом; указаны период, связанные операции/срезы и неполнота; прогноз не представлен как полученный доход.
- **Уровень:** `integration`.

#### AC-021

- **Дано:** Есть утверждённый бюджет и предложение перераспределения.
- **Когда:** Приходит новый расход, затем уполномоченный участник подтверждает предложенное изменение.
- **Тогда:** До подтверждения план неизменен; подтверждение применяет показанную версию предложения один раз; устаревшее предложение пересогласуется.
- **Уровень:** `integration`.

#### AC-030

- **Дано:** Есть RUB-бюджет, будущая зарплата, обязательный платёж, резерв цели и USDT на другом счёте.
- **Когда:** Рассчитываются лимиты на оставшиеся дни месяца.
- **Тогда:** Доступный RUB-лимит исключает будущую зарплату, USDT, долг и резервы; прогноз учитывает даты поступлений и показывает кассовые разрывы; общий предел не размножается по категориям.
- **Уровень:** `integration`.

#### AC-037

- **Дано:** Расход USD 10 оценён в RUB 900; текущий курс стал RUB 100/USD.
- **Когда:** Обновляются котировки и дашборд.
- **Тогда:** Исторический расход остаётся RUB 900, USD-остаток переоценивается; видны источник, время курса и отдельное курсовое изменение.
- **Уровень:** `integration`.

#### AC-051

- **Дано:** OpenAI недоступен либо израсходован разрешённый бюджет с резервами текущих запросов.
- **Когда:** Поступают новый импорт, ручной расход и запрос AI.
- **Тогда:** Учёт и расчёты доступны; статус AI ожидает; новые платные запросы не запускаются сверх разрешённого резерва; неизвестная стоимость не освобождается молча.
- **Уровень:** `integration`.

#### AC-052

- **Дано:** Подготовлен месяц с несколькими валютами, переводом, расходами, целью и неполным источником.
- **Когда:** Владелец открывает дашборд и раскрывает показатели.
- **Тогда:** Итоги согласованы с учётом; видны состав, фильтры, валюты, свежесть и неполнота; скрытого двойного учёта нет.
- **Уровень:** `end-to-end`.

#### AC-076

- **Дано:** Подготовлен синтетический набор сотен операций со сложными переводами, чеками и эталонными ответами.
- **Когда:** Выполняются AI-eval, нагрузочная проверка и ручной дневной сценарий.
- **Тогда:** Отчёт показывает ошибки, уточнения, латентность, токены/стоимость и время пользователя; финансовые инварианты проходят, бюджет оценивается по измерению; непроверенное качество не объявлено доказанным.
- **Уровень:** `manual+integration`.

#### AC-080

- **Дано:** Зарплата поступила на счёт A, общая аренда оплачена B, у A нет доступного остатка.
- **Когда:** Строятся семейный бюджет, персональные расходы и обеспеченность по валютам.
- **Тогда:** Доход общий с сохранением получателя; аренда учтена в семье один раз и в личных видах по долям. Доступность семьи включает средства обоих без автоматического обмена валют и без кредитного лимита.
- **Уровень:** `integration`.

#### AC-083

- **Дано:** Есть личные цели A и B, общая цель RUB 600000 и виртуальный резерв RUB 10000.
- **Когда:** Оба открывают личные и семейный виды, увеличивают разрешённый резерв и связывают выделенный счёт.
- **Тогда:** Общая цель показана целиком в общем блоке; персональные половины не создаются. Резерв уменьшает семейную доступность один раз; нет автоматического распределения свободных средств на цели.
- **Уровень:** `end-to-end`.

#### AC-084

- **Дано:** Осталось 10 дней, K=RUB 1000, положительные персональные остатки A=3000 и B=1000.
- **Когда:** Рассчитаны доступные лимиты; затем меняется плательщик общей покупки или дата ожидаемого дохода.
- **Тогда:** Семейный лимит 100/день, A 75, B 25; суммы не дублируют K. Плательщик не меняет доли; перенос дохода меняет прогноз, не доступный остаток. Неизвестное назначение не скрывает факт расхода.
- **Уровень:** `integration`.

#### AC-085

- **Дано:** Оба видят общий чат; A имеет личную цель, B просит AI изменить её от имени A.
- **Когда:** Модель предлагает действие, а затем A подтверждает новую адресованную ему версию предложения.
- **Тогда:** Сообщение B не выдаёт полномочия A; до разрешённого подтверждения изменения нет. Аудит хранит автора сообщения, подтвердившего и AI-основание; секретов в чате нет.
- **Уровень:** `integration`.

### Проверка результата

```sh
make eval-ai SUITE=insights && make test-integration AREA=insights
```

Числа и ссылки воспроизводимы; недоказанные выводы не представлены как факт, plan changes требуют решения.

Команды Make являются интерфейсами проверки; наличие target не доказывает реализацию соответствующей suite. Если требуемой suite ещё нет, реализовать её в рамках задачи; отсутствие suite не считается успешной проверкой. Фиксировать фактический результат каждого прогона. make docs-check проверяет документацию и исследовательские инструменты, но не финансовый runtime, браузеры или production. Live/manual-проверки, доступ и расходы фиксируются отдельно.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Provide verifiable explanations of spending, income and plan progress.

**Status:** Open; ready-for-agent. The description is ready for implementation; open prerequisites and the entry conditions below govern execution. The label does not establish completion, live admission or production authorization; SDD Ready is not a blocker.

**Dependencies:** `task-5.1`, `task-6.8`.

**Kind:** `implementation`.

### Change and contracts

### Current behavior

Private attachments, the transaction ledger, revision checks and the budget-controlled gateway are included in the base. Receipt processing, shared chat and product insights have separate acceptance criteria.

### Desired behavior

Provide AI with domain-calculated aggregates, coverage and transaction references; validate figures/claims against those inputs. Generate daily/monthly and on-demand explanations with forecast/incompleteness markers. A suggested budget stays a proposal until the owner decides. Regeneration uses data revisions and API budget, not each render. Apply model/limits from evidence/openai.en.md; check explanations for source contradictions, forecast/partial labeling and useful next actions. Numeric evaluation scores do not establish explanation quality; include a manual review protocol. task-0.8 research is complete: use gpt-5.6-terra xhigh and the final strict schema in evidence/openai.prompts.json; do not automatically enable Luna/Sol/MiniMax/DeepSeek. The final xhigh evaluation 206/206 does not replace runtime/locale checks. reasoning.effort=xhigh; no automatic downgrade at the USD 50 cap. The application supplies persistence status, not the model explanation.

### Key interfaces

Domain report and forecast slices with coverage, transaction references, budget-controlled gateway and plan proposals without automatic approval.

### Out of scope

Do not change the agreed model or spending cap, or expose secrets or payment tools to the model. Product screens and adjacent workflows remain with their owning tasks.

### Blocked by

- [#35](https://github.com/andrey-krasheninnikov/want-keep/issues/35) (`task-5.1`): completed.
- [#47](https://github.com/andrey-krasheninnikov/want-keep/issues/47) (`task-6.8`): open; blocks execution.

### Change boundaries

- `backend/internal/insights/`

Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-020:** AI income/expense insights reference verifiable data and separate forecasts from facts.
- **REQ-021:** AI changes an approved budget, income forecast or goals only on an explicit decision by a member authorized for the change.
- **REQ-030:** Daily limits show household and individual available/forecast allowances, by category and with separate funding in each currency.
- **REQ-037:** Historical expenses use a fixed transaction-date valuation; current wealth uses a current valuation.
- **REQ-051:** AI is limited to $50/month and degrades to a waiting queue without stopping ordinary accounting.
- **REQ-052:** The dashboard combines accounts, plan/actuals, income, expenses, goals and daily limits with drill-down.
- **REQ-055:** The web app targets macOS laptops in Chrome and Arc; window resizing and zoom preserve daily accounting access.
- **REQ-056:** Deployment fits $40/month for a server in DE/NL/BG; no separately paid data sources are used.
- **REQ-066:** All income and available funds enter the household pool; household and individual budget views share one financial fact.
- **REQ-069:** Personal and joint goal reservations are explicit; joint goals appear in a separate shared block without personal shares.
- **REQ-070:** Individual daily allowances sum to no more than the household ceiling in one currency; the payer’s account does not change expense shares.
- **REQ-071:** One shared chat retains message authors and checks the AI command initiator’s authority at execution.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-020

- **Given:** Category expenses are known for two months and one source is stale.
- **When:** AI generates a monthly insight.
- **Then:** Figures are reproducible from reports; period, linked transactions/aggregates and incompleteness are shown; forecast income is not presented as received.
- **Level:** `integration`.

#### AC-021

- **Given:** An approved budget and a reallocation proposal exist.
- **When:** A new expense arrives and an authorized member later confirms the proposal.
- **Then:** The plan stays unchanged until confirmation; confirmation applies the displayed proposal version once; a stale proposal must be reconfirmed.
- **Level:** `integration`.

#### AC-030

- **Given:** There is a RUB budget, future salary, an obligation, a goal reservation and USDT in another account.
- **When:** Limits are calculated for the remaining days of the month.
- **Then:** Available RUB allowance excludes future salary, USDT, debt and reservations; the forecast uses receipt dates and shows cash shortfalls; the overall ceiling is not duplicated across categories.
- **Level:** `integration`.

#### AC-037

- **Given:** A USD 10 expense was valued at RUB 900; the current rate becomes RUB 100/USD.
- **When:** Quotes and dashboard refresh.
- **Then:** Historical expense remains RUB 900 and the USD balance is revalued; rate source/time and separate FX change are visible.
- **Level:** `integration`.

#### AC-051

- **Given:** OpenAI is unavailable or the allowed budget including in-flight reservations is exhausted.
- **When:** A new import, manual expense and AI request arrive.
- **Then:** Accounting and calculations remain available; AI status is waiting; no new paid calls exceed the allowed reservation; unknown cost is not silently released.
- **Level:** `integration`.

#### AC-052

- **Given:** A month includes multiple currencies, a transfer, expenses, a goal and an incomplete source.
- **When:** The owner opens the dashboard and drills into metrics.
- **Then:** Totals reconcile to accounting; composition, filters, currencies, freshness and incompleteness are visible; no hidden double counting occurs.
- **Level:** `end-to-end`.

#### AC-076

- **Given:** A synthetic hundreds-of-transactions set includes difficult transfers, receipts and reference answers.
- **When:** AI evaluation, load checks and a manual daily flow run.
- **Then:** The report shows errors, clarifications, latency, tokens/cost and user time; financial invariants pass and cost uses measurements; untested quality is not claimed as proven.
- **Level:** `manual+integration`.

#### AC-080

- **Given:** Salary arrived in A’s account, B paid joint rent and A has no available balance.
- **When:** The household budget, individual expenses and currency funding are calculated.
- **Then:** Income is pooled with recipient retained; rent appears once for the household and by shares in individual views. Household availability includes both members’ funds without automatic currency exchange or credit limits.
- **Level:** `integration`.

#### AC-083

- **Given:** There are personal goals of A and B, a joint RUB 600,000 goal and a RUB 10,000 virtual reserve.
- **When:** Both open individual and household views, increase an authorized reserve and link a dedicated account.
- **Then:** The joint goal appears whole in the shared block; personal halves are not created. The reserve reduces household availability once; free funds are not automatically allocated to goals.
- **Level:** `end-to-end`.

#### AC-084

- **Given:** 10 days remain, K=RUB 1,000 and positive individual remainders are A=3,000 and B=1,000.
- **When:** Available allowances are calculated; then a joint purchase payer or expected-income date changes.
- **Then:** Household allowance is 100/day, A 75, B 25; totals do not duplicate K. Payer does not change shares; rescheduling income changes forecast, not available funds. Unknown attribution does not hide actual expense.
- **Level:** `integration`.

#### AC-085

- **Given:** Both see the shared chat; A has a personal goal and B asks AI to change it as A.
- **When:** The model proposes an action and A later confirms a new proposal version addressed to A.
- **Then:** B’s message grants no authority of A; nothing changes before authorized confirmation. Audit records message author, approver and AI rationale; chat contains no secrets.
- **Level:** `integration`.

### Verification

```sh
make eval-ai SUITE=insights && make test-integration AREA=insights
```

Figures and references reproduce; unsupported conclusions are not facts and plan changes require an owner decision.

Make commands are verification interfaces; the existence of a target does not establish an implemented suite. If a required suite is missing, implement it as part of the task; a missing suite is not a passing check. Record the actual outcome of each run. make docs-check validates documentation and research tooling, not financial runtime, browsers or production. Record live/manual checks, access and costs separately.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
