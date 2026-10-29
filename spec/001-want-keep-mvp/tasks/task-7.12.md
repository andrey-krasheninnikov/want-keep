<!-- want-keep-task: task-7.12 -->
# task-7.12 — Проверить desktop UX и визуальную приёмку / Verify desktop UX and visual acceptance

## RU

Доказать понятность семи сценариев на ноутбуке.

**Состояние:** Открыта; ready-for-human. Приёмка требует участия пользователей и отдельных live/manual-доказательств. Начало зависит от незакрытых задач и условий ниже; SDD Ready не является блокером.

**Зависимости:** `task-7.2`, `task-7.3`, `task-7.4`, `task-7.5`, `task-7.6`, `task-7.7`, `task-7.8`, `task-7.9`, `task-7.13`, `task-7.14`, `task-7.15`.

**Тип:** `verification`.

### Изменение и контракты

### Текущее поведение

Desktop-оболочка, отдельный вход RU/EN, дизайн-система и backend семейных прав включены в базу. Их наличие не доказывает готовность всех экранов и сценариев этой карточки.

### Требуемое поведение

Проверить все SCR-001–SCR-035 и их состояния по screens.md, включая продуктовые варианты SCR-008. Chromium E2E дополнить реальными Chrome/Arc на macOS, RU/EN, 1280×720/1440×900 и zoom 200%, keyboard/reduced motion. Оба участника выполняют семь сценариев без подсказок разработчика. Фиксировать задания, правильные ожидаемые финансовые ответы, фактический результат, время, ошибки, браузер/OS/viewport и артефакты без личных данных. Исправления направлять владельцу UI/домена и повторять затронутые сценарии.

### Ключевые контракты

SCR-001–SCR-035, FORM/UISTATE, семь пользовательских заданий, автоматический Chromium E2E и отдельный ручной протокол Chrome/Arc.

### Вне объёма

Только desktop web на ноутбуке с macOS; мобильные экраны не добавлять. Финансовые расчёты и проверка полномочий остаются на сервере. Не расширять согласованный продуктовый объём.

### Блокирующие зависимости

- [#49](https://github.com/andrey-krasheninnikov/want-keep/issues/49) (`task-7.2`): открыта; блокирует выполнение.
- [#50](https://github.com/andrey-krasheninnikov/want-keep/issues/50) (`task-7.3`): открыта; блокирует выполнение.
- [#51](https://github.com/andrey-krasheninnikov/want-keep/issues/51) (`task-7.4`): открыта; блокирует выполнение.
- [#52](https://github.com/andrey-krasheninnikov/want-keep/issues/52) (`task-7.5`): открыта; блокирует выполнение.
- [#53](https://github.com/andrey-krasheninnikov/want-keep/issues/53) (`task-7.6`): открыта; блокирует выполнение.
- [#54](https://github.com/andrey-krasheninnikov/want-keep/issues/54) (`task-7.7`): открыта; блокирует выполнение.
- [#55](https://github.com/andrey-krasheninnikov/want-keep/issues/55) (`task-7.8`): открыта; блокирует выполнение.
- [#56](https://github.com/andrey-krasheninnikov/want-keep/issues/56) (`task-7.9`): открыта; блокирует выполнение.
- [#60](https://github.com/andrey-krasheninnikov/want-keep/issues/60) (`task-7.13`): открыта; блокирует выполнение.
- [#61](https://github.com/andrey-krasheninnikov/want-keep/issues/61) (`task-7.14`): открыта; блокирует выполнение.
- [#62](https://github.com/andrey-krasheninnikov/want-keep/issues/62) (`task-7.15`): открыта; блокирует выполнение.

### Границы изменений

- `web/tests/`
- `docs/verification/desktop-ux/`

Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-054:** Интерфейс, чат и документация поддерживают RU/EN без изменения финансовой семантики.
- **REQ-055:** Веб-приложение предназначено для ноутбука macOS в Chrome и Arc; изменение окна и масштаба сохраняет доступность ежедневного учёта.
- **REQ-077:** Тёмные токены Want Keep: сдержанный киберпанк и архитектурный ритм Ближнего Востока.
- **REQ-078:** Pixelify Sans используется для бренда и крупных акцентов, Manrope — для повседневного интерфейса.
- **REQ-079:** Компоненты shadcn/ui на Base UI принадлежат проекту и оформляются собственными семантическими токенами.
- **REQ-080:** Каждый экран отвечает на вопрос пользователя и ведёт к следующему полезному действию.
- **REQ-081:** Навигация desktop сохраняет контекст и не меняет права при смене представления семьи.
- **REQ-082:** Экранные состояния объясняют последствия и безопасный следующий шаг без потери ввода.
- **REQ-083:** Экран входа сохраняет композицию присланного референса и личное восстановление доступа.
- **REQ-084:** Доступность проверяется на реальных Chrome и Arc, включая клавиатуру, фокус, контраст, масштаб и reduced motion.
- **REQ-085:** Семь основных пользовательских задач выполняются без помощи разработчика с объяснимым финансовым результатом.
- **REQ-086:** Детализация счёта зависит от продукта и показывает доступность денег перед служебными сведениями.
- **REQ-087:** Контекстные pixel-анимации подтверждают значимые события и предупреждают о лимитах, сохраняя доступность и достоверность результата.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-055

- **Дано:** Владелец проверяет день с расходами, чеком, уточнением, бюджетом и целью.
- **Когда:** Участник проходит сценарий в реальных Chrome и Arc при 1280×720 и 1440×900 CSS px, затем увеличивает масштаб до 200%.
- **Тогда:** Основные действия доступны без потери данных и горизонтального прокручивания форм; измерено фактическое время сценария относительно личного ориентира до 45 минут в день.
- **Уровень:** `manual`.

#### AC-075

- **Дано:** Один сценарий ввода чека и исправления категории доступен на двух языках.
- **Когда:** Сценарий выполняется с клавиатурой в Chrome и Arc на macOS в обоих контрольных размерах и при увеличении масштаба.
- **Тогда:** Все обязательные поля и ошибки доступны; переключение языка не сбрасывает ввод; суммы локализуются только при отображении.
- **Уровень:** `end-to-end+manual`.

#### AC-094

- **Дано:** Обзор, вход, форма и таблица используют одну дизайн-систему.
- **Когда:** Проверяются фон, поверхности, акценты и состояния.
- **Тогда:** Фон #1A1A1A, рабочие поверхности #202020–#262626, бренд #5F4EF5; нет светлой темы. Брендовые плоскости #111114/#18171E, редкие песочные акценты #C7AF8F. Нет обводок карточек/полей; группировка заливкой, пространством и типографикой. Фокус заметен инверсией заливки; статусы имеют текст/значок.
- **Уровень:** `manual+e2e`.

#### AC-095

- **Дано:** Есть RU/EN строки, RUB/USD/USDT/USDC/BTC/ETH и длинные точные суммы.
- **Когда:** Проверяются загруженные и недоступные шрифты.
- **Тогда:** Шрифты размещены локально с лицензиями; кириллица и валютные символы читаемы, fallback не теряет символы или разряды. Таблицы и формы используют Manrope, крупные суммы могут использовать Pixelify Sans.
- **Уровень:** `manual+e2e`.

#### AC-096

- **Дано:** Каталог содержит кнопки, поля, диалоги, панели, таблицы и состояния данных.
- **Когда:** Проверяются состояния и реализация примитивов.
- **Тогда:** Зафиксированы версии и Base UI; компоненты не сохраняют стандартное оформление shadcn, не дублируют бизнес-логику и поддерживают клавиатуру, подписи и фокус.
- **Уровень:** `manual+e2e`.

#### AC-097

- **Дано:** Доступны все SCR-001–SCR-035.
- **Когда:** Проверяется порядок ответа, действия, объяснения и детализации.
- **Тогда:** Главные суммы подписаны по смыслу; обзор начинает с доступного лимита и обязательств. Прогноз/факт/резерв различаются. Источник, ID и аудит раскрываются по запросу; график ведёт к объясняющим операциям.
- **Уровень:** `manual+e2e`.

#### AC-098

- **Дано:** Пользователь отфильтровал месяц, валюту, участника и список.
- **Когда:** Он открывает детализацию, возвращается, меняет язык и открывает прямую ссылку.
- **Тогда:** Обзор открывается после входа; левое меню содержит Обзор, Деньги, План, Аналитика, Чат, внизу Подключения/Настройки, сверху уведомления. Контекст сохраняется; сервер проверяет текущего автора независимо от фильтра.
- **Уровень:** `manual+e2e`.

#### AC-099

- **Дано:** Есть загрузка, пустой список/поиск, устаревшие/частичные данные, offline, отказ и конкурирующие правки.
- **Когда:** Пользователь выполняет чтение или сохранение.
- **Тогда:** Неизвестное не становится нулём, подтверждение даётся после readback; неизвестный исход проверяется по ID команды до повторного создания. Конфликт сохраняет ввод и предлагает сравнение. Истечение сессии ведёт к входу, банковская reauth — к нужному владельцу, ожидание AI не блокирует обычный учёт.
- **Уровень:** `manual+e2e`.

#### AC-100

- **Дано:** Участник открывает /login в обоих языках.
- **Когда:** Вход ожидает passkey, отменён или завершился ошибкой.
- **Тогда:** Компактные логотип/Want Keep по центру, свободное пространство, одна основная кнопка «Войти с passkey» / «Log in with Passkeys», ненавязчивые язык и помощь. Статус не ломает композицию; помощь ведёт к личному восстановлению. SVG не искажается.
- **Уровень:** `manual+e2e`.

#### AC-101

- **Дано:** Экраны и формы доступны в RU/EN на macOS.
- **Когда:** Проверяются 1280×720 и 1440×900 CSS px, масштаб 100%/200%, клавиатура и уменьшение движения.
- **Тогда:** Нет скрытых действий, обрезанных сумм и горизонтальной прокрутки форм; таблицы при необходимости имеют обозначенную область прокрутки. Контраст обычного текста ≥4.5:1, крупного ≥3:1, значимых границ/фокуса ≥3:1. Фокус видим и возвращается, статусы доступны без цвета. Записаны реальные версии Chrome/Arc/macOS; Chromium CI отдельно.
- **Уровень:** `manual+e2e`.

#### AC-102

- **Дано:** Оба участника получают синтетические данные и задания без подсказок по интерфейсу.
- **Когда:** Они определяют лимит/нехватку, объясняют сумму обзора, вносят чек, меняют доли, резерв цели, устраняют reauth/ожидание AI, входят и восстанавливают доступ.
- **Тогда:** Каждый завершает все семь сценариев, правильно объясняет результат и отсутствие двойного учёта. Фиксируются время, ошибки, затруднения и версии браузеров; ошибочное финансовое понимание или помощь разработчика требуют исправления и повторения затронутого сценария.
- **Уровень:** `manual+e2e`.

#### AC-103

- **Дано:** Есть текущий счёт, кредитка, вклад/Earn/Coinhold и криптопродукт с неполными данными.
- **Когда:** Пользователь открывает каждый вариант SCR-008.
- **Тогда:** Текущий счёт показывает доступно/резерв/блокировки, кредитка долг/ближайший платёж/grace, накопление факт/прогноз/срок, крипто доступные активы/позиции/комиссии. Неизвестные условия помечены, заёмные деньги не объявляются доступным семейным пулом.
- **Уровень:** `manual+e2e`.

#### AC-104

- **Дано:** Включены эффекты: добавление счёта в учёт, подтверждённое пополнение накопления, достижение цели и превышение лимита; есть повторы синхронизации и reduced motion.
- **Когда:** Пользователь получает подтверждённое событие, открывает страницу повторно, отключает эффекты или включает reduced motion.
- **Тогда:** Ракета/огоньки/конфетти/мягкое диско применяются по design.md и ID события, не повторяются от refresh/retry/backfill. Превышение лимита даёт спокойное предупреждение с действием, не награду. Нет вспышек/стробоскопа, блокировки формы или скрытого текста; отключение и reduced motion заменяют эффект статичным подтверждением. Финансовые цифры не анимируются через ложные промежуточные значения.
- **Уровень:** `manual+e2e`.

### Проверка результата

```sh
make e2e SCENARIO=desktop-ux
```

Автоматический отчёт плюс отдельный ручной протокол Chrome/Arc и обоих пользователей; отсутствие реального браузера оставляет приёмку незавершённой.

Команды Make являются интерфейсами проверки; наличие target не доказывает реализацию соответствующей suite. Если требуемой suite ещё нет, реализовать её в рамках задачи; отсутствие suite не считается успешной проверкой. Фиксировать фактический результат каждого прогона. make docs-check проверяет документацию и исследовательские инструменты, но не финансовый runtime, браузеры или production. Live/manual-проверки, доступ и расходы фиксируются отдельно.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Demonstrate seven understandable laptop flows.

**Status:** Open; ready-for-human. Acceptance requires user participation and separate live/manual evidence. Open prerequisites and the entry conditions below govern execution; SDD Ready is not a blocker.

**Dependencies:** `task-7.2`, `task-7.3`, `task-7.4`, `task-7.5`, `task-7.6`, `task-7.7`, `task-7.8`, `task-7.9`, `task-7.13`, `task-7.14`, `task-7.15`.

**Kind:** `verification`.

### Change and contracts

### Current behavior

The desktop shell, separate RU/EN sign-in, design system and household-permission backend are included in the base. Their presence does not establish readiness of every screen and scenario in this task.

### Desired behavior

Verify all SCR-001–SCR-035 and their states from screens.en.md including SCR-008 product variants. Supplement Chromium E2E with actual Chrome/Arc on macOS, RU/EN, 1280×720/1440×900 and 200% zoom, keyboard/reduced motion. Both members complete seven flows without developer hints. Record tasks, correct expected financial answers, actual outcomes, time, mistakes, browser/OS/viewport and nonprivate artifacts. Route fixes to the UI/domain owner and repeat affected flows.

### Key interfaces

SCR-001–SCR-035, FORM/UISTATE, seven user tasks, automated Chromium E2E and a separate manual Chrome/Arc protocol.

### Out of scope

Desktop web on a macOS laptop only; do not add mobile screens. Financial calculations and authority checks remain server-side. Do not expand the agreed product scope.

### Blocked by

- [#49](https://github.com/andrey-krasheninnikov/want-keep/issues/49) (`task-7.2`): open; blocks execution.
- [#50](https://github.com/andrey-krasheninnikov/want-keep/issues/50) (`task-7.3`): open; blocks execution.
- [#51](https://github.com/andrey-krasheninnikov/want-keep/issues/51) (`task-7.4`): open; blocks execution.
- [#52](https://github.com/andrey-krasheninnikov/want-keep/issues/52) (`task-7.5`): open; blocks execution.
- [#53](https://github.com/andrey-krasheninnikov/want-keep/issues/53) (`task-7.6`): open; blocks execution.
- [#54](https://github.com/andrey-krasheninnikov/want-keep/issues/54) (`task-7.7`): open; blocks execution.
- [#55](https://github.com/andrey-krasheninnikov/want-keep/issues/55) (`task-7.8`): open; blocks execution.
- [#56](https://github.com/andrey-krasheninnikov/want-keep/issues/56) (`task-7.9`): open; blocks execution.
- [#60](https://github.com/andrey-krasheninnikov/want-keep/issues/60) (`task-7.13`): open; blocks execution.
- [#61](https://github.com/andrey-krasheninnikov/want-keep/issues/61) (`task-7.14`): open; blocks execution.
- [#62](https://github.com/andrey-krasheninnikov/want-keep/issues/62) (`task-7.15`): open; blocks execution.

### Change boundaries

- `web/tests/`
- `docs/verification/desktop-ux/`

Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-054:** UI, chat and documentation support RU/EN without changing financial semantics.
- **REQ-055:** The web app targets macOS laptops in Chrome and Arc; window resizing and zoom preserve daily accounting access.
- **REQ-077:** Dark Want Keep tokens: restrained cyberpunk and Middle Eastern architectural rhythm.
- **REQ-078:** Pixelify Sans serves branding and large accents; Manrope serves everyday UI.
- **REQ-079:** Project-owned shadcn/ui components on Base UI use custom semantic tokens.
- **REQ-080:** Each screen answers a user question and leads to a useful next action.
- **REQ-081:** Desktop navigation preserves context and changing household views never changes authority.
- **REQ-082:** Screen states explain consequences and a safe next step without losing input.
- **REQ-083:** Sign-in preserves the supplied reference composition and personal access recovery.
- **REQ-084:** Accessibility is checked in actual Chrome and Arc, including keyboard, focus, contrast, zoom and reduced motion.
- **REQ-085:** Seven core user tasks are completed without developer help with explainable financial outcomes.
- **REQ-086:** Account details depend on the product and show availability before technical details.
- **REQ-087:** Contextual pixel animations acknowledge milestones and warn about limits while preserving accessibility and truthful outcomes.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-055

- **Given:** The owner reviews a day containing expenses, a receipt, clarification, budget and goal.
- **When:** A member completes the flow in actual Chrome and Arc at 1280×720 and 1440×900 CSS px, then zooms to 200%.
- **Then:** Core actions work without data loss or horizontally scrolling forms; observed flow time is recorded against the owner's up-to-45-minutes/day target.
- **Level:** `manual`.

#### AC-075

- **Given:** The same receipt-entry/category-correction flow exists in both languages.
- **When:** The flow runs with a keyboard in Chrome and Arc on macOS at both reference sizes and with zoom.
- **Then:** Required fields and errors remain accessible; language switching preserves input; amounts are localized only for display.
- **Level:** `end-to-end+manual`.

#### AC-094

- **Given:** Overview, sign-in, form and table use one design system.
- **When:** Backgrounds, surfaces, accents and states are inspected.
- **Then:** Base #1A1A1A, working surfaces #202020–#262626, brand #5F4EF5; no light theme. Brand planes #111114/#18171E, sparse sand accents #C7AF8F. No card/field outlines; group through fills, space and typography. Focus uses visible inverted fill; statuses have text/icons.
- **Level:** `manual+e2e`.

#### AC-095

- **Given:** RU/EN strings, RUB/USD/USDT/USDC/BTC/ETH and long precise amounts exist.
- **When:** Loaded and unavailable fonts are tested.
- **Then:** Fonts are hosted locally with licenses; Cyrillic and currency symbols remain legible and fallback loses no glyphs or digits. Tables/forms use Manrope; large amounts may use Pixelify Sans.
- **Level:** `manual+e2e`.

#### AC-096

- **Given:** A catalog contains buttons, fields, dialogs, panels, tables and data states.
- **When:** States and primitive implementations are inspected.
- **Then:** Versions and Base UI are pinned; components replace stock shadcn styling, contain no duplicated business rules and support keyboard, labels and focus.
- **Level:** `manual+e2e`.

#### AC-097

- **Given:** All SCR-001–SCR-035 are available.
- **When:** Answer, action, explanation and detail order is inspected.
- **Then:** Primary amounts have meaningful labels; overview starts with available allowance and obligations. Forecast/actual/reserve are distinct. Source, IDs and audit are expandable; charts lead to explanatory transactions.
- **Level:** `manual+e2e`.

#### AC-098

- **Given:** The user filtered month, currency, member and list.
- **When:** They open details, return, change language and open a direct link.
- **Then:** Overview opens after sign-in; left navigation contains Overview, Money, Plan, Analytics, Chat, lower Connections/Settings and top notifications. Context survives; the server checks the current actor independently of filters.
- **Level:** `manual+e2e`.

#### AC-099

- **Given:** Loading, empty list/search, stale/partial data, offline, failure and concurrent edits occur.
- **When:** The user reads or saves.
- **Then:** Unknown never becomes zero and success follows readback; unknown outcomes are reconciled by command ID before another creation. Conflicts retain input and offer comparison. Session expiry leads to sign-in, bank reauth to the proper owner, and AI waiting does not block ordinary accounting.
- **Level:** `manual+e2e`.

#### AC-100

- **Given:** A member opens /login in either language.
- **When:** Passkey sign-in is waiting, cancelled or failed.
- **Then:** Compact centered logo/Want Keep, whitespace, one primary “Войти с passkey” / “Log in with Passkeys” button, subtle language/help. Status preserves composition; help leads to personal recovery. SVG proportions remain intact.
- **Level:** `manual+e2e`.

#### AC-101

- **Given:** Screens and forms are available in RU/EN on macOS.
- **When:** 1280×720 and 1440×900 CSS px, 100%/200% zoom, keyboard and reduced motion are tested.
- **Then:** No hidden actions, clipped amounts or horizontally scrolling forms; tables have a labelled scroll region when needed. Normal text contrast ≥4.5:1, large text ≥3:1, meaningful boundaries/focus ≥3:1. Focus is visible and restored; states work without color. Actual Chrome/Arc/macOS versions are recorded separately from Chromium CI.
- **Level:** `manual+e2e`.

#### AC-102

- **Given:** Both members receive synthetic data and tasks without interface hints.
- **When:** They identify allowance/shortfall, explain an overview amount, enter a receipt, change shares, reserve for a goal, handle reauth/AI waiting, and sign in/recover.
- **Then:** Each completes all seven flows and correctly explains outcomes and no double counting. Time, mistakes, difficulties and browser versions are recorded; financial misunderstanding or developer assistance requires a fix and repetition of the affected flow.
- **Level:** `manual+e2e`.

#### AC-103

- **Given:** A current account, credit card, deposit/Earn/Coinhold and crypto product have incomplete data.
- **When:** The user opens each SCR-008 variant.
- **Then:** Current account shows available/reserved/blocked, credit shows debt/next payment/grace, savings shows actual/forecast/maturity, crypto shows available assets/positions/fees. Unknown terms are labelled; borrowed funds are not presented as available household pool.
- **Level:** `manual+e2e`.

#### AC-104

- **Given:** Effects are enabled for adding an accounting account, confirmed savings top-up, goal achievement and limit breach; repeated sync and reduced motion occur.
- **When:** The user receives a confirmed event, revisits the page, disables effects or enables reduced motion.
- **Then:** Rocket/sparkles/confetti/soft disco follow design.en.md and event IDs, never replay from refresh/retry/backfill. Limit breach gets a calm actionable warning, not a reward. No flashes/strobe, blocked forms or hidden text; disabled/reduced motion uses a static acknowledgement. Financial figures never animate through false intermediate values.
- **Level:** `manual+e2e`.

### Verification

```sh
make e2e SCENARIO=desktop-ux
```

Automation report plus separate actual Chrome/Arc and both-user protocol; unavailable real browsers leave acceptance incomplete.

Make commands are verification interfaces; the existence of a target does not establish an implemented suite. If a required suite is missing, implement it as part of the task; a missing suite is not a passing check. Record the actual outcome of each run. make docs-check validates documentation and research tooling, not financial runtime, browsers or production. Record live/manual checks, access and costs separately.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
