<!-- want-keep-task: task-7.4 -->
# task-7.4 — Создать редактор месячного бюджета / Create the monthly budget editor

## RU

Редактировать и утверждать план доходов и расходов по валютам.

**Состояние:** Не начато; задача ожидает собственные зависимости и entry gates.

**Зависимости:** `task-7.1`, `task-6.6`, `task-7.9`.

**Тип:** `implementation`.

### Изменение и контракты

Показать fixed/dated и flexible allocations, повторения, прогнозные доходы и версии утверждения. Реализовать копирование месяца, исполнение/частичное исполнение, отклонение AI-предложения и предупреждения о перерасходе/кассовом разрыве. Currency conversion — явный информационный срез, native budgets не сливаются.

### Границы изменений

- `web/src/features/budget/`

### Экранный контракт

### SCR-014 — Месячный план

`/plan`

**Вопрос:** Как распределить деньги на месяц?

**Главный ответ:** План/факт и обеспеченность по каждой валюте.

**Структура сверху вниз:** Месяц/семья/участник → ожидаемые доходы → обязательства → гибкие расходы → личное/общее → дефицит/остаток.

**Следующее действие:** Редактировать SCR-015, копировать месяц, календарь SCR-016, лимиты SCR-017.

**Объяснение и детализация:** Доход ожидаемый не полученный; копирование не переносит прошлый остаток, валютный эквивалент не покрывает дефицит.

**Права:** Оба видят; личное изменяет только владелец, совместное — любой участник.

Forms: FORM-10.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-14.

### SCR-015 — Редактор плана

`/plan/edit`

**Вопрос:** Что изменится после моего решения?

**Главный ответ:** Preview лимитов и обеспечения до сохранения.

**Структура сверху вниз:** Текущая версия → статьи/доходы/даты/доли → сравнение → подтвердить.

**Следующее действие:** Добавить/изменить FORM-10; сохранить → SCR-014, отменить с защитой несохранённого ввода.

**Объяснение и детализация:** Чужие личные строки только чтение; изменения общего плана уведомляют партнёра.

**Права:** Оба видят; личное изменяет только владелец, совместное — любой участник.

Forms: FORM-10, FORM-12.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-14, UISTATE-17.

### SCR-016 — Календарь бюджета

`/plan/calendar`

**Вопрос:** Когда нужны деньги и ожидаются поступления?

**Главный ответ:** Ближайшие даты с риском нехватки валюты.

**Структура сверху вниз:** Месяц → календарь и доступный списком agenda → доходы/платежи → факт исполнения/связь.

**Следующее действие:** Открыть статью FORM-10 или связанную SCR-010, посмотреть SCR-017.

**Объяснение и детализация:** Будущая дата не создаёт фактическую операцию; просроченное/исполненное различаются.

**Права:** Оба видят; личное изменяет только владелец, совместное — любой участник.

Forms: FORM-10.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-04.

### SCR-017 — Дневные лимиты

`/plan/limits`

**Вопрос:** Сколько можно сегодня и почему?

**Главный ответ:** Доступный и прогнозный варианты по валюте/категории/участнику.

**Структура сверху вниз:** Доступно сегодня → прогноз отдельно → оставшиеся дни → обязательства/резервы/факт → личное распределение.

**Следующее действие:** Раскрыть формулу, изменить свой план SCR-015 или резерв SCR-019.

**Объяснение и детализация:** Сумма личных лимитов не больше семейного; дефицит показан явно, недостающие входы не превращаются в ноль.

**Права:** Оба участника видят; действия проверяет сервер по членству и владельцу ресурса.

Forms: —.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-14.

#### FORM-10 — Статья плана и прогноз дохода

**Поля:** Месяц, тип обязательство/гибкий расход/доход, сумма/валюта, дата/повтор, категория, personal/shared, владелец/доли.

**Проверки и права:** Личное меняет владелец, общее любой member. Preview обеспечения/лимита по валютам, expectedRevision; утверждённый план AI меняет только по решению.

**Результат:** Новая версия плана/прогноза, уведомление партнёру; копирование не переносит остатки.

#### FORM-12 — Ответ и предложение AI

**Поля:** expectedRevision вопроса/предложения, subjectExpectedRevision операции и ровно choiceId либо свободный answer; для предложения явное apply/reject.

**Проверки и права:** Оба для операций, только владелец для личного плана/цели; actor не меняется текстом. Одновременный ответ проверяет версию; preview перед финансовым изменением.

**Результат:** Команда подтверждена, отказана, устарела или ожидает; ответ/основание и автор сохранены.

- **UISTATE-01 — Загрузка:** Скелетон структуры и подпись загрузки; суммы не подменяются нулями.
- **UISTATE-02 — Обновление:** Сохранить предыдущие данные и контекст, показать время последнего успеха; блокировать только конфликтующие действия.
- **UISTATE-03 — Пусто:** Объяснить полезный результат и предложить первое действие: счёт, чек, план или цель.
- **UISTATE-04 — Нет совпадений:** Сохранить фильтры, объяснить отсутствие результатов, предложить очистить условия.
- **UISTATE-05 — Частичные данные:** Назвать отсутствующий источник/период и последствия для суммы; доступные блоки работают; неизвестное обозначить отдельно.
- **UISTATE-06 — Устаревшие данные:** Показать дату последнего успеха и влияние на решение; дать обновить или перейти к подключению.
- **UISTATE-07 — Ошибка:** Понятная причина и следующий шаг у проблемного блока; ввод и исправные данные сохранить, диагностику раскрывать отдельно.
- **UISTATE-08 — Offline:** Показать отсутствие связи; не обещать сохранение. Чувствительные черновики только в памяти текущей вкладки, без новой offline-очереди.
- **UISTATE-09 — Сохранение:** Немедленно показать прогресс текущего действия и не допускать дублирующую отправку команды.
- **UISTATE-10 — Исход неизвестен:** Сохранить ID команды/ввод, запросить её результат; не создавать новую финансовую команду вслепую. После перезагрузки сверять серверный список недавних команд.
- **UISTATE-11 — Конфликт версии:** Показать авторов и различия, сохранить мой ввод; загрузить актуальную версию и дать повторно применить выбранные изменения после проверки.
- **UISTATE-12 — Недостаточно прав:** Финансовые данные доступны семье; запрещённое изменение объясняет владельца. Сервер отклоняет команду независимо от видимости кнопки.
- **UISTATE-13 — Сессия истекла:** Закрыть защищённое содержимое; вход для того же участника, безопасный возврат по внутреннему маршруту. Чужой вход не получает прежний черновик.
- **UISTATE-14 — Ожидание AI:** Отличать очередь, обработку, уточнение и паузу из-за лимита/API; обычный учёт доступен, результат не выдумывать.
- **UISTATE-16 — Подтверждено:** После подтверждённого сервером результата показать что изменилось, ссылку на объект и доступное исправление; не полагаться на исчезающий toast.
- **UISTATE-17 — Отмена:** Объяснить отсутствие нового подтверждённого результата, дать повторить явно; не выдавать отмену системного passkey за поломку.


Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-021:** AI меняет утверждённый бюджет, прогноз доходов или цели только по явному решению участника с правом на изменение.
- **REQ-023:** Единый семейный бюджет составляется на календарный месяц с суммами в исходных валютах и индивидуальными разрезами.
- **REQ-024:** План поддерживает обязательные расходы по датам и повторяемые платежи.
- **REQ-025:** Гибкие категории ограничивают траты за месяц и показывают остаток и перерасход.
- **REQ-026:** Прогнозируемые доходы имеют сумму, валюту, дату и отдельное состояние исполнения.
- **REQ-027:** План можно копировать; остатки и перерасход прошлого месяца автоматически не переносятся.
- **REQ-030:** Дневные лимиты показывают семейный и индивидуальный доступный/прогнозный остаток, по категориям и с отдельным обеспечением каждой валютой.
- **REQ-031:** Кредитные карты показывают задолженность, собственные средства, лимит, минимальный платёж и дату по данным источника.
- **REQ-054:** Интерфейс, чат и документация поддерживают RU/EN без изменения финансовой семантики.
- **REQ-064:** Оба участника видят все финансовые данные и изменяют операции; личные цели и части плана изменяет только их владелец.
- **REQ-066:** Все доходы и доступные средства входят в семейный пул; общий бюджет и личные разрезы используют один финансовый факт.
- **REQ-067:** Расходы и позиции чеков имеют личное или совместное назначение; общая доля по умолчанию 50/50 с исключениями статьи или покупки.
- **REQ-072:** Конкурирующие изменения, ответы на уточнения и отмены проверяют версию и текущие права, сохраняя обоих авторов.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

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

#### AC-054

- **Дано:** Есть русская и английская версии одной операции, бюджета и ошибки.
- **Когда:** Переключается язык.
- **Тогда:** Суммы, даты, валюты и смысл совпадают; форматирование локализовано, идентификаторы и категории пользователя не переводятся с потерей данных.
- **Уровень:** `end-to-end+static`.

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

#### AC-031

- **Дано:** Покупка RUB 1 000 сделана с кредитки, затем долг погашен с собственного счёта.
- **Когда:** Формируется бюджет и сводка кредитки.
- **Тогда:** Покупка учтена один раз; погашение не второй расход; проценты и комиссии — отдельные расходы; неизвестный минимальный платёж не вычисляется догадкой.
- **Уровень:** `integration`.

### Проверка результата

```sh
make test-web FILTER=budget && make e2e SCENARIO=planning
```

Копирование, approval/version conflict, future income и отчёт возврата не меняют утверждённый план скрытно.

Команды `make` — будущий контракт, создаваемый task-1.1; сейчас они не существуют. Live/paid/manual проверки отдельно фиксируют доступ и фактический результат. Исследования не обходят блокер отсутствующего доступа.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Edit and approve per-currency income/expense plans.

**Status:** Not started; the task awaits its own dependencies and entry gates.

**Dependencies:** `task-7.1`, `task-6.6`, `task-7.9`.

**Kind:** `implementation`.

### Change and contracts

Show fixed/dated and flexible allocations, recurrence, forecast income and approval versions. Implement month copying, fulfillment/partial fulfillment, AI proposal rejection and overspend/cash-shortfall feedback. Currency conversion is an explicit informational view; native budgets remain separate.

### Change boundaries

- `web/src/features/budget/`

### Screen contract

### SCR-014 — Monthly plan

`/plan`

**Question:** How should we allocate money this month?

**Primary answer:** Plan/actual and funding for each currency.

**Top-down structure:** Month/household/member → expected income → obligations → flexible spending → personal/shared → shortfall/remainder.

**Next action:** Edit SCR-015, copy month, calendar SCR-016, limits SCR-017.

**Explanation and details:** Expected income is not received; copying never carries old remainder, currency equivalent never covers a shortfall.

**Permissions:** Both read; only the owner edits personal resources, either member edits shared resources.

Forms: FORM-10.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-14.

### SCR-015 — Plan editor

`/plan/edit`

**Question:** What changes after my decision?

**Primary answer:** Allowance and funding preview before saving.

**Top-down structure:** Current revision → lines/income/dates/shares → comparison → confirm.

**Next action:** Add/edit FORM-10; save → SCR-014, cancel with unsaved-input protection.

**Explanation and details:** Partner personal lines are read-only; shared plan changes notify the partner.

**Permissions:** Both read; only the owner edits personal resources, either member edits shared resources.

Forms: FORM-10, FORM-12.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-14, UISTATE-17.

### SCR-016 — Budget calendar

`/plan/calendar`

**Question:** When is money due or expected?

**Primary answer:** Upcoming dates with currency shortfall risk.

**Top-down structure:** Month → calendar and accessible agenda list → income/payments → fulfillment/link.

**Next action:** Open line FORM-10 or linked SCR-010, view SCR-017.

**Explanation and details:** Future date creates no actual transaction; overdue/fulfilled remain distinct.

**Permissions:** Both read; only the owner edits personal resources, either member edits shared resources.

Forms: FORM-10.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-04.

### SCR-017 — Daily allowances

`/plan/limits`

**Question:** How much today and why?

**Primary answer:** Available and forecast variants by currency/category/member.

**Top-down structure:** Available today → separate forecast → remaining days → obligations/reserves/actual → personal allocation.

**Next action:** Expand formula, change own plan SCR-015 or reserve SCR-019.

**Explanation and details:** Personal allowances sum to no more than household cap; shortfall explicit and missing inputs never become zero.

**Permissions:** Both members can read; server checks membership and resource ownership for actions.

Forms: —.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-14.

#### FORM-10 — Plan line and income forecast

**Fields:** Month, obligation/flexible expense/income type, amount/currency, date/repeat, category, personal/shared, owner/shares.

**Validation and permissions:** Owner edits personal, any member shared. Currency funding/allowance preview, expectedRevision; AI changes approved plans only after a decision.

**Outcome:** New plan/forecast revision and partner notification; copying never rolls balances over.

#### FORM-12 — AI response and proposal

**Fields:** Question/proposal expectedRevision, transaction subjectExpectedRevision and exactly choiceId or free-text answer; explicit apply/reject for a proposal.

**Validation and permissions:** Either member for transactions, owner only for personal plan/goal; text cannot change actor. Concurrent response checks revision; preview before financial change.

**Outcome:** Command confirmed, rejected, stale or pending; response/reason and author retained.

- **UISTATE-01 — Loading:** Structural skeleton and loading label; amounts are never replaced by zero.
- **UISTATE-02 — Refreshing:** Keep previous data/context and last-success time; block only conflicting actions.
- **UISTATE-03 — Empty:** Explain the useful outcome and offer a first account, receipt, plan or goal action.
- **UISTATE-04 — No matches:** Keep filters, explain no results and offer to clear conditions.
- **UISTATE-05 — Partial data:** Name the missing source/period and its effect on the amount; available sections work and unknowns stay explicit.
- **UISTATE-06 — Stale data:** Show last-success date and impact on the decision; offer refresh or connection details.
- **UISTATE-07 — Error:** Plain cause and next step beside the affected section; preserve input/healthy data and expand diagnostics separately.
- **UISTATE-08 — Offline:** Show missing connectivity and do not promise saved data. Sensitive drafts remain only in current-tab memory, without a new offline queue.
- **UISTATE-09 — Saving:** Immediately show current-action progress and prevent duplicate command submission.
- **UISTATE-10 — Unknown outcome:** Keep command ID/input and query its result; never blindly create another financial command. After reload reconcile the server list of recent commands.
- **UISTATE-11 — Version conflict:** Show authors/differences and keep my input; load current version and allow chosen changes to be reapplied after validation.
- **UISTATE-12 — Insufficient permission:** Household can read financial data; forbidden edits explain ownership. Server rejects the command regardless of button visibility.
- **UISTATE-13 — Session expired:** Hide protected contents; require the same member to sign in and return through a safe internal route. Another identity never receives the prior draft.
- **UISTATE-14 — AI waiting:** Distinguish queued, processing, clarification and budget/API pause; ordinary accounting remains available and results are not invented.
- **UISTATE-16 — Confirmed:** After server-confirmed outcome show what changed, an object link and available correction; do not rely on a disappearing toast.
- **UISTATE-17 — Cancelled:** Explain that no new outcome was confirmed and offer explicit retry; cancelled system passkey prompts are not a malfunction.


Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-021:** AI changes an approved budget, income forecast or goals only on an explicit decision by a member authorized for the change.
- **REQ-023:** One household budget covers a calendar month in native currencies with individual views.
- **REQ-024:** The plan supports dated obligations and recurring payments.
- **REQ-025:** Flexible categories limit monthly spending and show remaining allowance and overspend.
- **REQ-026:** Forecast income has an amount, currency, date and separate fulfillment state.
- **REQ-027:** Plans can be copied; prior-month remaining allowances and overspend do not roll over automatically.
- **REQ-030:** Daily limits show household and individual available/forecast allowances, by category and with separate funding in each currency.
- **REQ-031:** Credit cards show debt, own funds, credit limit, minimum payment and due date from source data.
- **REQ-054:** UI, chat and documentation support RU/EN without changing financial semantics.
- **REQ-064:** Both members see all financial data and edit transactions; only the owner edits personal goals and plan portions.
- **REQ-066:** All income and available funds enter the household pool; household and individual budget views share one financial fact.
- **REQ-067:** Expenses and receipt items have personal or joint attribution; joint shares default to 50/50 with line or purchase overrides.
- **REQ-072:** Competing edits, clarification answers and reversals check revision and current permissions while retaining both authors.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

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

#### AC-054

- **Given:** Russian and English versions of the same transaction, budget and error exist.
- **When:** The language is switched.
- **Then:** Amounts, dates, currencies and meaning agree; formatting is localized while IDs and owner categories are not destructively translated.
- **Level:** `end-to-end+static`.

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

#### AC-031

- **Given:** A RUB 1,000 credit-card purchase is followed by repayment from an owned account.
- **When:** The budget and card summary are built.
- **Then:** The purchase is counted once; repayment is not another expense; interest and fees are separate expenses; an unknown minimum payment is not guessed.
- **Level:** `integration`.

### Verification

```sh
make test-web FILTER=budget && make e2e SCENARIO=planning
```

Copying, approval/version conflicts, future income and refund reporting do not silently change approved plans.

The `make` commands are a future contract established by task-1.1; they do not exist yet. Live/paid/manual checks separately record access and actual outcomes. Research does not bypass missing-access blockers.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
