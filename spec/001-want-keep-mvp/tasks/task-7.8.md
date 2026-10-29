<!-- want-keep-task: task-7.8 -->
# task-7.8 — Добавить уведомления и web-push / Add notifications and web push

## RU

Доставлять напоминания и сводки с контролем приватности.

**Состояние:** Открыта; ready-for-agent. Описание готово к реализации; начало зависит от незакрытых задач и условий ниже. Метка не подтверждает выполнение, live-допуск или разрешение на production; SDD Ready не является блокером.

**Зависимости:** `task-7.1`, `task-6.8`, `task-5.5`, `task-1.4`, `task-3.1`, `task-7.9`.

**Тип:** `implementation`.

### Изменение и контракты

### Текущее поведение

Desktop-оболочка, отдельный вход RU/EN, дизайн-система и backend семейных прав включены в базу. Их наличие не доказывает готовность всех экранов и сценариев этой карточки.

### Требуемое поведение

Создать in-app центр, дневную сводку, платежные напоминания, sync/AI/backup alerts и Web Push subscriptions. Дедуплицировать уведомления по событию/периоду/устройству; показывать только разрешённые payload без финансовых деталей по умолчанию. Обработать revoke/expired endpoints и проверять авторизацию после deep link. Проверить разрешение, запрет, отзыв и реальную доставку в Chrome и Arc на macOS; недоставка push не считается прочтением.

### Ключевые контракты

Outbox событий, персональное прочтение, push subscriptions и session revocation; безопасные deep links и настройки уведомлений.

### Вне объёма

Только desktop web на ноутбуке с macOS; мобильные экраны не добавлять. Финансовые расчёты и проверка полномочий остаются на сервере. Не расширять согласованный продуктовый объём.

### Блокирующие зависимости

- [#48](https://github.com/andrey-krasheninnikov/want-keep/issues/48) (`task-7.1`): выполнена.
- [#47](https://github.com/andrey-krasheninnikov/want-keep/issues/47) (`task-6.8`): открыта; блокирует выполнение.
- [#39](https://github.com/andrey-krasheninnikov/want-keep/issues/39) (`task-5.5`): открыта; блокирует выполнение.
- [#14](https://github.com/andrey-krasheninnikov/want-keep/issues/14) (`task-1.4`): выполнена.
- [#26](https://github.com/andrey-krasheninnikov/want-keep/issues/26) (`task-3.1`): выполнена.
- [#56](https://github.com/andrey-krasheninnikov/want-keep/issues/56) (`task-7.9`): открыта; блокирует выполнение.

### Границы изменений

- `backend/internal/notifications/`
- `web/src/features/notifications/`
- `web/public/`

### Экранный контракт

### SCR-030 — Уведомления

`/notifications`

**Вопрос:** Что важно лично для меня сейчас?

**Главный ответ:** Приоритетные события с конкретным действием.

**Структура сверху вниз:** Непрочитанные/все → нехватка/платёж/уточнение/правка/источник → дата/автор → переход.

**Следующее действие:** Открыть объект; отметить прочитанным; настройки push SCR-031.

**Объяснение и детализация:** Прочтение персональное; отказ push не скрывает in-app, sensitive details отсутствуют в push по умолчанию.

**Права:** Оба участника видят; действия проверяет сервер по членству и владельцу ресурса.

Forms: —.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-04.

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
- **UISTATE-16 — Подтверждено:** После подтверждённого сервером результата показать что изменилось, ссылку на объект и доступное исправление; не полагаться на исчезающий toast.


Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-024:** План поддерживает обязательные расходы по датам и повторяемые платежи.
- **REQ-031:** Кредитные карты показывают задолженность, собственные средства, лимит, минимальный платёж и дату по данным источника.
- **REQ-032:** Грейс-период опирается на условия конкретной карты и показывает сумму и срок сохранения льготы.
- **REQ-040:** Каждый источник обновляется раз в час и по запросу с видимым временем успешного обновления.
- **REQ-049:** Каждый участник входит со своими passkey и одноразовыми кодами восстановления; сброс чужого входа партнёром недоступен.
- **REQ-050:** Файлы, ключи источников, сессии и финансовые журналы защищены от постороннего доступа.
- **REQ-051:** AI ограничен бюджетом $50/месяц и деградирует в очередь ожидания без остановки обычного учёта.
- **REQ-053:** Напоминания и сводки доступны внутри приложения и через разрешённый web-push.
- **REQ-054:** Интерфейс, чат и документация поддерживают RU/EN без изменения финансовой семантики.
- **REQ-058:** Операционные статусы показывают ошибки импорта, AI, курсов, резервирования и расходы без утечки финансового содержимого.
- **REQ-073:** Оба управляют подключениями; банковскую авторизацию выполняет владелец внешнего аккаунта без раскрытия секретов партнёру или AI.
- **REQ-074:** Изменения плана и целей уведомляют второго участника; прочтение и push-подписки принадлежат конкретному пользователю.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-024

- **Дано:** Аренда запланирована на 5-е число, подписка повторяется ежемесячно.
- **Когда:** Наступает дата платежа и импортируется фактическое списание.
- **Тогда:** Плановая строка сама не создаёт расход; сопоставленный факт погашает обязательство без двойного резервирования.
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

#### AC-040

- **Дано:** Два источника доступны, третий требует повторного входа.
- **Когда:** Срабатывает расписание и одновременно нажата кнопка обновления.
- **Тогда:** Нет параллельного дублирования одного задания; доступные источники обновлены, проблемный имеет отдельный статус и старый timestamp.
- **Уровень:** `integration`.

#### AC-051

- **Дано:** OpenAI недоступен либо израсходован разрешённый бюджет с резервами текущих запросов.
- **Когда:** Поступают новый импорт, ручной расход и запрос AI.
- **Тогда:** Учёт и расчёты доступны; статус AI ожидает; новые платные запросы не запускаются сверх разрешённого резерва; неизвестная стоимость не освобождается молча.
- **Уровень:** `integration`.

#### AC-053

- **Дано:** Есть обязательный платёж, дневная сводка и ошибка синхронизации.
- **Когда:** Наступает время уведомления; push разрешён, затем отозван.
- **Тогда:** Внутренние уведомления сохраняются; разрешённый push отправляется без дублей; отзыв push не отключает внутренний канал; детали денег по умолчанию не раскрываются на экране блокировки.
- **Уровень:** `end-to-end+manual`.

#### AC-054

- **Дано:** Есть русская и английская версии одной операции, бюджета и ошибки.
- **Когда:** Переключается язык.
- **Тогда:** Суммы, даты, валюты и смысл совпадают; форматирование локализовано, идентификаторы и категории пользователя не переводятся с потерей данных.
- **Уровень:** `end-to-end+static`.

#### AC-058

- **Дано:** Сломан один коннектор, задержан AI и устарела копия.
- **Когда:** Открывается состояние системы и читаются диагностические логи.
- **Тогда:** Видны отдельные проблемы и действия восстановления; логи содержат идентификаторы/коды, а не чеки, ключи или тексты финансовых сообщений.
- **Уровень:** `integration`.

#### AC-072

- **Дано:** Активны две сессии и push-подписка.
- **Когда:** Владелец восстанавливает доступ и отзывает старое устройство.
- **Тогда:** Старые сессии/привязанные подписки отозваны; ссылка из push требует действующей авторизации; финансовых деталей в push по умолчанию нет.
- **Уровень:** `end-to-end+manual`.

#### AC-087

- **Дано:** A владеет внешним аккаунтом, B инициирует повторную авторизацию или отключение.
- **Когда:** Запрашивается MFA; одновременно завершает работу старое задание синхронизации.
- **Тогда:** MFA адресован A; B видит статус, но не пароль/код/сессию. Отключение отзывает lease/version и запрещает применение старого результата; реальные платежи недоступны обоим.
- **Уровень:** `integration`.

#### AC-088

- **Дано:** Оба имеют уведомления и устройства, A изменяет разрешённую цель.
- **Когда:** B читает уведомление; A восстанавливает вход и отзывает своё устройство.
- **Тогда:** Событие доставлено B один раз; read-state и отзыв устройства одного не меняют подписки другого. Уточнения личного плана направлены его владельцу, остальные доступны обоим.
- **Уровень:** `end-to-end`.

### Проверка результата

```sh
make test-integration AREA=notifications && make e2e SCENARIO=notifications
```

Повторы, revoke и timezone-boundaries проверены; отдельная manual device-проверка подтверждает получение push, не только ответ SDK.

Команды Make являются интерфейсами проверки; наличие target не доказывает реализацию соответствующей suite. Если требуемой suite ещё нет, реализовать её в рамках задачи; отсутствие suite не считается успешной проверкой. Фиксировать фактический результат каждого прогона. make docs-check проверяет документацию и исследовательские инструменты, но не финансовый runtime, браузеры или production. Live/manual-проверки, доступ и расходы фиксируются отдельно.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Deliver reminders and summaries with privacy control.

**Status:** Open; ready-for-agent. The description is ready for implementation; open prerequisites and the entry conditions below govern execution. The label does not establish completion, live admission or production authorization; SDD Ready is not a blocker.

**Dependencies:** `task-7.1`, `task-6.8`, `task-5.5`, `task-1.4`, `task-3.1`, `task-7.9`.

**Kind:** `implementation`.

### Change and contracts

### Current behavior

The desktop shell, separate RU/EN sign-in, design system and household-permission backend are included in the base. Their presence does not establish readiness of every screen and scenario in this task.

### Desired behavior

Create an in-app center, daily summary, payment reminders, sync/AI/backup alerts and Web Push subscriptions. Deduplicate by event/period/device and use authorized payloads without financial details by default. Handle revocation/expired endpoints and authorize deep-link access. Verify permission, denial, revocation and actual delivery in Chrome and Arc on macOS; undelivered push is not read.

### Key interfaces

Event outbox, personal read state, push subscriptions and session revocation; safe deep links and notification preferences.

### Out of scope

Desktop web on a macOS laptop only; do not add mobile screens. Financial calculations and authority checks remain server-side. Do not expand the agreed product scope.

### Blocked by

- [#48](https://github.com/andrey-krasheninnikov/want-keep/issues/48) (`task-7.1`): completed.
- [#47](https://github.com/andrey-krasheninnikov/want-keep/issues/47) (`task-6.8`): open; blocks execution.
- [#39](https://github.com/andrey-krasheninnikov/want-keep/issues/39) (`task-5.5`): open; blocks execution.
- [#14](https://github.com/andrey-krasheninnikov/want-keep/issues/14) (`task-1.4`): completed.
- [#26](https://github.com/andrey-krasheninnikov/want-keep/issues/26) (`task-3.1`): completed.
- [#56](https://github.com/andrey-krasheninnikov/want-keep/issues/56) (`task-7.9`): open; blocks execution.

### Change boundaries

- `backend/internal/notifications/`
- `web/src/features/notifications/`
- `web/public/`

### Screen contract

### SCR-030 — Notifications

`/notifications`

**Question:** What matters to me right now?

**Primary answer:** Prioritized events with a concrete action.

**Top-down structure:** Unread/all → shortfall/payment/clarification/change/source → date/author → destination.

**Next action:** Open object; mark read; push settings SCR-031.

**Explanation and details:** Reading is personal; push denial never hides in-app and push omits sensitive details by default.

**Permissions:** Both members can read; server checks membership and resource ownership for actions.

Forms: —.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-04.

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
- **UISTATE-16 — Confirmed:** After server-confirmed outcome show what changed, an object link and available correction; do not rely on a disappearing toast.


Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-024:** The plan supports dated obligations and recurring payments.
- **REQ-031:** Credit cards show debt, own funds, credit limit, minimum payment and due date from source data.
- **REQ-032:** Grace-period tracking uses the specific card's terms and shows the amount and deadline needed to preserve the benefit.
- **REQ-040:** Each source refreshes hourly and on demand with a visible last-success timestamp.
- **REQ-049:** Each member signs in with their own passkeys and one-time recovery codes; partner-assisted reset is unavailable.
- **REQ-050:** Files, source keys, sessions and financial records are protected against unauthorized access.
- **REQ-051:** AI is limited to $50/month and degrades to a waiting queue without stopping ordinary accounting.
- **REQ-053:** Reminders and summaries are available in-app and through authorized web push.
- **REQ-054:** UI, chat and documentation support RU/EN without changing financial semantics.
- **REQ-058:** Operational status exposes import, AI, FX, backup failures and spend without leaking financial content.
- **REQ-073:** Both manage connections; the external-account owner performs bank authentication without exposing secrets to the partner or AI.
- **REQ-074:** Plan and goal changes notify the other member; read state and push subscriptions belong to the individual user.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-024

- **Given:** Rent is planned for the 5th and a subscription recurs monthly.
- **When:** The due date arrives and the actual charge is imported.
- **Then:** The planned row does not itself create an expense; matched actual payment settles the obligation without double reservation.
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

#### AC-040

- **Given:** Two sources are available and a third requires sign-in again.
- **When:** The schedule fires while the refresh button is pressed.
- **Then:** The same job is not duplicated concurrently; available sources refresh and the failing source has its own status and old timestamp.
- **Level:** `integration`.

#### AC-051

- **Given:** OpenAI is unavailable or the allowed budget including in-flight reservations is exhausted.
- **When:** A new import, manual expense and AI request arrive.
- **Then:** Accounting and calculations remain available; AI status is waiting; no new paid calls exceed the allowed reservation; unknown cost is not silently released.
- **Level:** `integration`.

#### AC-053

- **Given:** There is an obligation, a daily summary and a sync error.
- **When:** Notification time arrives; push permission is enabled and later revoked.
- **Then:** In-app notifications remain; allowed push is sent without duplicates; revocation does not disable in-app delivery; financial details are hidden on the lock screen by default.
- **Level:** `end-to-end+manual`.

#### AC-054

- **Given:** Russian and English versions of the same transaction, budget and error exist.
- **When:** The language is switched.
- **Then:** Amounts, dates, currencies and meaning agree; formatting is localized while IDs and owner categories are not destructively translated.
- **Level:** `end-to-end+static`.

#### AC-058

- **Given:** A connector fails, AI is delayed and a backup is stale.
- **When:** System health and diagnostic logs are inspected.
- **Then:** Separate failures and recovery actions are visible; logs contain identifiers/codes, not receipts, keys or financial message text.
- **Level:** `integration`.

#### AC-072

- **Given:** Two sessions and a push subscription are active.
- **When:** The owner recovers access and revokes an old device.
- **Then:** Old sessions/associated subscriptions are revoked; push links require current authorization; push contains no financial details by default.
- **Level:** `end-to-end+manual`.

#### AC-087

- **Given:** A owns the external account and B initiates reauthorization or disconnect.
- **When:** MFA is requested while an old sync job completes.
- **Then:** MFA is addressed to A; B sees status but no password/code/session. Disconnect revokes lease/version and prevents stale-result application; actual payments are unavailable to both.
- **Level:** `integration`.

#### AC-088

- **Given:** Both have notifications and devices and A changes an authorized goal.
- **When:** B reads the notification; A recovers sign-in and revokes their device.
- **Then:** B receives the event once; one member’s read state and device revocation do not alter the other’s subscriptions. Personal-plan clarifications target its owner; other clarifications are open to both.
- **Level:** `end-to-end`.

### Verification

```sh
make test-integration AREA=notifications && make e2e SCENARIO=notifications
```

Replay, revocation and timezone boundaries pass; separate manual device evidence confirms push receipt, not just an SDK response.

Make commands are verification interfaces; the existence of a target does not establish an implemented suite. If a required suite is missing, implement it as part of the task; a missing suite is not a passing check. Record the actual outcome of each run. make docs-check validates documentation and research tooling, not financial runtime, browsers or production. Record live/manual checks, access and costs separately.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
