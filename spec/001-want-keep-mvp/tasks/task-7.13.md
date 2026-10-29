<!-- want-keep-task: task-7.13 -->
# task-7.13 — Создать экраны подключений и повторного входа / Create connection and reauthorization screens

## RU

Понятная свежесть данных и безопасное восстановление чтения.

**Состояние:** Открыта; ready-for-agent. Описание готово к реализации; начало зависит от незакрытых задач и условий ниже. Метка не подтверждает выполнение, live-допуск или разрешение на production; SDD Ready не является блокером.

**Зависимости:** `task-7.1`, `task-7.9`, `task-3.3`, `task-4.1`, `task-4.2`, `task-4.3`, `task-4.4`, `task-4.5`, `task-4.6`.

**Тип:** `implementation`.

### Изменение и контракты

### Текущее поведение

Desktop-оболочка, отдельный вход RU/EN, дизайн-система и backend семейных прав включены в базу. Их наличие не доказывает готовность всех экранов и сценариев этой карточки.

### Требуемое поведение

SCR-027–SCR-029: шесть платформ, несколько владельцев/аккаунтов, история с даты и пробелы, последний успех, sync по запросу, отключение с сохранением журнала. Reauth только владельцу внешнего аккаунта; партнёр видит запрос действия без секрета. Повторное подключение использует стабильную identity. Возможности отображать по подтверждённому coverage, неполное покрытие не объявлять готовностью.

### Ключевые контракты

Connection API: freshness/coverage, выбранная дата истории, sync/disconnect и owner-only reauthorization; read-only deploymentGate.

### Вне объёма

Только desktop web на ноутбуке с macOS; мобильные экраны не добавлять. Финансовые расчёты и проверка полномочий остаются на сервере. Не расширять согласованный продуктовый объём.

### Блокирующие зависимости

- [#48](https://github.com/andrey-krasheninnikov/want-keep/issues/48) (`task-7.1`): выполнена.
- [#56](https://github.com/andrey-krasheninnikov/want-keep/issues/56) (`task-7.9`): открыта; блокирует выполнение.
- [#28](https://github.com/andrey-krasheninnikov/want-keep/issues/28) (`task-3.3`): выполнена.
- [#29](https://github.com/andrey-krasheninnikov/want-keep/issues/29) (`task-4.1`): выполнена.
- [#30](https://github.com/andrey-krasheninnikov/want-keep/issues/30) (`task-4.2`): выполнена.
- [#31](https://github.com/andrey-krasheninnikov/want-keep/issues/31) (`task-4.3`): открыта; блокирует выполнение.
- [#32](https://github.com/andrey-krasheninnikov/want-keep/issues/32) (`task-4.4`): открыта; блокирует выполнение.
- [#33](https://github.com/andrey-krasheninnikov/want-keep/issues/33) (`task-4.5`): открыта; блокирует выполнение.
- [#34](https://github.com/andrey-krasheninnikov/want-keep/issues/34) (`task-4.6`): открыта; блокирует выполнение.

### Границы изменений

- `web/src/features/connections/`

### Экранный контракт

### SCR-027 — Подключения

`/connections`

**Вопрос:** Данные обновляются и где нужно моё действие?

**Главный ответ:** Последнее успешное чтение и проблемы каждого подключения.

**Структура сверху вниз:** Требует внимания → аккаунты шести платформ/владельцы → свежесть/покрытие → добавить.

**Следующее действие:** Подключить SCR-029, открыть SCR-028, обновить конкретный источник.

**Объяснение и детализация:** Неподтверждённая возможность не показывается как работающая; одинаковая платформа может иметь разные аккаунты.

**Права:** Оба участника видят; действия проверяет сервер по членству и владельцу ресурса.

Forms: FORM-13.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-04, UISTATE-15.

### SCR-028 — Состояние подключения

`/connections/:id`

**Вопрос:** Почему данные этого сервиса неполны?

**Главный ответ:** Понятная причина, доступный период и нужное действие.

**Структура сверху вниз:** Статус/владелец → последний успех/следующий sync → продукты/границы истории → действия.

**Следующее действие:** Обновить, войти заново SCR-029, отключить с preview сохранения истории.

**Объяснение и детализация:** Ошибки/курсор/diagnostic code раскрываются; не обещать полную историю после частичного импорта.

**Права:** Оба участника видят; действия проверяет сервер по членству и владельцу ресурса.

Forms: FORM-13.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-15.

### SCR-029 — Авторизация платформы

`/connections/new; /connections/:id/reauth`

**Вопрос:** Как безопасно подключить мой аккаунт?

**Главный ответ:** Read-only подключение с вводом доступа его владельцем.

**Структура сверху вниз:** Платформа/владелец → объяснение чтения → изолированный ввод → MFA → результат/дата истории.

**Следующее действие:** Продолжить FORM-13 → SCR-028; партнёру показать ожидание владельца.

**Объяснение и детализация:** Истечение/отмена не теряют существующий журнал; секрет не попадает в чат, логи или общий экран.

**Права:** Оба управляют, только владелец внешнего аккаунта вводит секреты.

Forms: FORM-13.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-15, UISTATE-17.

#### FORM-13 — Подключение и reauth

**Поля:** Платформа, владелец внешнего аккаунта, дата истории, доступные продукты; секрет только в изолированном авторизационном потоке.

**Проверки и права:** Управляют оба, ввод ключа/пароля/MFA только внешним владельцем. Read-only scopes, identity и coverage подтверждены адаптером, без обхода MFA/CAPTCHA.

**Результат:** Подключено/синхронизация/ожидается владелец/ошибка; отключение сохраняет историю и инвалидирует generation.

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
- **UISTATE-15 — Нужен банковский вход:** Назвать подключение и владельца, дать ему безопасно войти; партнёру показать ожидание без доступа к секрету.
- **UISTATE-16 — Подтверждено:** После подтверждённого сервером результата показать что изменилось, ссылку на объект и доступное исправление; не полагаться на исчезающий toast.
- **UISTATE-17 — Отмена:** Объяснить отсутствие нового подтверждённого результата, дать повторить явно; не выдавать отмену системного passkey за поломку.


Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-010:** Возврат уменьшает расходы исходного месяца покупки, сохраняя дату реального поступления денег.
- **REQ-016:** Позиции чека распределяют одну оплаченную сумму по категориям без дублирования итога.
- **REQ-040:** Каждый источник обновляется раз в час и по запросу с видимым временем успешного обновления.
- **REQ-041:** История сохраняет границы покрытия, курсоры, пробелы и статусы источника.
- **REQ-048:** Интеграции и браузерный сборщик выполняют только разрешённые операции чтения.
- **REQ-050:** Файлы, ключи источников, сессии и финансовые журналы защищены от постороннего доступа.
- **REQ-067:** Расходы и позиции чеков имеют личное или совместное назначение; общая доля по умолчанию 50/50 с исключениями статьи или покупки.
- **REQ-072:** Конкурирующие изменения, ответы на уточнения и отмены проверяют версию и текущие права, сохраняя обоих авторов.
- **REQ-073:** Оба управляют подключениями; банковскую авторизацию выполняет владелец внешнего аккаунта без раскрытия секретов партнёру или AI.
- **REQ-076:** Семейная область проверяется для API, файлов, AI, фоновых задач и внешних ID независимо от присланных actor/owner.
- **REQ-082:** Экранные состояния объясняют последствия и безопасный следующий шаг без потери ввода.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-040

- **Дано:** Два источника доступны, третий требует повторного входа.
- **Когда:** Срабатывает расписание и одновременно нажата кнопка обновления.
- **Тогда:** Нет параллельного дублирования одного задания; доступные источники обновлены, проблемный имеет отдельный статус и старый timestamp.
- **Уровень:** `integration`.

#### AC-041

- **Дано:** Источник выдаёт несколько страниц с ограничением глубины; второй запрос завершился ошибкой.
- **Когда:** Импорт возобновляется.
- **Тогда:** Подтверждённые страницы сохранены без дублей; курсор не перескакивает пропуск; неполная история и её границы видны.
- **Уровень:** `integration`.

#### AC-048

- **Дано:** Сборщик имеет сессию личного кабинета с более широкими внешними правами.
- **Когда:** Возникают запрос на платёж, неподтверждённый маршрут или MFA/CAPTCHA.
- **Тогда:** Платёж и неизвестный маршрут блокируются; MFA/CAPTCHA передаётся владельцу, источник приостанавливается; остальные источники продолжают работать.
- **Уровень:** `integration`.

#### AC-087

- **Дано:** A владеет внешним аккаунтом, B инициирует повторную авторизацию или отключение.
- **Когда:** Запрашивается MFA; одновременно завершает работу старое задание синхронизации.
- **Тогда:** MFA адресован A; B видит статус, но не пароль/код/сессию. Отключение отзывает lease/version и запрещает применение старого результата; реальные платежи недоступны обоим.
- **Уровень:** `integration`.

#### AC-090

- **Дано:** В тестах созданы две изолированные семьи; запрос или задача подменяет householdId/actor/resourceId.
- **Когда:** Проверяются чтение файла, импорт, исправление, поиск AI и дедупликация.
- **Тогда:** Чужие объекты недоступны и не объединяются; сервер берёт principal из сессии или проверенного контекста задания. Отказ не раскрывает чужое содержимое.
- **Уровень:** `integration`.

#### AC-091

- **Дано:** Январский чек: общие товары 600 (50/50), личные A 100, B 300; в феврале доли статьи стали 60/40.
- **Когда:** В феврале возвращаются общие товары на 200.
- **Тогда:** Январь уменьшается семье на 200, каждому на 100 с исходной исторической оценкой; февральская пропорция не меняет январь, cash date возврата остаётся февральской.
- **Уровень:** `integration`.

#### AC-099

- **Дано:** Есть загрузка, пустой список/поиск, устаревшие/частичные данные, offline, отказ и конкурирующие правки.
- **Когда:** Пользователь выполняет чтение или сохранение.
- **Тогда:** Неизвестное не становится нулём, подтверждение даётся после readback; неизвестный исход проверяется по ID команды до повторного создания. Конфликт сохраняет ввод и предлагает сравнение. Истечение сессии ведёт к входу, банковская reauth — к нужному владельцу, ожидание AI не блокирует обычный учёт.
- **Уровень:** `manual+e2e`.

#### AC-050

- **Дано:** Существует приватный чек и активное подключение источника.
- **Когда:** Проверяются прямой URL файла, экспорт без сессии, логи и отзыв подключения.
- **Тогда:** Без авторизации доступ закрыт; секреты зашифрованы и не журналируются; отзыв подключения прекращает дальнейший сбор.
- **Уровень:** `integration`.

### Проверка результата

```sh
make e2e SCENARIO=connection-ui
```

Два аккаунта одного сервиса различаются; reauth не раскрывает секреты; ошибка объясняет следующий шаг, повтор sync не дублирует учёт.

Команды Make являются интерфейсами проверки; наличие target не доказывает реализацию соответствующей suite. Если требуемой suite ещё нет, реализовать её в рамках задачи; отсутствие suite не считается успешной проверкой. Фиксировать фактический результат каждого прогона. make docs-check проверяет документацию и исследовательские инструменты, но не финансовый runtime, браузеры или production. Live/manual-проверки, доступ и расходы фиксируются отдельно.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Understandable freshness and safe restoration of read access.

**Status:** Open; ready-for-agent. The description is ready for implementation; open prerequisites and the entry conditions below govern execution. The label does not establish completion, live admission or production authorization; SDD Ready is not a blocker.

**Dependencies:** `task-7.1`, `task-7.9`, `task-3.3`, `task-4.1`, `task-4.2`, `task-4.3`, `task-4.4`, `task-4.5`, `task-4.6`.

**Kind:** `implementation`.

### Change and contracts

### Current behavior

The desktop shell, separate RU/EN sign-in, design system and household-permission backend are included in the base. Their presence does not establish readiness of every screen and scenario in this task.

### Desired behavior

SCR-027–SCR-029: six platforms, multiple owners/accounts, history start/gaps, last success, manual sync and disconnect preserving the ledger. Only external-account owners reauthorize; partners see an action request without secrets. Reconnection uses stable identity. Render confirmed coverage, never declare partial coverage complete.

### Key interfaces

Connection API: freshness/coverage, selected history date, sync/disconnect and owner-only reauthorization; read-only deploymentGate.

### Out of scope

Desktop web on a macOS laptop only; do not add mobile screens. Financial calculations and authority checks remain server-side. Do not expand the agreed product scope.

### Blocked by

- [#48](https://github.com/andrey-krasheninnikov/want-keep/issues/48) (`task-7.1`): completed.
- [#56](https://github.com/andrey-krasheninnikov/want-keep/issues/56) (`task-7.9`): open; blocks execution.
- [#28](https://github.com/andrey-krasheninnikov/want-keep/issues/28) (`task-3.3`): completed.
- [#29](https://github.com/andrey-krasheninnikov/want-keep/issues/29) (`task-4.1`): completed.
- [#30](https://github.com/andrey-krasheninnikov/want-keep/issues/30) (`task-4.2`): completed.
- [#31](https://github.com/andrey-krasheninnikov/want-keep/issues/31) (`task-4.3`): open; blocks execution.
- [#32](https://github.com/andrey-krasheninnikov/want-keep/issues/32) (`task-4.4`): open; blocks execution.
- [#33](https://github.com/andrey-krasheninnikov/want-keep/issues/33) (`task-4.5`): open; blocks execution.
- [#34](https://github.com/andrey-krasheninnikov/want-keep/issues/34) (`task-4.6`): open; blocks execution.

### Change boundaries

- `web/src/features/connections/`

### Screen contract

### SCR-027 — Connections

`/connections`

**Question:** Are data updating and where is action needed?

**Primary answer:** Last successful read and issues per connection.

**Top-down structure:** Needs attention → six-platform accounts/owners → freshness/coverage → add.

**Next action:** Connect SCR-029, open SCR-028, refresh a specific source.

**Explanation and details:** Unconfirmed capability never appears operational; same platform may have distinct accounts.

**Permissions:** Both members can read; server checks membership and resource ownership for actions.

Forms: FORM-13.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-04, UISTATE-15.

### SCR-028 — Connection status

`/connections/:id`

**Question:** Why are this service’s data incomplete?

**Primary answer:** Plain cause, available period and required action.

**Top-down structure:** Status/owner → last success/next sync → products/history boundaries → actions.

**Next action:** Refresh, reauthorize SCR-029, disconnect with history-preservation preview.

**Explanation and details:** Errors/cursor/diagnostic code expand; partial import never implies complete history.

**Permissions:** Both members can read; server checks membership and resource ownership for actions.

Forms: FORM-13.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-15.

### SCR-029 — Platform authorization

`/connections/new; /connections/:id/reauth`

**Question:** How do I safely connect my account?

**Primary answer:** Read-only connection with credentials supplied by its owner.

**Top-down structure:** Platform/owner → read-access explanation → isolated input → MFA → outcome/history date.

**Next action:** Continue FORM-13 → SCR-028; partner sees waiting for owner.

**Explanation and details:** Expiry/cancel preserves existing ledger; secret never enters chat, logs or shared screen.

**Permissions:** Both manage; only external-account owner enters secrets.

Forms: FORM-13.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-15, UISTATE-17.

#### FORM-13 — Connection and reauth

**Fields:** Platform, external-account owner, history start, available products; secret only in isolated authorization flow.

**Validation and permissions:** Both manage; external owner alone supplies key/password/MFA. Adapter-confirmed read-only scopes, identity and coverage; no MFA/CAPTCHA bypass.

**Outcome:** Connected/syncing/awaiting owner/error; disconnect preserves history and invalidates generation.

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
- **UISTATE-15 — Bank sign-in needed:** Name connection and owner, offer safe owner sign-in; partner sees waiting without secret access.
- **UISTATE-16 — Confirmed:** After server-confirmed outcome show what changed, an object link and available correction; do not rely on a disappearing toast.
- **UISTATE-17 — Cancelled:** Explain that no new outcome was confirmed and offer explicit retry; cancelled system passkey prompts are not a malfunction.


Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-010:** A refund reduces expenses in the purchase month while preserving the actual cash receipt date.
- **REQ-016:** Receipt items allocate one paid amount across categories without duplicating the total.
- **REQ-040:** Each source refreshes hourly and on demand with a visible last-success timestamp.
- **REQ-041:** History retains coverage boundaries, cursors, gaps and source status.
- **REQ-048:** Integrations and the browser collector perform authorized read operations only.
- **REQ-050:** Files, source keys, sessions and financial records are protected against unauthorized access.
- **REQ-067:** Expenses and receipt items have personal or joint attribution; joint shares default to 50/50 with line or purchase overrides.
- **REQ-072:** Competing edits, clarification answers and reversals check revision and current permissions while retaining both authors.
- **REQ-073:** Both manage connections; the external-account owner performs bank authentication without exposing secrets to the partner or AI.
- **REQ-076:** Household scope is checked for APIs, files, AI, jobs and external IDs independently of supplied actor/owner fields.
- **REQ-082:** Screen states explain consequences and a safe next step without losing input.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-040

- **Given:** Two sources are available and a third requires sign-in again.
- **When:** The schedule fires while the refresh button is pressed.
- **Then:** The same job is not duplicated concurrently; available sources refresh and the failing source has its own status and old timestamp.
- **Level:** `integration`.

#### AC-041

- **Given:** A source provides paginated history with a retention limit; the second request fails.
- **When:** Import resumes.
- **Then:** Confirmed pages remain without duplicates; the cursor does not skip the gap; incomplete history and its boundaries are visible.
- **Level:** `integration`.

#### AC-048

- **Given:** The collector has a personal-account session with broader provider permissions.
- **When:** A payment request, unapproved route or MFA/CAPTCHA appears.
- **Then:** Payments and unknown routes are blocked; MFA/CAPTCHA is handed to the owner and that source pauses; other sources continue.
- **Level:** `integration`.

#### AC-087

- **Given:** A owns the external account and B initiates reauthorization or disconnect.
- **When:** MFA is requested while an old sync job completes.
- **Then:** MFA is addressed to A; B sees status but no password/code/session. Disconnect revokes lease/version and prevents stale-result application; actual payments are unavailable to both.
- **Level:** `integration`.

#### AC-090

- **Given:** Tests contain two isolated households; a request or job forges householdId/actor/resourceId.
- **When:** File reads, import, correction, AI retrieval and deduplication are exercised.
- **Then:** Foreign objects are inaccessible and never merged; the server takes principal from the session or validated job context. Denial reveals no foreign content.
- **Level:** `integration`.

#### AC-091

- **Given:** January receipt: joint items 600 (50/50), personal A 100, B 300; February plan shares became 60/40.
- **When:** Joint items worth 200 are returned in February.
- **Then:** January falls by 200 for the household and 100 for each member using original historical valuation; February shares do not rewrite January and refund cash date stays in February.
- **Level:** `integration`.

#### AC-099

- **Given:** Loading, empty list/search, stale/partial data, offline, failure and concurrent edits occur.
- **When:** The user reads or saves.
- **Then:** Unknown never becomes zero and success follows readback; unknown outcomes are reconciled by command ID before another creation. Conflicts retain input and offer comparison. Session expiry leads to sign-in, bank reauth to the proper owner, and AI waiting does not block ordinary accounting.
- **Level:** `manual+e2e`.

#### AC-050

- **Given:** A private receipt and an active source connection exist.
- **When:** A direct file URL, unauthenticated export, logs and disconnection are checked.
- **Then:** Unauthenticated access fails; secrets are encrypted and not logged; disconnecting stops further collection.
- **Level:** `integration`.

### Verification

```sh
make e2e SCENARIO=connection-ui
```

Two accounts on one service remain distinct; reauth exposes no secrets; errors explain next steps and repeat sync does not duplicate accounting.

Make commands are verification interfaces; the existence of a target does not establish an implemented suite. If a required suite is missing, implement it as part of the task; a missing suite is not a passing check. Record the actual outcome of each run. make docs-check validates documentation and research tooling, not financial runtime, browsers or production. Record live/manual checks, access and costs separately.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
