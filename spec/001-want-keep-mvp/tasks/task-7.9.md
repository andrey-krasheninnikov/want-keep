<!-- want-keep-task: task-7.9 -->
# task-7.9 — Добавить семейный контекст и принадлежность в интерфейс / Add household context and ownership to the interface

## RU

Добавить семейный контекст и принадлежность в интерфейс.

**Состояние:** Открыта; ready-for-agent. Описание готово к реализации; начало зависит от незакрытых задач и условий ниже. Метка не подтверждает выполнение, live-допуск или разрешение на production; SDD Ready не является блокером.

**Зависимости:** `task-7.1`, `task-1.6`.

**Тип:** `implementation`.

### Изменение и контракты

### Текущее поведение

Desktop-оболочка, отдельный вход RU/EN, дизайн-система и backend семейных прав включены в базу. Их наличие не доказывает готовность всех экранов и сценариев этой карточки.

### Требуемое поведение

Показать текущего участника, отдельный вход/приглашение, принадлежность личное/семейное и переключение семейного/индивидуального вида. Персональный фильтр меняет представление, не права и не principal. Рендерить права сервера; запрещённые действия защищены backend. Одно семейное пространство и общий чат; состояние уведомлений персональное.

### Ключевые контракты

Me/Household, приглашение и passkey, Ownership, семейный/индивидуальный фильтр; текущий principal не меняется вместе с фильтром.

### Вне объёма

Только desktop web на ноутбуке с macOS; мобильные экраны не добавлять. Финансовые расчёты и проверка полномочий остаются на сервере. Не расширять согласованный продуктовый объём.

### Блокирующие зависимости

- [#48](https://github.com/andrey-krasheninnikov/want-keep/issues/48) (`task-7.1`): выполнена.
- [#16](https://github.com/andrey-krasheninnikov/want-keep/issues/16) (`task-1.6`): выполнена.

### Границы изменений

- `web/src/features/household/`
- `web/src/features/auth/`
- `web/src/app/`

### Экранный контракт

### SCR-032 — Семья

`/settings/household`

**Вопрос:** Кто в семье и что каждый может?

**Главный ответ:** Два отдельных участника с прозрачными правами.

**Структура сверху вниз:** Состав/приглашение → описание общего доступа → личные/общие ресурсы и правила.

**Следующее действие:** Если есть место, выдать/перевыпустить приглашение после собственного passkey-подтверждения до 5 минут; отозвать действующей сессией. Сначала показать метаданные/revision; секрет повторно не читается.

**Объяснение и детализация:** Нет смены ролей, выхода/замены участника и восстановления партнёром в MVP.

**Права:** Оба участника видят; действия проверяет сервер по членству и владельцу ресурса.

Forms: FORM-02.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16.

#### FORM-02 — Начало семьи и приглашение

**Поля:** Имя участника/семьи, язык, таймзона/валюта; закрытое приглашение, имя второго участника и passkey.

**Проверки и права:** Операторский bootstrap однократен. Приглашение одно, случайное, на 24 часа; выдача/перевыпуск требуют собственной auth до 5 минут и expectedRevision. Отзыв с CSRF не требует свежей auth. Принятие проверяет browser/purpose/revision, inviter membership и лимит атомарно с новым passkey/сессией/кодами.

**Результат:** Создано членство, показаны личные recovery-коды; далее onboarding. Секреты не в URL журналов/аналитики.

- **UISTATE-01 — Загрузка:** Скелетон структуры и подпись загрузки; суммы не подменяются нулями.
- **UISTATE-02 — Обновление:** Сохранить предыдущие данные и контекст, показать время последнего успеха; блокировать только конфликтующие действия.
- **UISTATE-03 — Пусто:** Объяснить полезный результат и предложить первое действие: счёт, чек, план или цель.
- **UISTATE-05 — Частичные данные:** Назвать отсутствующий источник/период и последствия для суммы; доступные блоки работают; неизвестное обозначить отдельно.
- **UISTATE-06 — Устаревшие данные:** Показать дату последнего успеха и влияние на решение; дать обновить или перейти к подключению.
- **UISTATE-07 — Ошибка:** Понятная причина и следующий шаг у проблемного блока; ввод и исправные данные сохранить, диагностику раскрывать отдельно.
- **UISTATE-08 — Offline:** Показать отсутствие связи; не обещать сохранение. Чувствительные черновики только в памяти текущей вкладки, без новой offline-очереди.
- **UISTATE-09 — Сохранение:** Немедленно показать прогресс текущего действия и не допускать дублирующую отправку команды.
- **UISTATE-10 — Исход неизвестен:** Сохранить ID команды/ввод, запросить её результат; не создавать новую финансовую команду вслепую. После перезагрузки сверять серверный список недавних команд.
- **UISTATE-11 — Конфликт версии:** Показать авторов и различия, сохранить мой ввод; загрузить актуальную версию и дать повторно применить выбранные изменения после проверки.
- **UISTATE-12 — Недостаточно прав:** Финансовые данные доступны семье; запрещённое изменение объясняет владельца. Сервер отклоняет команду независимо от видимости кнопки.
- **UISTATE-13 — Сессия истекла:** Закрыть защищённое содержимое; вход для того же участника, безопасный возврат по внутреннему маршруту. Чужой вход не получает прежний черновик.
- **UISTATE-16 — Подтверждено:** После подтверждённого сервером результата показать что изменилось, ссылку на объект и доступное исправление; не полагаться на исчезающий toast.


Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-001:** Семейный пилот обслуживает двух участников с раздельным входом и закрытым присоединением; публичной регистрации нет.
- **REQ-063:** Пользователь, семья и членство моделируются отдельно; ограничение двух участников задаётся конфигурацией.
- **REQ-064:** Оба участника видят все финансовые данные и изменяют операции; личные цели и части плана изменяет только их владелец.
- **REQ-065:** Принадлежность счёта, владелец внешнего аккаунта, автор записи и принадлежность расхода являются отдельными признаками.
- **REQ-071:** Один общий чат сохраняет автора сообщения и проверяет полномочия инициатора AI-команды при исполнении.
- **REQ-074:** Изменения плана и целей уведомляют второго участника; прочтение и push-подписки принадлежат конкретному пользователю.
- **REQ-076:** Семейная область проверяется для API, файлов, AI, фоновых задач и внешних ID независимо от присланных actor/owner.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-077

- **Дано:** Два пользователя состоят в одной семье.
- **Когда:** Проверяются схема, авторизация и ограничение членства.
- **Тогда:** Нет полей partner1/partner2 и ветвлений по конкретным пользователям; роли и принадлежность отделены от личности. Выход, замена и новые роли не реализованы.
- **Уровень:** `integration`.

#### AC-078

- **Дано:** У A есть личная цель и статья плана; у семьи общая статья и операции обоих.
- **Когда:** B читает все данные, исправляет операцию A и общий план, затем пытается изменить личную цель/план A через API и AI.
- **Тогда:** Чтение, операции и общее изменение разрешены; личные план/цель A защищены сервером. Одного уполномоченного подтверждения достаточно, второй уведомлён.
- **Уровень:** `end-to-end`.

#### AC-079

- **Дано:** A и B имеют разные аккаунты одного провайдера и общий счёт; B заносит покупку A со счёта B.
- **Когда:** Выполняются ввод, импорт обоих аккаунтов и повторное подключение того же внешнего аккаунта.
- **Тогда:** Разные аккаунты не сливаются; повторный источник не удваивает остатки. Плательщик, автор и получатель расхода сохраняются независимо. Неустановленное совпадение блокирует новый учёт до уточнения.
- **Уровень:** `integration`.

#### AC-085

- **Дано:** Оба видят общий чат; A имеет личную цель, B просит AI изменить её от имени A.
- **Когда:** Модель предлагает действие, а затем A подтверждает новую адресованную ему версию предложения.
- **Тогда:** Сообщение B не выдаёт полномочия A; до разрешённого подтверждения изменения нет. Аудит хранит автора сообщения, подтвердившего и AI-основание; секретов в чате нет.
- **Уровень:** `integration`.

#### AC-088

- **Дано:** Оба имеют уведомления и устройства, A изменяет разрешённую цель.
- **Когда:** B читает уведомление; A восстанавливает вход и отзывает своё устройство.
- **Тогда:** Событие доставлено B один раз; read-state и отзыв устройства одного не меняют подписки другого. Уточнения личного плана направлены его владельцу, остальные доступны обоим.
- **Уровень:** `end-to-end`.

#### AC-090

- **Дано:** В тестах созданы две изолированные семьи; запрос или задача подменяет householdId/actor/resourceId.
- **Когда:** Проверяются чтение файла, импорт, исправление, поиск AI и дедупликация.
- **Тогда:** Чужие объекты недоступны и не объединяются; сервер берёт principal из сессии или проверенного контекста задания. Отказ не раскрывает чужое содержимое.
- **Уровень:** `integration`.

#### AC-001

- **Дано:** Создана семья, первый участник вошёл, лимит активных участников равен 2.
- **Когда:** Второй участник принимает приглашение; посторонний пробует открытый вход, повтор приглашения и присоединение сверх лимита.
- **Тогда:** Приглашение создаёт отдельное членство один раз; посторонний не получает данных, повтор и превышение лимита отклонены.
- **Уровень:** `end-to-end`.

### Проверка результата

```sh
make test-web FILTER=household
make e2e SCENARIO=family-access
```

RU/EN и desktop-вид различают текущего автора и выбранный разрез; попытка изменить роль через фильтр не даёт доступа; приглашение не допускает постороннего.

Команды Make являются интерфейсами проверки; наличие target не доказывает реализацию соответствующей suite. Если требуемой suite ещё нет, реализовать её в рамках задачи; отсутствие suite не считается успешной проверкой. Фиксировать фактический результат каждого прогона. make docs-check проверяет документацию и исследовательские инструменты, но не финансовый runtime, браузеры или production. Live/manual-проверки, доступ и расходы фиксируются отдельно.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Add household context and ownership to the interface.

**Status:** Open; ready-for-agent. The description is ready for implementation; open prerequisites and the entry conditions below govern execution. The label does not establish completion, live admission or production authorization; SDD Ready is not a blocker.

**Dependencies:** `task-7.1`, `task-1.6`.

**Kind:** `implementation`.

### Change and contracts

### Current behavior

The desktop shell, separate RU/EN sign-in, design system and household-permission backend are included in the base. Their presence does not establish readiness of every screen and scenario in this task.

### Desired behavior

Show the current member, separate sign-in/invitation, personal/household ownership and household/individual view selection. A person filter changes presentation, never permissions or principal. Render server capabilities; backend protects denied actions. One household workspace and shared chat; notification read state is personal.

### Key interfaces

Me/Household, invitation and passkey, Ownership, household/individual filters; filtering does not change the current principal.

### Out of scope

Desktop web on a macOS laptop only; do not add mobile screens. Financial calculations and authority checks remain server-side. Do not expand the agreed product scope.

### Blocked by

- [#48](https://github.com/andrey-krasheninnikov/want-keep/issues/48) (`task-7.1`): completed.
- [#16](https://github.com/andrey-krasheninnikov/want-keep/issues/16) (`task-1.6`): completed.

### Change boundaries

- `web/src/features/household/`
- `web/src/features/auth/`
- `web/src/app/`

### Screen contract

### SCR-032 — Household

`/settings/household`

**Question:** Who belongs and what can each do?

**Primary answer:** Two distinct members with transparent permissions.

**Top-down structure:** Members/invitation → shared-access explanation → personal/shared resources and rules.

**Next action:** When capacity exists, issue/reissue after own passkey confirmation within 5 minutes; revoke with an active session. Show metadata/revision first; never reread the secret.

**Explanation and details:** No role changing, member exit/replacement or partner recovery in MVP.

**Permissions:** Both members can read; server checks membership and resource ownership for actions.

Forms: FORM-02.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16.

#### FORM-02 — Household setup and invitation

**Fields:** Member/household name, language, timezone/currency; private invitation, joining member name and passkey.

**Validation and permissions:** Operator bootstrap is one-time. One random invitation lasts 24 hours; issue/reissue requires own authentication within 5 minutes and expectedRevision. Revocation uses CSRF without fresh auth. Acceptance checks browser/purpose/revision, inviter membership and capacity atomically with a new passkey/session/codes.

**Outcome:** Membership created, personal recovery codes shown; proceed to onboarding. Secrets excluded from URL logs/analytics.

- **UISTATE-01 — Loading:** Structural skeleton and loading label; amounts are never replaced by zero.
- **UISTATE-02 — Refreshing:** Keep previous data/context and last-success time; block only conflicting actions.
- **UISTATE-03 — Empty:** Explain the useful outcome and offer a first account, receipt, plan or goal action.
- **UISTATE-05 — Partial data:** Name the missing source/period and its effect on the amount; available sections work and unknowns stay explicit.
- **UISTATE-06 — Stale data:** Show last-success date and impact on the decision; offer refresh or connection details.
- **UISTATE-07 — Error:** Plain cause and next step beside the affected section; preserve input/healthy data and expand diagnostics separately.
- **UISTATE-08 — Offline:** Show missing connectivity and do not promise saved data. Sensitive drafts remain only in current-tab memory, without a new offline queue.
- **UISTATE-09 — Saving:** Immediately show current-action progress and prevent duplicate command submission.
- **UISTATE-10 — Unknown outcome:** Keep command ID/input and query its result; never blindly create another financial command. After reload reconcile the server list of recent commands.
- **UISTATE-11 — Version conflict:** Show authors/differences and keep my input; load current version and allow chosen changes to be reapplied after validation.
- **UISTATE-12 — Insufficient permission:** Household can read financial data; forbidden edits explain ownership. Server rejects the command regardless of button visibility.
- **UISTATE-13 — Session expired:** Hide protected contents; require the same member to sign in and return through a safe internal route. Another identity never receives the prior draft.
- **UISTATE-16 — Confirmed:** After server-confirmed outcome show what changed, an object link and available correction; do not rely on a disappearing toast.


Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-001:** The family pilot serves two members with separate sign-in and restricted joining; public registration is unavailable.
- **REQ-063:** User, household and membership are separate models; the two-member limit is configured.
- **REQ-064:** Both members see all financial data and edit transactions; only the owner edits personal goals and plan portions.
- **REQ-065:** Account ownership, external-account owner, record author and expense attribution are distinct dimensions.
- **REQ-071:** One shared chat retains message authors and checks the AI command initiator’s authority at execution.
- **REQ-074:** Plan and goal changes notify the other member; read state and push subscriptions belong to the individual user.
- **REQ-076:** Household scope is checked for APIs, files, AI, jobs and external IDs independently of supplied actor/owner fields.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-077

- **Given:** Two users belong to one household.
- **When:** Schema, authorization and the membership limit are inspected.
- **Then:** There are no partner1/partner2 fields or specific-user branches; roles and ownership are separate from identity. Exit, replacement and new roles are not implemented.
- **Level:** `integration`.

#### AC-078

- **Given:** A has a personal goal and plan line; the household has a joint line and both members’ transactions.
- **When:** B reads all data, edits A’s transaction and the joint plan, then attempts to change A’s personal goal/plan through API and AI.
- **Then:** Reads, transaction edits and joint changes succeed; A’s personal plan/goal are protected server-side. One authorized confirmation suffices and the other member is notified.
- **Level:** `end-to-end`.

#### AC-079

- **Given:** A and B have separate accounts at one provider and a joint account; B enters A’s purchase paid from B’s account.
- **When:** Entry, import of both accounts and reconnection of the same external account run.
- **Then:** Distinct accounts are not merged; a repeated source does not double balances. Payer, author and expense beneficiary remain independent. Unresolved source identity blocks new posting pending clarification.
- **Level:** `integration`.

#### AC-085

- **Given:** Both see the shared chat; A has a personal goal and B asks AI to change it as A.
- **When:** The model proposes an action and A later confirms a new proposal version addressed to A.
- **Then:** B’s message grants no authority of A; nothing changes before authorized confirmation. Audit records message author, approver and AI rationale; chat contains no secrets.
- **Level:** `integration`.

#### AC-088

- **Given:** Both have notifications and devices and A changes an authorized goal.
- **When:** B reads the notification; A recovers sign-in and revokes their device.
- **Then:** B receives the event once; one member’s read state and device revocation do not alter the other’s subscriptions. Personal-plan clarifications target its owner; other clarifications are open to both.
- **Level:** `end-to-end`.

#### AC-090

- **Given:** Tests contain two isolated households; a request or job forges householdId/actor/resourceId.
- **When:** File reads, import, correction, AI retrieval and deduplication are exercised.
- **Then:** Foreign objects are inaccessible and never merged; the server takes principal from the session or validated job context. Denial reveals no foreign content.
- **Level:** `integration`.

#### AC-001

- **Given:** A household exists, the first member is signed in and the active-member limit is 2.
- **When:** The second member accepts an invitation; an outsider attempts open registration, invitation replay and joining beyond the limit.
- **Then:** The invitation creates one separate membership; outsiders receive no data and replay or exceeding the limit is rejected.
- **Level:** `end-to-end`.

### Verification

```sh
make test-web FILTER=household
make e2e SCENARIO=family-access
```

RU/EN and desktop views distinguish current actor from selected view; changing the filter grants no authority and invitations do not admit outsiders.

Make commands are verification interfaces; the existence of a target does not establish an implemented suite. If a required suite is missing, implement it as part of the task; a missing suite is not a passing check. Record the actual outcome of each run. make docs-check validates documentation and research tooling, not financial runtime, browsers or production. Record live/manual checks, access and costs separately.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
