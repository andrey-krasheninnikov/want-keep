<!-- want-keep-task: task-7.15 -->
# task-7.15 — Добавить контекстные анимации финансовых событий / Add contextual financial-event animations

## RU

Понятная эмоциональная обратная связь без потери контроля и точности.

**Состояние:** Не начато; задача ожидает собственные зависимости и entry gates.

**Зависимости:** `task-7.11`, `task-7.2`, `task-7.4`, `task-7.5`, `task-7.14`.

**Тип:** `implementation`.

### Изменение и контракты

По design.md реализовать MOT-01–MOT-05: запуск ракеты при добавлении счёта в учёт, огоньки при подтверждённом пополнении накопления, конфетти при достигнутой цели, короткое мягкое диско как вариант крупного совместного достижения, спокойный сигнал превышения лимита. Только confirmed domain event, no inference from render/forecast/AI text. Дедуп по household event + user с подтверждением показа; sync/retry/history/backfill не запускают повтор. Отключение эффектов персонально; prefers-reduced-motion всегда убирает декоративное движение. CSS/SVG предпочтительны, без новых тяжёлых библиотек; один эффект за раз, не перекрывает формы, Escape завершает, без вспышек, звука и анимации финансового числа.

### Границы изменений

- `web/src/design-system/motion/`
- `web/src/features/accounts/`
- `web/src/features/goals/`
- `web/src/features/budget/`
- `web/src/features/settings/`
- `backend/internal/notifications/`
- `backend/internal/accounts/`
- `backend/internal/goals/`
- `backend/internal/budget/`

### Экранный контракт

### SCR-007 — Деньги и счета

`/accounts`

**Вопрос:** Где деньги и сколько доступно?

**Главный ответ:** Собственные средства, доступно, резерв и задолженность по валютам.

**Структура сверху вниз:** Сводка → личные/семейные группы счетов → доступно/резерв/блокировка/долг → свежесть.

**Следующее действие:** Открыть SCR-008, добавить счёт FORM-03 или подключить SCR-029; операции → SCR-009.

**Объяснение и детализация:** Эквивалент не означает обеспеченность другой валютой; скрытые из фильтра счета остаются частью семейного пула.

**Права:** Оба участника видят и исправляют факты любого счёта семьи; actor из сессии.

Forms: FORM-03.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-04, UISTATE-15.

### SCR-008 — Карточка счёта

`/accounts/:id`

**Вопрос:** Что доступно именно на этом счёте?

**Главный ответ:** Ответ зависит от продукта, но доступность и обязательства идут первыми.

**Структура сверху вниз:** Текущий: доступно/резерв/блокировки. Кредитка: долг, обязательный платёж/дата, сумма сохранения grace, свои средства. Вклад/Earn/Coinhold: факт/прогноз, срок/условия вывода. Крипто: активы/позиции, доступно/маржа/блокировка, результат/fees/funding/mining. Затем операции.

**Следующее действие:** Уточнить остаток → SCR-012; посмотреть доходность → SCR-021/022; учесть движение FORM-05.

**Объяснение и детализация:** Условия, источник, дата оценки и история раскрываются; неизвестный grace/вывод не имитировать.

**Права:** Оба участника видят и исправляют факты любого счёта семьи; actor из сессии.

Forms: FORM-03, FORM-05, FORM-06.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-04, UISTATE-15.

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

### SCR-031 — Настройки

`/settings`

**Вопрос:** Как настроить удобный учёт?

**Главный ответ:** Личные предпочтения и понятные разделы настроек.

**Структура сверху вниз:** Язык/валюта → уведомления → семья → безопасность → категории → состояние системы.

**Следующее действие:** Сохранить FORM-14; перейти SCR-032/033/034/035.

**Объяснение и детализация:** Смена языка/валюты не меняет финансовый факт или права; переключателя темы нет.

**Права:** Оба участника видят; действия проверяет сервер по членству и владельцу ресурса.

Forms: FORM-14.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-17.

#### FORM-03 — Счёт и начальный остаток

**Поля:** Название, валюта, личный/семейный наличный счёт, дата начала в timezone семьи и точный остаток. Импортный продукт имеет отдельные owned/available/locked/debt, подтверждение и источник; карты — алиасы без баланса.

**Проверки и права:** Оба member создают счета и исправляют факты учёта. При смене владельца или personal/household принадлежности существующего личного счёта требуется его текущий владелец; для семейного счёта — любой member. Проверенный внешний владелец и история операций этим не меняются. Точные decimal, валюта обязательна. Импортируемые поля меняются через correction; начальный остаток не доход. Ручное создание — только наличные, личный счёт только для себя. Перенос даты сохраняет операции до неё в истории; новое открытие заменяет прежнее в расчёте. Дата не может быть в будущем. Неизвестный ответ восстанавливается по прежнему Idempotency-Key через /commands; partial/unknown не равны нулю.

**Результат:** Счёт в учёте создан/исправлен с audit; это не открытие банковского продукта.

#### FORM-05 — Учесть перевод или обмен

**Поля:** Откуда/куда, даты, обе суммы/валюты, комиссии и счёт комиссии, существующие движения.

**Проверки и права:** Оба участника; разные счета семьи; одна исходящая и одна входящая сторона. Principal не входит в доходы/расходы; отдельные комиссии, включая третий актив. Пустой existingTransactions создаёт новое движение. Непустой список содержит ID/revisions всех участников (до 100); суммы, комиссии и дата основной записи проверяются без создания недостающих сторон. Связь переводов/обменов/оплаты подтверждается по версиям, неоднозначность остаётся в matching; отдельная покупка снимает ожидание один раз.

**Результат:** Связано движение денег в журнале; никакой реальной отправки или покупки актива.

#### FORM-06 — Исправление, сопоставление и отмена

**Поля:** Операция, expectedRevision, основание; полный principal и fees, дата, payer, raw merchant/note; category и merchant identity через set|clear, позиции через replace|clear. Undo: decisionId и expectedRevisions; сравнение до/после и с источником.

**Проверки и права:** Оба участника исправляют семейные факты. Сервер сохраняет actor, счета/активы principal и происхождение, проверяет активные категории/продавцов и меняет полный item set атомарно. Позиции и скидки точно равны principal; top-level category с позициями запрещена. Undo сохраняет поздние независимые поля. Сопоставление и доли остаются профильным задачам.

**Результат:** Новое решение и финансовые revisions с историей, либо no_change/conflict без эффекта и потери ввода. Исключение не меняет банковский статус; undo пересчитывает текущий эффект.

#### FORM-11 — Цель и резерв

**Поля:** Название, personal/shared, владелец если личная, сумма/валюта/срок; явный резерв или выделенный счёт; изменение суммы резерва.

**Проверки и права:** Личную меняет владелец, совместную любой member. Preview свободных денег/дневного лимита в исходной валюте; no double reserve; совместная без личных долей.

**Результат:** Цель/резерв обновлены один раз, сообщение партнёру и audit.

#### FORM-12 — Ответ и предложение AI

**Поля:** expectedRevision вопроса/предложения, subjectExpectedRevision операции и ровно choiceId либо свободный answer; для предложения явное apply/reject.

**Проверки и права:** Оба для операций, только владелец для личного плана/цели; actor не меняется текстом. Одновременный ответ проверяет версию; preview перед финансовым изменением.

**Результат:** Команда подтверждена, отказана, устарела или ожидает; ответ/основание и автор сохранены.

#### FORM-14 — Личные настройки и безопасность

**Поля:** Язык, валюта отображения, push; имя passkey, отзыв своего устройства, перевыпуск собственных recovery-кодов.

**Проверки и права:** Только собственная безопасность; опасные изменения требуют reauthentication по auth-контракту. Не удалять последний путь входа без замены. Запрет push не блокирует in-app.

**Результат:** Персональные предпочтения сохранены; отозванная сессия/подписка перестаёт работать, коды не попадают в чат.

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
- **UISTATE-15 — Нужен банковский вход:** Назвать подключение и владельца, дать ему безопасно войти; партнёру показать ожидание без доступа к секрету.
- **UISTATE-16 — Подтверждено:** После подтверждённого сервером результата показать что изменилось, ссылку на объект и доступное исправление; не полагаться на исчезающий toast.
- **UISTATE-17 — Отмена:** Объяснить отсутствие нового подтверждённого результата, дать повторить явно; не выдавать отмену системного passkey за поломку.


Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-077:** Тёмные токены Want Keep: сдержанный киберпанк и архитектурный ритм Ближнего Востока.
- **REQ-082:** Экранные состояния объясняют последствия и безопасный следующий шаг без потери ввода.
- **REQ-084:** Доступность проверяется на реальных Chrome и Arc, включая клавиатуру, фокус, контраст, масштаб и reduced motion.
- **REQ-087:** Контекстные pixel-анимации подтверждают значимые события и предупреждают о лимитах, сохраняя доступность и достоверность результата.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-094

- **Дано:** Обзор, вход, форма и таблица используют одну дизайн-систему.
- **Когда:** Проверяются фон, поверхности, акценты и состояния.
- **Тогда:** Фон #1A1A1A, рабочие поверхности #202020–#262626, бренд #5F4EF5; нет светлой темы. Брендовые плоскости #111114/#18171E, редкие песочные акценты #C7AF8F. Нет обводок карточек/полей; группировка заливкой, пространством и типографикой. Фокус заметен инверсией заливки; статусы имеют текст/значок.
- **Уровень:** `manual+e2e`.

#### AC-099

- **Дано:** Есть загрузка, пустой список/поиск, устаревшие/частичные данные, offline, отказ и конкурирующие правки.
- **Когда:** Пользователь выполняет чтение или сохранение.
- **Тогда:** Неизвестное не становится нулём, подтверждение даётся после readback; неизвестный исход проверяется по ID команды до повторного создания. Конфликт сохраняет ввод и предлагает сравнение. Истечение сессии ведёт к входу, банковская reauth — к нужному владельцу, ожидание AI не блокирует обычный учёт.
- **Уровень:** `manual+e2e`.

#### AC-101

- **Дано:** Экраны и формы доступны в RU/EN на macOS.
- **Когда:** Проверяются 1280×720 и 1440×900 CSS px, масштаб 100%/200%, клавиатура и уменьшение движения.
- **Тогда:** Нет скрытых действий, обрезанных сумм и горизонтальной прокрутки форм; таблицы при необходимости имеют обозначенную область прокрутки. Контраст обычного текста ≥4.5:1, крупного ≥3:1, значимых границ/фокуса ≥3:1. Фокус видим и возвращается, статусы доступны без цвета. Записаны реальные версии Chrome/Arc/macOS; Chromium CI отдельно.
- **Уровень:** `manual+e2e`.

#### AC-104

- **Дано:** Включены эффекты: добавление счёта в учёт, подтверждённое пополнение накопления, достижение цели и превышение лимита; есть повторы синхронизации и reduced motion.
- **Когда:** Пользователь получает подтверждённое событие, открывает страницу повторно, отключает эффекты или включает reduced motion.
- **Тогда:** Ракета/огоньки/конфетти/мягкое диско применяются по design.md и ID события, не повторяются от refresh/retry/backfill. Превышение лимита даёт спокойное предупреждение с действием, не награду. Нет вспышек/стробоскопа, блокировки формы или скрытого текста; отключение и reduced motion заменяют эффект статичным подтверждением. Финансовые цифры не анимируются через ложные промежуточные значения.
- **Уровень:** `manual+e2e`.

### Проверка результата

```sh
make e2e SCENARIO=event-motion
```

Для каждого MOT проверены событие/повтор/отмена факта/refresh, оба участника, настройка off и reduced motion, Chrome/Arc. Нет повторной награды за импорт/обновление и празднования перерасхода; статичный результат всегда доступен.

Команды `make` — будущий контракт, создаваемый task-1.1; сейчас они не существуют. Live/paid/manual проверки отдельно фиксируют доступ и фактический результат. Исследования не обходят блокер отсутствующего доступа.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Understandable expressive feedback without losing control or accuracy.

**Status:** Not started; the task awaits its own dependencies and entry gates.

**Dependencies:** `task-7.11`, `task-7.2`, `task-7.4`, `task-7.5`, `task-7.14`.

**Kind:** `implementation`.

### Change and contracts

Implement MOT-01–MOT-05 from design.en.md: rocket for adding an accounting account, sparkles for confirmed savings top-up, confetti for goal reached, short soft disco as a large joint-achievement variant, calm limit-breach signal. Only confirmed domain events, never render/forecast/AI-text inference. Deduplicate by household event + user with presentation acknowledgement; sync/retry/history/backfill never replay. Effects preference is personal; prefers-reduced-motion always removes decorative movement. Prefer CSS/SVG without new heavy libraries; one effect at a time, no form obstruction, Escape ends it, no flashing, sound or financial-number animation.

### Change boundaries

- `web/src/design-system/motion/`
- `web/src/features/accounts/`
- `web/src/features/goals/`
- `web/src/features/budget/`
- `web/src/features/settings/`
- `backend/internal/notifications/`
- `backend/internal/accounts/`
- `backend/internal/goals/`
- `backend/internal/budget/`

### Screen contract

### SCR-007 — Money and accounts

`/accounts`

**Question:** Where is the money and how much is available?

**Primary answer:** Own funds, available, reserved and debt by currency.

**Top-down structure:** Summary → personal/household account groups → available/reserved/blocked/debt → freshness.

**Next action:** Open SCR-008, add account FORM-03 or connect SCR-029; transactions → SCR-009.

**Explanation and details:** Equivalent does not fund another currency; accounts outside a view filter remain in the household pool.

**Permissions:** Both members read/correct facts for any household account; actor from session.

Forms: FORM-03.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-04, UISTATE-15.

### SCR-008 — Account details

`/accounts/:id`

**Question:** What is available in this account?

**Primary answer:** The answer depends on product; availability and obligations come first.

**Top-down structure:** Current: available/reserved/blocked. Credit: debt, required payment/date, grace-preserving amount, own funds. Deposit/Earn/Coinhold: actual/forecast, maturity/withdrawal terms. Crypto: assets/positions, available/margin/blocked, P&L/fees/funding/mining. Then transactions.

**Next action:** Resolve balance → SCR-012; returns → SCR-021/022; record movement FORM-05.

**Explanation and details:** Terms, source, valuation date and history expand; never invent unknown grace/withdrawal terms.

**Permissions:** Both members read/correct facts for any household account; actor from session.

Forms: FORM-03, FORM-05, FORM-06.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-04, UISTATE-15.

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

### SCR-031 — Settings

`/settings`

**Question:** How do I configure comfortable accounting?

**Primary answer:** Personal preferences and clear settings groups.

**Top-down structure:** Language/currency → notifications → household → security → categories → system health.

**Next action:** Save FORM-14; open SCR-032/033/034/035.

**Explanation and details:** Changing language/currency changes neither financial fact nor authority; no theme toggle.

**Permissions:** Both members can read; server checks membership and resource ownership for actions.

Forms: FORM-14.

States: UISTATE-01, UISTATE-02, UISTATE-03, UISTATE-05, UISTATE-06, UISTATE-07, UISTATE-08, UISTATE-12, UISTATE-13, UISTATE-09, UISTATE-10, UISTATE-11, UISTATE-16, UISTATE-17.

#### FORM-03 — Account and opening balance

**Fields:** Name, asset, personal/household cash account, start date in household timezone and exact opening balance. Imported products have separate owned/available/locked/debt, confirmation and provenance; cards are balance-free aliases.

**Validation and permissions:** Both members create accounts and correct accounting facts. Changing owner or personal/household scope of an existing personal account requires its current owner; either member may change a household account. This never changes verified external ownership or transaction history. Exact decimals and currency required. Imported fields change through correction; opening balance is not income. Manual creation is cash only; a personal account is created for oneself. Moving the date keeps earlier operations in history; the new opening replaces the previous calculation input. The date cannot be in the future. Unknown outcomes use the original Idempotency-Key via /commands; partial/unknown are not zero.

**Outcome:** Accounting account created/corrected with audit; this does not open a bank product.

#### FORM-05 — Record transfer or exchange

**Fields:** From/to accounts, dates, both amounts/currencies, fees/fee account, existing movements.

**Validation and permissions:** Either member; distinct household accounts; one outgoing and one incoming side. Principal is excluded from income/expenses; separate fees may use a third asset. Empty existingTransactions creates new movement. A nonempty list supplies all participant IDs/revisions (up to 100); amounts, fees and primary date are checked without creating missing sides. Transfer/exchange/payment links require version checks; ambiguity remains in matching; separate-purchase confirmation releases waiting once.

**Outcome:** Ledger movements linked; no actual transfer or asset purchase.

#### FORM-06 — Correction, matching and undo

**Fields:** Transaction, expectedRevision and reason; complete principal and fees, date, payer, raw merchant/note; category and merchant identity via set|clear, items via replace|clear. Undo uses decisionId and expectedRevisions; compare before/after and source.

**Validation and permissions:** Both members correct household facts. The server retains actor, principal accounts/assets and provenance, validates active categories/merchants and replaces the complete item set atomically. Items and discounts equal principal exactly; top-level category with items is forbidden. Undo preserves later independent fields. Matching and shares remain with their owning tasks.

**Outcome:** New decision and financial revisions with history, or no_change/conflict without effect or lost input. Exclusion does not change bank state; undo recomputes the current effect.

#### FORM-11 — Goal and reserve

**Fields:** Name, personal/shared, personal owner if applicable, amount/currency/deadline; explicit reserve or dedicated account; reserve delta.

**Validation and permissions:** Owner edits personal, any member joint. Preview free money/daily allowance in native currency; no double reserve; joint goals have no personal shares.

**Outcome:** Goal/reserve updated once, partner notification and audit.

#### FORM-12 — AI response and proposal

**Fields:** Question/proposal expectedRevision, transaction subjectExpectedRevision and exactly choiceId or free-text answer; explicit apply/reject for a proposal.

**Validation and permissions:** Either member for transactions, owner only for personal plan/goal; text cannot change actor. Concurrent response checks revision; preview before financial change.

**Outcome:** Command confirmed, rejected, stale or pending; response/reason and author retained.

#### FORM-14 — Personal preferences and security

**Fields:** Language, display currency, push; passkey name, revoke own device, regenerate own recovery codes.

**Validation and permissions:** Own security only; sensitive changes require auth-contract reauthentication. Do not remove the last access path without replacement. Push denial never blocks in-app.

**Outcome:** Personal preferences saved; revoked session/subscription stops working and codes never enter chat.

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
- **UISTATE-15 — Bank sign-in needed:** Name connection and owner, offer safe owner sign-in; partner sees waiting without secret access.
- **UISTATE-16 — Confirmed:** After server-confirmed outcome show what changed, an object link and available correction; do not rely on a disappearing toast.
- **UISTATE-17 — Cancelled:** Explain that no new outcome was confirmed and offer explicit retry; cancelled system passkey prompts are not a malfunction.


Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-077:** Dark Want Keep tokens: restrained cyberpunk and Middle Eastern architectural rhythm.
- **REQ-082:** Screen states explain consequences and a safe next step without losing input.
- **REQ-084:** Accessibility is checked in actual Chrome and Arc, including keyboard, focus, contrast, zoom and reduced motion.
- **REQ-087:** Contextual pixel animations acknowledge milestones and warn about limits while preserving accessibility and truthful outcomes.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-094

- **Given:** Overview, sign-in, form and table use one design system.
- **When:** Backgrounds, surfaces, accents and states are inspected.
- **Then:** Base #1A1A1A, working surfaces #202020–#262626, brand #5F4EF5; no light theme. Brand planes #111114/#18171E, sparse sand accents #C7AF8F. No card/field outlines; group through fills, space and typography. Focus uses visible inverted fill; statuses have text/icons.
- **Level:** `manual+e2e`.

#### AC-099

- **Given:** Loading, empty list/search, stale/partial data, offline, failure and concurrent edits occur.
- **When:** The user reads or saves.
- **Then:** Unknown never becomes zero and success follows readback; unknown outcomes are reconciled by command ID before another creation. Conflicts retain input and offer comparison. Session expiry leads to sign-in, bank reauth to the proper owner, and AI waiting does not block ordinary accounting.
- **Level:** `manual+e2e`.

#### AC-101

- **Given:** Screens and forms are available in RU/EN on macOS.
- **When:** 1280×720 and 1440×900 CSS px, 100%/200% zoom, keyboard and reduced motion are tested.
- **Then:** No hidden actions, clipped amounts or horizontally scrolling forms; tables have a labelled scroll region when needed. Normal text contrast ≥4.5:1, large text ≥3:1, meaningful boundaries/focus ≥3:1. Focus is visible and restored; states work without color. Actual Chrome/Arc/macOS versions are recorded separately from Chromium CI.
- **Level:** `manual+e2e`.

#### AC-104

- **Given:** Effects are enabled for adding an accounting account, confirmed savings top-up, goal achievement and limit breach; repeated sync and reduced motion occur.
- **When:** The user receives a confirmed event, revisits the page, disables effects or enables reduced motion.
- **Then:** Rocket/sparkles/confetti/soft disco follow design.en.md and event IDs, never replay from refresh/retry/backfill. Limit breach gets a calm actionable warning, not a reward. No flashes/strobe, blocked forms or hidden text; disabled/reduced motion uses a static acknowledgement. Financial figures never animate through false intermediate values.
- **Level:** `manual+e2e`.

### Verification

```sh
make e2e SCENARIO=event-motion
```

Every MOT covers trigger/repeat/fact correction/refresh, both members, off preference and reduced motion in Chrome/Arc. No replay reward on import/refresh or celebration of overspending; static outcome always available.

The `make` commands are a future contract established by task-1.1; they do not exist yet. Live/paid/manual checks separately record access and actual outcomes. Research does not bypass missing-access blockers.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
