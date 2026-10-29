<!-- want-keep-task: task-7.11 -->
# task-7.11 — Поддерживать каталог общих компонентов Want / Maintain the shared Want component catalog

## RU

Доступные компоненты с собственным оформлением.

**Состояние:** Реализованы компоненты и dev/test-каталог; зависимость task-7.10 закрыта. Проверки и границы приёмки: evidence/task-7.11-components.md.

**Зависимости:** `task-7.10`.

**Тип:** `implementation`.

### Изменение и контракты

Текущие примитивы поставляет @want-foundation/desyst-react 0.2.1 на Radix. Want Keep сохраняет только продуктовые композиции и адаптацию публичных props: формы, денежные строки, черновики дат, навигацию, keyset-pagination и уведомления. Дата остаётся YYYY-MM-DD в диапазоне 0001–9999. Каталог /__design/components и UISTATE-01–17 остаются синтетическими и исключаются из production. Историческая реализация Base UI завершена ранее; миграция отслеживается отдельно в task-7.16 / #97.

### Границы изменений

- `web/src/design-system/`
- `web/src/design-system/catalog/`

Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-077:** Полная тёмная тема want-desyst используется без локальной замены палитры, геометрии или фокуса.
- **REQ-078:** Days One используется для display, Manrope для повседневного интерфейса; шрифты поставляет want-desyst.
- **REQ-079:** Публичные компоненты want-desyst реализуют примитивы; Want Keep владеет продуктовыми композициями и состоянием.
- **REQ-082:** Экранные состояния объясняют последствия и безопасный следующий шаг без потери ввода.
- **REQ-084:** Доступность проверяется на реальных Chrome и Arc, включая клавиатуру, фокус, контраст, масштаб и reduced motion.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-094

- **Дано:** Обзор, вход, форма и таблица используют одну дизайн-систему.
- **Когда:** Проверяются фон, поверхности, акценты и состояния.
- **Тогда:** Фон #141414, поверхности #1C1C1C/#252525 и акцент #5962F0 берутся из закреплённого foundation. Статусы, радиусы, клавиатурный фокус и motion следуют общей теме. data-theme=dark и dark установлены до первого кадра. Название и логотип Want Keep сохраняются; светлой темы нет.
- **Уровень:** `manual+e2e`.

#### AC-095

- **Дано:** Есть RU/EN строки, RUB/USD/USDT/USDC/BTC/ETH и длинные точные суммы.
- **Когда:** Проверяются загруженные и недоступные шрифты.
- **Тогда:** Шрифты foundation обслуживаются локально с лицензиями. Кириллица, валютные символы и fallback остаются читаемыми. Таблицы и сопоставляемые суммы используют Manrope с табличными цифрами. Pixelify Sans и прежние локальные font-face удалены; суммы остаются точными строками.
- **Уровень:** `manual+e2e`.

#### AC-096

- **Дано:** Каталог содержит кнопки, поля, диалоги, панели, таблицы и состояния данных.
- **Когда:** Проверяются состояния и реализация примитивов.
- **Тогда:** Foundation/react закреплены одной версией 0.2.1 с lockfile. Используются публичные exports и собранные CSS/шрифты. Base UI/shadcn и дубли примитивов удалены. Сохранены клавиатура, подписи, ref, controlled state, порталы, черновики, ошибки и восстановление фокуса. Финансовые правила и API не входят в общую библиотеку.
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

### Проверка результата

```sh
make test-web FILTER=design-components
```

Сохраняются клавиатура, подписи, ошибки и возврат фокуса; тема и взаимодействие следуют want-desyst.

Команды существуют: make check; make test-web FILTER=design-components; make e2e SCENARIO=design-components; make e2e SCENARIO=design-tokens; git diff --check. Контракты: design-components.md; результаты и границы: evidence/task-7.11-components.md. По решению пользователя реальная браузерная проверка этой задачи использует Chrome; Arc остаётся последующей приёмке.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Accessible components with custom styling.

**Status:** Components and the dev/test catalog are implemented; task-7.10 is complete. Checks and acceptance limits: evidence/task-7.11-components.en.md.

**Dependencies:** `task-7.10`.

**Kind:** `implementation`.

### Change and contracts

Current primitives come from Radix-based @want-foundation/desyst-react 0.2.1. Want Keep retains only product compositions and public-prop adaptations: forms, monetary strings, date drafts, navigation, keyset pagination and feedback. Dates remain YYYY-MM-DD within years 0001–9999. /__design/components and UISTATE-01–17 remain synthetic and excluded from production. The historical Base UI implementation was completed earlier; task-7.16 / #97 tracks migration separately.

### Change boundaries

- `web/src/design-system/`
- `web/src/design-system/catalog/`

Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-077:** Use the complete want-desyst dark theme without local palette, geometry or focus replacements.
- **REQ-078:** Use Days One for display and Manrope for everyday UI, supplied by want-desyst.
- **REQ-079:** Public want-desyst components implement primitives; Want Keep owns product compositions and state.
- **REQ-082:** Screen states explain consequences and a safe next step without losing input.
- **REQ-084:** Accessibility is checked in actual Chrome and Arc, including keyboard, focus, contrast, zoom and reduced motion.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-094

- **Given:** Overview, sign-in, form and table use one design system.
- **When:** Backgrounds, surfaces, accents and states are inspected.
- **Then:** Background #141414, surfaces #1C1C1C/#252525 and primary #5962F0 come from the pinned foundation. Status roles, radii, keyboard focus and motion follow the shared theme. Set data-theme=dark and dark before first paint. Keep the Want Keep name and logo; no light theme.
- **Level:** `manual+e2e`.

#### AC-095

- **Given:** RU/EN strings, RUB/USD/USDT/USDC/BTC/ETH and long precise amounts exist.
- **When:** Loaded and unavailable fonts are tested.
- **Then:** Foundation fonts are served locally with their licenses. Cyrillic, currency glyphs and fallback remain legible. Tables and comparable amounts use Manrope tabular figures. Remove Pixelify Sans and the previous local font-face definitions; monetary values remain exact strings.
- **Level:** `manual+e2e`.

#### AC-096

- **Given:** A catalog contains buttons, fields, dialogs, panels, tables and data states.
- **When:** States and primitive implementations are inspected.
- **Then:** Pin foundation/react to matching 0.2.1 with a lockfile. Use public exports and compiled CSS/fonts. Remove Base UI/shadcn and duplicate primitives. Preserve keyboard access, labels, refs, controlled state, portals, drafts, errors and focus restoration. Shared packages contain no financial rules or product APIs.
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

### Verification

```sh
make test-web FILTER=design-components
```

Keyboard access, labels, errors and focus restoration remain intact; theme and interaction follow want-desyst.

Commands exist: make check; make test-web FILTER=design-components; make e2e SCENARIO=design-components; make e2e SCENARIO=design-tokens; git diff --check. Contracts: design-components.en.md; results and limits: evidence/task-7.11-components.en.md. Per user decision, actual-browser verification for this task uses Chrome; Arc remains later acceptance.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
