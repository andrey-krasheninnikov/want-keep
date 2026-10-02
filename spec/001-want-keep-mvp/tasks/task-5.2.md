<!-- want-keep-task: task-5.2 -->
# task-5.2 — Проверять каждую операцию через AI-команды / Review every transaction through AI commands

## RU

Автоматизировать классификацию и подтверждённые связи с защитой финансовых инвариантов.

**Состояние:** Не начато; задача ожидает собственные зависимости и entry gates.

**Зависимости:** `task-5.1`, `task-2.4`, `task-2.6`, `task-2.8`, `task-2.9`.

**Тип:** `implementation`.

### Изменение и контракты

Проверять каждую содержательную revision через сохраняемый runtime-контракт transaction_review_v1. Сохранять frozen projection и pseudonymous reference map до provider IO; завершённый ответ и отдельное задание ai_validation фиксировать атомарно. Повторная валидация не повторяет оплачиваемый вызов. Принимать только классификацию по активному каталогу, распределение по сохранённым правилам, предложения связи, уточнение, no_change или отказ. Суммы, даты, комиссии, банковские статусы, actor, household и произвольные инструменты не назначаются моделью. Проверять текущие revisions, права и защиту ручных полей; эффект, решения, аудит и receipt атомарны. Неподтверждённые доли/связи требуют решения участника. Ответ свободным текстом создаёт отдельную job по clarification ID и answer revision; terminal job исходной версии не запускается повторно. Бюджеты/цели требуют явного подтверждения; до подключения владельцев их применение возвращает feature_unavailable. Сохраняются gpt-5.6-terra xhigh, общий $50 и retry/fencing task-5.1. Новый prompt/schema проверяются offline, production_admitted=false до отдельной live-квалификации; исходный eval 206/206 не подтверждает новый контракт.

### Границы изменений

- `backend/internal/ai/`
- `backend/internal/delivery/review/`
- `backend/internal/storage/`
- `backend/migrations/023_transaction_review_commands.sql`
- `api/schemas/chat.yaml`
- `backend/test/integration/ai-commands/`
- `backend/test/ai/transactions/`

### Экранный контракт

### SCR-025 — Уточнения

`/chat/clarifications`

**Вопрос:** Что нужно уточнить для правильного учёта?

**Главный ответ:** Конкретный вопрос с безопасными вариантами и контекстом.

**Структура сверху вниз:** Ожидают ответа → причина/покупка → варианты и свободный ответ → preview → статус.

**Следующее действие:** Ответить FORM-12, открыть оригинал SCR-010/011; уже отвеченное показать с автором.

**Объяснение и детализация:** Ответ привязан к revision вопроса и операции. Вариант применяется транзакционно; свободный текст сохраняется с автором и отдельной answer revision и ожидает новую проверку. Конфликт сохраняет ввод; личные бюджет/цели изменяет только владелец.

**Права:** Оба видят; личное изменяет только владелец, совместное — любой участник.

Forms: FORM-12.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-04, UISTATE-14.

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

- **REQ-008:** Повторные импорты, чек и запись чата объединяют доказательства одной операции без повторного учёта.
- **REQ-012:** Исправление учёта сохраняет оригинал, автора, основание, версию и возможность отмены решения.
- **REQ-017:** Чат создаёт установленную операцию, уточняет недостающие данные и явно объясняет пропуск неподходящего документа.
- **REQ-018:** Каждая новая или содержательно изменённая операция получает AI-проверку своей версии.
- **REQ-019:** AI автоматизирует внутренний учёт через проверяемые команды; неопределённость остаётся явной.
- **REQ-021:** AI меняет утверждённый бюджет, прогноз доходов или цели только по явному решению участника с правом на изменение.
- **REQ-022:** Полные операции и чеки могут передаваться OpenAI, секреты доступа и лишние закрытые данные исключаются.
- **REQ-051:** AI ограничен бюджетом $50/месяц и деградирует в очередь ожидания без остановки обычного учёта.
- **REQ-060:** Текст чеков, банковских описаний и ответов AI не может расширять полномочия агента.
- **REQ-061:** Повторные задания, перезапуски и параллельные изменения не создают двойных финансовых эффектов.
- **REQ-064:** Оба участника видят все финансовые данные и изменяют операции; личные цели и части плана изменяет только их владелец.
- **REQ-067:** Расходы и позиции чеков имеют личное или совместное назначение; общая доля по умолчанию 50/50 с исключениями статьи или покупки.
- **REQ-071:** Один общий чат сохраняет автора сообщения и проверяет полномочия инициатора AI-команды при исполнении.
- **REQ-072:** Конкурирующие изменения, ответы на уточнения и отмены проверяют версию и текущие права, сохраняя обоих авторов.
- **REQ-076:** Семейная область проверяется для API, файлов, AI, фоновых задач и внешних ID независимо от присланных actor/owner.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-012

- **Дано:** AI ошибочно связал две операции; исходные импортированные записи сохранены.
- **Когда:** Владелец отменяет связь и исправляет категорию.
- **Тогда:** Пересчитаны производные отчёты; видна история; повторный импорт не стирает правку владельца.
- **Уровень:** `integration`.

#### AC-018

- **Дано:** Есть импортированная, ручная и созданная чатом операции.
- **Когда:** Они создаются, а затем одна финансовая запись исправляется.
- **Тогда:** Каждая актуальная версия поставлена на проверку; повторная доставка задачи не дублирует эффект; устаревший ответ AI не меняет новую версию.
- **Уровень:** `integration`.

#### AC-019

- **Дано:** AI предлагает сумму, противоречащую источнику, и связь с несколькими кандидатами.
- **Когда:** Приложение проверяет предложения.
- **Тогда:** Противоречивое изменение отклонено, неоднозначность поступает в очередь уточнений; категории и подтверждённые связи могут применяться автоматически.
- **Уровень:** `integration`.

#### AC-021

- **Дано:** Есть утверждённый бюджет и предложение перераспределения.
- **Когда:** Приходит новый расход, затем уполномоченный участник подтверждает предложенное изменение.
- **Тогда:** До подтверждения план неизменен; подтверждение применяет показанную версию предложения один раз; устаревшее предложение пересогласуется.
- **Уровень:** `integration`.

#### AC-022

- **Дано:** Операция и чек доступны владельцу; рядом в системе хранятся ключи источника.
- **Когда:** Формируется запрос AI и диагностическая запись.
- **Тогда:** В запросе только разрешённые данные операции/документа; ключи и сессии отсутствуют в запросе и логах; политика хранения OpenAI раскрыта.
- **Уровень:** `contract`.

#### AC-060

- **Дано:** В PDF или описании операции есть инструкция раскрыть ключ либо сделать перевод.
- **Когда:** Документ обрабатывается AI.
- **Тогда:** Инструкция считается данными; секреты и платёжные инструменты недоступны; недопустимая команда отклонена и не меняет учёт.
- **Уровень:** `integration`.

#### AC-064

- **Дано:** На одном счёте две покупки RUB 300 близко по времени; владелец исправил одну.
- **Когда:** AI получает чек без уникального идентификатора и старый результат классификации.
- **Тогда:** Запрашивается выбор совпадения; покупки не сливаются по вероятности; правка человека не теряется.
- **Уровень:** `integration`.

#### AC-069

- **Дано:** Три ответа AI: отказ, JSON вне схемы и корректный ответ на уже изменённую версию.
- **Когда:** Одновременно исчерпывается резерв месячного бюджета.
- **Тогда:** Ни один ответ не даёт неверного финансового эффекта; статусы различимы, платные повторы ограничены; данные и подтверждённые операции не теряются.
- **Уровень:** `integration`.

#### AC-078

- **Дано:** У A есть личная цель и статья плана; у семьи общая статья и операции обоих.
- **Когда:** B читает все данные, исправляет операцию A и общий план, затем пытается изменить личную цель/план A через API и AI.
- **Тогда:** Чтение, операции и общее изменение разрешены; личные план/цель A защищены сервером. Одного уполномоченного подтверждения достаточно, второй уведомлён.
- **Уровень:** `end-to-end`.

#### AC-081

- **Дано:** Чек RUB 1000 содержит общие продукты 600 и личные покупки A 100 и B 300.
- **Когда:** Чек заносит любой участник; AI применяет правила или уточняет неизвестное назначение.
- **Тогда:** Факт семьи 1000, A 400, B 600; доли суммируются точно. Исключение покупки приоритетнее статьи, затем 50/50; неоднозначная трата сохранена без вымышленной принадлежности.
- **Уровень:** `integration`.

#### AC-085

- **Дано:** Оба видят общий чат; A имеет личную цель, B просит AI изменить её от имени A.
- **Когда:** Модель предлагает действие, а затем A подтверждает новую адресованную ему версию предложения.
- **Тогда:** Сообщение B не выдаёт полномочия A; до разрешённого подтверждения изменения нет. Аудит хранит автора сообщения, подтвердившего и AI-основание; секретов в чате нет.
- **Уровень:** `integration`.

#### AC-086

- **Дано:** A и B открыли одну версию операции или уточнения.
- **Когда:** Оба отправляют несовместимые изменения и повторяют один запрос.
- **Тогда:** Один результат применяется; второй получает конфликт с необходимостью перечитать состояние. Повтор не дублирует эффект; отмена создаёт новую проверенную revision и не стирает чужую последующую правку.
- **Уровень:** `integration`.

#### AC-090

- **Дано:** В тестах созданы две изолированные семьи; запрос или задача подменяет householdId/actor/resourceId.
- **Когда:** Проверяются чтение файла, импорт, исправление, поиск AI и дедупликация.
- **Тогда:** Чужие объекты недоступны и не объединяются; сервер берёт principal из сессии или проверенного контекста задания. Отказ не раскрывает чужое содержимое.
- **Уровень:** `integration`.

### Проверка результата

```sh
make test-integration AREA=ai-commands && make test-ai-commands-race && make eval-ai SUITE=transactions
```

Каждая версия отслеживается; invalid/refusal/stale/injection отклоняются; автоматические эффекты сохраняют инварианты и аудит.

Backend/API task-5.2 и offline regression реализованы; доказательства и ограничения: evidence/task-5.2-transaction-review.md. Реальный OpenAI, чат, извлечение чеков, экраны, бюджет/цели и production остаются соответствующим задачам. SDD Ready for development не означает эксплуатационную готовность.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Automate classification and substantiated links while protecting financial invariants.

**Status:** Not started; the task awaits its own dependencies and entry gates.

**Dependencies:** `task-5.1`, `task-2.4`, `task-2.6`, `task-2.8`, `task-2.9`.

**Kind:** `implementation`.

### Change and contracts

Review each material revision using the persisted transaction_review_v1 runtime contract. Freeze the projection and pseudonymous reference map before provider IO; commit the completed outcome and a separate ai_validation job atomically. Validation retries never repeat a paid call. Accept active-catalog classification, saved-rule distribution, link proposals, clarification, no_change or rejection only. Models cannot assign amounts, dates, fees, bank statuses, actor, household or arbitrary tools. Recheck current revisions, authority and human field protections; effects, decisions, audit and receipts are atomic. Unconfirmed shares/links require a member decision. A free-text answer creates a separate job bound to clarification ID and answer revision; the original terminal job is not restarted. Budgets/goals require explicit approval; applying them returns feature_unavailable until their owning handlers are connected. Preserve gpt-5.6-terra xhigh, the shared USD 50 cap and task-5.1 retry/fencing. Verify the new prompt/schema offline; production_admitted=false pending separate live qualification. The original 206/206 evaluation does not qualify the new contract.

### Change boundaries

- `backend/internal/ai/`
- `backend/internal/delivery/review/`
- `backend/internal/storage/`
- `backend/migrations/023_transaction_review_commands.sql`
- `api/schemas/chat.yaml`
- `backend/test/integration/ai-commands/`
- `backend/test/ai/transactions/`

### Screen contract

### SCR-025 — Clarifications

`/chat/clarifications`

**Question:** What needs clarification for correct accounting?

**Primary answer:** A specific question with safe options and context.

**Top-down structure:** Awaiting answer → reason/purchase → choices and free text → preview → status.

**Next action:** Answer FORM-12, open original SCR-010/011; show answered items with author.

**Explanation and details:** Answers bind to question and transaction revisions. A choice applies transactionally; free text preserves its author and distinct answer revision and awaits a new check. Conflicts preserve input; only the owner changes personal budgets/goals.

**Permissions:** Both read; only the owner edits personal resources, either member edits shared resources.

Forms: FORM-12.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-04, UISTATE-14.

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

- **REQ-008:** Repeated imports, receipts and chat entries combine evidence of one transaction without double counting.
- **REQ-012:** Accounting corrections preserve the original, actor, reason, version and ability to undo a decision.
- **REQ-017:** Chat records an established transaction, clarifies missing data and explicitly explains skipped irrelevant documents.
- **REQ-018:** Every new or materially changed transaction receives AI review of its version.
- **REQ-019:** AI automates internal accounting through validated commands; uncertainty remains explicit.
- **REQ-021:** AI changes an approved budget, income forecast or goals only on an explicit decision by a member authorized for the change.
- **REQ-022:** Full transactions and receipts may be sent to OpenAI; access secrets and unrelated private data are excluded.
- **REQ-051:** AI is limited to $50/month and degrades to a waiting queue without stopping ordinary accounting.
- **REQ-060:** Receipt text, bank descriptions and AI outputs cannot expand agent authority.
- **REQ-061:** Repeated jobs, restarts and concurrent changes cannot create duplicate financial effects.
- **REQ-064:** Both members see all financial data and edit transactions; only the owner edits personal goals and plan portions.
- **REQ-067:** Expenses and receipt items have personal or joint attribution; joint shares default to 50/50 with line or purchase overrides.
- **REQ-071:** One shared chat retains message authors and checks the AI command initiator’s authority at execution.
- **REQ-072:** Competing edits, clarification answers and reversals check revision and current permissions while retaining both authors.
- **REQ-076:** Household scope is checked for APIs, files, AI, jobs and external IDs independently of supplied actor/owner fields.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-012

- **Given:** AI incorrectly linked two transactions; original imports are retained.
- **When:** The owner unlinks them and corrects the category.
- **Then:** Derived reports are recalculated, history is visible and reimport does not overwrite the owner's correction.
- **Level:** `integration`.

#### AC-018

- **Given:** Imported, manual and chat-created transactions exist.
- **When:** They are created and one financial record is then corrected.
- **Then:** Every current version is queued for review; repeated job delivery does not duplicate effects; a stale AI response cannot alter a newer version.
- **Level:** `integration`.

#### AC-019

- **Given:** AI proposes a source-conflicting amount and a match with several candidates.
- **When:** The application validates the proposals.
- **Then:** The conflicting change is rejected and ambiguity enters the clarification queue; categories and substantiated links may be applied automatically.
- **Level:** `integration`.

#### AC-021

- **Given:** An approved budget and a reallocation proposal exist.
- **When:** A new expense arrives and an authorized member later confirms the proposal.
- **Then:** The plan stays unchanged until confirmation; confirmation applies the displayed proposal version once; a stale proposal must be reconfirmed.
- **Level:** `integration`.

#### AC-022

- **Given:** A transaction and receipt are available to the owner; source credentials are stored elsewhere.
- **When:** An AI request and diagnostic record are produced.
- **Then:** The request contains only permitted transaction/document data; keys and sessions appear in neither request nor logs; OpenAI retention policy is disclosed.
- **Level:** `contract`.

#### AC-060

- **Given:** A PDF or transaction description instructs the agent to reveal a key or transfer funds.
- **When:** AI processes the document.
- **Then:** The instruction is treated as data; secrets and payment tools are unavailable; an invalid command is rejected without changing accounting.
- **Level:** `integration`.

#### AC-064

- **Given:** One account has two RUB 300 purchases close in time; the owner corrected one.
- **When:** AI receives a receipt without a unique identifier and a stale classification result.
- **Then:** A match selection is requested; purchases are not merged merely by likelihood and the owner's correction survives.
- **Level:** `integration`.

#### AC-069

- **Given:** Three AI outputs are a refusal, off-schema JSON and a valid response to an obsolete version.
- **When:** The monthly spend reservation is exhausted concurrently.
- **Then:** No response produces an incorrect financial effect; states are distinguishable, paid retries are bounded and data/confirmed transactions are retained.
- **Level:** `integration`.

#### AC-078

- **Given:** A has a personal goal and plan line; the household has a joint line and both members’ transactions.
- **When:** B reads all data, edits A’s transaction and the joint plan, then attempts to change A’s personal goal/plan through API and AI.
- **Then:** Reads, transaction edits and joint changes succeed; A’s personal plan/goal are protected server-side. One authorized confirmation suffices and the other member is notified.
- **Level:** `end-to-end`.

#### AC-081

- **Given:** A RUB 1,000 receipt contains joint groceries of 600 and personal purchases of A 100 and B 300.
- **When:** Either member enters the receipt; AI applies rules or clarifies unknown attribution.
- **Then:** Household actual is 1,000, A 400, B 600; shares sum exactly. Purchase override takes precedence over plan line, then 50/50; ambiguous spending persists without invented attribution.
- **Level:** `integration`.

#### AC-085

- **Given:** Both see the shared chat; A has a personal goal and B asks AI to change it as A.
- **When:** The model proposes an action and A later confirms a new proposal version addressed to A.
- **Then:** B’s message grants no authority of A; nothing changes before authorized confirmation. Audit records message author, approver and AI rationale; chat contains no secrets.
- **Level:** `integration`.

#### AC-086

- **Given:** A and B opened the same transaction or clarification revision.
- **When:** Both submit conflicting edits and replay one request.
- **Then:** One result applies; the other receives a conflict requiring refresh. Replay does not duplicate effects; reversal creates a checked new revision without erasing the other member’s later edit.
- **Level:** `integration`.

#### AC-090

- **Given:** Tests contain two isolated households; a request or job forges householdId/actor/resourceId.
- **When:** File reads, import, correction, AI retrieval and deduplication are exercised.
- **Then:** Foreign objects are inaccessible and never merged; the server takes principal from the session or validated job context. Denial reveals no foreign content.
- **Level:** `integration`.

### Verification

```sh
make test-integration AREA=ai-commands && make test-ai-commands-race && make eval-ai SUITE=transactions
```

Every version is tracked; invalid/refusal/stale/injection outputs are rejected; automatic effects preserve invariants and audit.

Backend/API task-5.2 and offline regressions are implemented; evidence and limitations: evidence/task-5.2-transaction-review.en.md. Live OpenAI, chat, receipt extraction, screens, budget/goal handlers and production remain with their owning tasks. SDD Ready for development is not operational readiness.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
