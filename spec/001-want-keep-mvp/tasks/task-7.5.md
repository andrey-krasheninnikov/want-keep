<!-- want-keep-task: task-7.5 -->
# task-7.5 — Показать цели и резервирование / Show goals and reservations

## RU

Управлять целями с понятным влиянием на доступные деньги.

**Состояние:** Не начато; задача ожидает собственные зависимости и entry gates.

**Зависимости:** `task-7.1`, `task-6.7`, `task-7.9`.

**Тип:** `implementation`.

### Изменение и контракты

Реализовать цель сумма/валюта/срок, virtual/dedicated выбор, funding account, прогресс и доступный остаток. До подтверждения показать влияние резерва и смены режима; обработать concurrent/version errors. Не предлагать перевод денег как исполняемое финансовое действие.

### Границы изменений

- `web/src/features/goals/`

### Экранный контракт

### SCR-018 — Цели

`/goals`

**Вопрос:** Как продвигаются наши цели?

**Главный ответ:** Прогресс личных целей и отдельный общий блок совместных.

**Структура сверху вниз:** Совместные цели → личные группы → срок/прогресс/резерв → доступно отложить по валюте.

**Следующее действие:** Создать FORM-11 или открыть SCR-019.

**Объяснение и детализация:** Общий резерв учитывается один раз, у совместных целей нет персональных долей.

**Права:** Оба видят; личное изменяет только владелец, совместное — любой участник.

Forms: FORM-11.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-04.

### SCR-019 — Карточка цели

`/goals/:id`

**Вопрос:** Сколько можно отложить без риска бюджету?

**Главный ответ:** Прогресс и влияние предлагаемого резерва на доступные деньги.

**Структура сверху вниз:** Цель/срок → накоплено и прогноз → резерв/выделенный счёт → preview изменения → история.

**Следующее действие:** Изменить резерв FORM-11, открыть выделенный SCR-008 или бюджет SCR-014.

**Объяснение и детализация:** Нет удвоения через выделенный счёт; достижение подтверждается фактом, не прогнозом.

**Права:** Оба видят; личное изменяет только владелец, совместное — любой участник.

Forms: FORM-11, FORM-12.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-14.

#### FORM-11 — Цель и резерв

**Поля:** Название, personal/shared, владелец если личная, сумма/валюта/срок; явный резерв или выделенный счёт; изменение суммы резерва.

**Проверки и права:** Личную меняет владелец, совместную любой member. Preview свободных денег/дневного лимита в исходной валюте; no double reserve; совместная без личных долей.

**Результат:** Цель/резерв обновлены один раз, сообщение партнёру и audit.

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


Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-021:** AI меняет утверждённый бюджет, прогноз доходов или цели только по явному решению участника с правом на изменение.
- **REQ-028:** Личная или совместная цель содержит сумму, валюту, срок и способ накопления: явный резерв либо выделенный счёт.
- **REQ-029:** Одни средства нельзя одновременно зарезервировать на несколько целей или повторно учесть через выделенный счёт.
- **REQ-030:** Дневные лимиты показывают семейный и индивидуальный доступный/прогнозный остаток, по категориям и с отдельным обеспечением каждой валютой.
- **REQ-054:** Интерфейс, чат и документация поддерживают RU/EN без изменения финансовой семантики.
- **REQ-064:** Оба участника видят все финансовые данные и изменяют операции; личные цели и части плана изменяет только их владелец.
- **REQ-069:** Резервы личных и совместных целей задаются явно; совместные цели отображаются отдельным общим блоком без персональных долей.
- **REQ-070:** Сумма индивидуальных дневных лимитов не превышает семейный предел одной валюты; счёт плательщика не меняет долю расходов.
- **REQ-072:** Конкурирующие изменения, ответы на уточнения и отмены проверяют версию и текущие права, сохраняя обоих авторов.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-021

- **Дано:** Есть утверждённый бюджет и предложение перераспределения.
- **Когда:** Приходит новый расход, затем уполномоченный участник подтверждает предложенное изменение.
- **Тогда:** До подтверждения план неизменен; подтверждение применяет показанную версию предложения один раз; устаревшее предложение пересогласуется.
- **Уровень:** `integration`.

#### AC-028

- **Дано:** Созданы цель покупки и цель накопления в USD.
- **Когда:** Одна цель получает резерв, другая связывается с выделенным счётом.
- **Тогда:** Показаны прогресс, остаток до цели и срок; перемещение на собственный накопительный счёт не становится потребительским расходом.
- **Уровень:** `integration`.

#### AC-029

- **Дано:** На счёте USD 100 уже зарезервировано USD 80.
- **Когда:** Вторая цель запрашивает USD 30 либо тот же резерв дублируется ссылкой на счёт.
- **Тогда:** Операция превышения отклоняется атомарно; свободно USD 20; параллельные запросы не обходят ограничение.
- **Уровень:** `integration`.

#### AC-030

- **Дано:** Есть RUB-бюджет, будущая зарплата, обязательный платёж, резерв цели и USDT на другом счёте.
- **Когда:** Рассчитываются лимиты на оставшиеся дни месяца.
- **Тогда:** Доступный RUB-лимит исключает будущую зарплату, USDT, долг и резервы; прогноз учитывает даты поступлений и показывает кассовые разрывы; общий предел не размножается по категориям.
- **Уровень:** `integration`.

#### AC-054

- **Дано:** Есть русская и английская версии одной операции, бюджета и ошибки.
- **Когда:** Переключается язык.
- **Тогда:** Суммы, даты, валюты и смысл совпадают; форматирование локализовано, идентификаторы и категории пользователя не переводятся с потерей данных.
- **Уровень:** `end-to-end+static`.

#### AC-067

- **Дано:** USD 100 размещены на выделенном счёте цели; ещё USD 50 на расходном.
- **Когда:** Строятся капитал, прогресс и дневной лимит.
- **Тогда:** Капитал USD 150, прогресс USD 100; доступно к тратам не более USD 50, резерв не вычтен второй раз.
- **Уровень:** `unit`.

#### AC-078

- **Дано:** У A есть личная цель и статья плана; у семьи общая статья и операции обоих.
- **Когда:** B читает все данные, исправляет операцию A и общий план, затем пытается изменить личную цель/план A через API и AI.
- **Тогда:** Чтение, операции и общее изменение разрешены; личные план/цель A защищены сервером. Одного уполномоченного подтверждения достаточно, второй уведомлён.
- **Уровень:** `end-to-end`.

#### AC-083

- **Дано:** Есть личные цели A и B, общая цель RUB 600000 и виртуальный резерв RUB 10000.
- **Когда:** Оба открывают личные и семейный виды, увеличивают разрешённый резерв и связывают выделенный счёт.
- **Тогда:** Общая цель показана целиком в общем блоке; персональные половины не создаются. Резерв уменьшает семейную доступность один раз; нет автоматического распределения свободных средств на цели.
- **Уровень:** `end-to-end`.

#### AC-092

- **Дано:** Свободно RUB 1000; оба пытаются зарезервировать по 800 для разрешённых целей.
- **Когда:** Команды исполняются одновременно.
- **Тогда:** Проверка общего доступного остатка и резерв атомарны: проходит максимум одна команда; отказ не уменьшает другой резерв, оба видят актуальный остаток.
- **Уровень:** `integration`.

#### AC-086

- **Дано:** A и B открыли одну версию операции или уточнения.
- **Когда:** Оба отправляют несовместимые изменения и повторяют один запрос.
- **Тогда:** Один результат применяется; второй получает конфликт с необходимостью перечитать состояние. Повтор не дублирует эффект; отмена создаёт новую проверенную revision и не стирает чужую последующую правку.
- **Уровень:** `integration`.

### Проверка результата

```sh
make test-web FILTER=goals && make e2e SCENARIO=goals
```

Резервы и смена режима отражают атомарный доменный результат; двойная аллокация отклоняется понятно.

Команды `make` — будущий контракт, создаваемый task-1.1; сейчас они не существуют. Live/paid/manual проверки отдельно фиксируют доступ и фактический результат. Исследования не обходят блокер отсутствующего доступа.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Manage goals with understandable effects on spendable money.

**Status:** Not started; the task awaits its own dependencies and entry gates.

**Dependencies:** `task-7.1`, `task-6.7`, `task-7.9`.

**Kind:** `implementation`.

### Change and contracts

Implement goal amount/currency/deadline, virtual/dedicated choice, funding account, progress and remaining spendable balance. Show reservation/mode-switch impact before confirmation and handle concurrency/version errors. Do not offer actual money movement as an executable action.

### Change boundaries

- `web/src/features/goals/`

### Screen contract

### SCR-018 — Goals

`/goals`

**Question:** How are our goals progressing?

**Primary answer:** Personal goal progress and a separate joint-goal block.

**Top-down structure:** Joint goals → personal groups → deadline/progress/reserve → available to reserve by currency.

**Next action:** Create FORM-11 or open SCR-019.

**Explanation and details:** Joint reserve counted once, joint goals have no personal shares.

**Permissions:** Both read; only the owner edits personal resources, either member edits shared resources.

Forms: FORM-11.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-04.

### SCR-019 — Goal details

`/goals/:id`

**Question:** How much can we reserve without risking the budget?

**Primary answer:** Progress and proposed reserve impact on available funds.

**Top-down structure:** Goal/deadline → saved and forecast → reserve/dedicated account → change preview → history.

**Next action:** Change reserve FORM-11, open dedicated SCR-008 or budget SCR-014.

**Explanation and details:** Dedicated account never doubles reserve; achievement uses actual data, not forecast.

**Permissions:** Both read; only the owner edits personal resources, either member edits shared resources.

Forms: FORM-11, FORM-12.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-14.

#### FORM-11 — Goal and reserve

**Fields:** Name, personal/shared, personal owner if applicable, amount/currency/deadline; explicit reserve or dedicated account; reserve delta.

**Validation and permissions:** Owner edits personal, any member joint. Preview free money/daily allowance in native currency; no double reserve; joint goals have no personal shares.

**Outcome:** Goal/reserve updated once, partner notification and audit.

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


Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-021:** AI changes an approved budget, income forecast or goals only on an explicit decision by a member authorized for the change.
- **REQ-028:** A personal or joint goal has an amount, currency, deadline and funding mode: an explicit reservation or dedicated account.
- **REQ-029:** The same money cannot be reserved for multiple goals or counted again through a dedicated account.
- **REQ-030:** Daily limits show household and individual available/forecast allowances, by category and with separate funding in each currency.
- **REQ-054:** UI, chat and documentation support RU/EN without changing financial semantics.
- **REQ-064:** Both members see all financial data and edit transactions; only the owner edits personal goals and plan portions.
- **REQ-069:** Personal and joint goal reservations are explicit; joint goals appear in a separate shared block without personal shares.
- **REQ-070:** Individual daily allowances sum to no more than the household ceiling in one currency; the payer’s account does not change expense shares.
- **REQ-072:** Competing edits, clarification answers and reversals check revision and current permissions while retaining both authors.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-021

- **Given:** An approved budget and a reallocation proposal exist.
- **When:** A new expense arrives and an authorized member later confirms the proposal.
- **Then:** The plan stays unchanged until confirmation; confirmation applies the displayed proposal version once; a stale proposal must be reconfirmed.
- **Level:** `integration`.

#### AC-028

- **Given:** A purchase goal and a USD savings goal exist.
- **When:** One goal receives a reservation and the other is linked to a dedicated account.
- **Then:** Progress, remaining target and deadline are shown; moving money to an owned savings account is not a consumer expense.
- **Level:** `integration`.

#### AC-029

- **Given:** USD 80 of an account's USD 100 is already reserved.
- **When:** A second goal requests USD 30 or an account link duplicates the reservation.
- **Then:** The over-allocation is rejected atomically; USD 20 remains free; concurrent requests cannot bypass the limit.
- **Level:** `integration`.

#### AC-030

- **Given:** There is a RUB budget, future salary, an obligation, a goal reservation and USDT in another account.
- **When:** Limits are calculated for the remaining days of the month.
- **Then:** Available RUB allowance excludes future salary, USDT, debt and reservations; the forecast uses receipt dates and shows cash shortfalls; the overall ceiling is not duplicated across categories.
- **Level:** `integration`.

#### AC-054

- **Given:** Russian and English versions of the same transaction, budget and error exist.
- **When:** The language is switched.
- **Then:** Amounts, dates, currencies and meaning agree; formatting is localized while IDs and owner categories are not destructively translated.
- **Level:** `end-to-end+static`.

#### AC-067

- **Given:** USD 100 is in a dedicated goal account and USD 50 in a spending account.
- **When:** Wealth, progress and the daily limit are built.
- **Then:** Wealth is USD 150 and progress USD 100; spendable cash is at most USD 50 and the reservation is not deducted twice.
- **Level:** `unit`.

#### AC-078

- **Given:** A has a personal goal and plan line; the household has a joint line and both members’ transactions.
- **When:** B reads all data, edits A’s transaction and the joint plan, then attempts to change A’s personal goal/plan through API and AI.
- **Then:** Reads, transaction edits and joint changes succeed; A’s personal plan/goal are protected server-side. One authorized confirmation suffices and the other member is notified.
- **Level:** `end-to-end`.

#### AC-083

- **Given:** There are personal goals of A and B, a joint RUB 600,000 goal and a RUB 10,000 virtual reserve.
- **When:** Both open individual and household views, increase an authorized reserve and link a dedicated account.
- **Then:** The joint goal appears whole in the shared block; personal halves are not created. The reserve reduces household availability once; free funds are not automatically allocated to goals.
- **Level:** `end-to-end`.

#### AC-092

- **Given:** RUB 1,000 is free; both attempt to reserve 800 for authorized goals.
- **When:** Commands execute concurrently.
- **Then:** Checking household availability and reserving are atomic: at most one command succeeds; rejection does not reduce another reserve and both see current availability.
- **Level:** `integration`.

#### AC-086

- **Given:** A and B opened the same transaction or clarification revision.
- **When:** Both submit conflicting edits and replay one request.
- **Then:** One result applies; the other receives a conflict requiring refresh. Replay does not duplicate effects; reversal creates a checked new revision without erasing the other member’s later edit.
- **Level:** `integration`.

### Verification

```sh
make test-web FILTER=goals && make e2e SCENARIO=goals
```

Reservations/mode switching reflect atomic domain results; double allocation is rejected clearly.

The `make` commands are a future contract established by task-1.1; they do not exist yet. Live/paid/manual checks separately record access and actual outcomes. Research does not bypass missing-access blockers.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
