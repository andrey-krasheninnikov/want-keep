<!-- want-keep-task: task-7.16 -->
# task-7.16 — Мигрировать Want Keep на want-desyst / Migrate Want Keep to want-desyst

## RU

Перевести существующий desktop UI на общую дизайн-систему с сохранением финансовых и авторизационных сценариев.

**Состояние:** Реализовано и проверено локально; итог review и публикации фиксируется в GitHub #97.

**Зависимости:** `task-7.1`, `task-7.10`, `task-7.11`.

**Тип:** `implementation`.

### Изменение и контракты

D-49: полная тёмная тема, Days One/Manrope, публичные компоненты foundation/react 0.2.1 и проектный want-design. Существующие оболочка, вход, приглашения, recovery, безопасность, семейная панель, наличные счета и dev-каталог используют одну реализацию примитивов. Сохраняются API, права, CSRF, неизвестный исход команды, точные суммы/даты, RU/EN, черновики, доступность и reduced motion. Upstream calendar fix: want-desyst #7 / PR #9. Карта компонентов и границы: design-components.md. Новые экраны, backend, БД, OpenAI, банки и production вне объёма.

### Границы изменений

- `web/`
- `spec/001-want-keep-mvp/`

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
make test-web FILTER=all
```

Web-проверки, build, docs-check, браузерные design/access-сценарии и целевая приёмка Chrome/Arc подтверждены отдельно.

Проверки миграции не подтверждают приёмку всех экранов, банков или production. Полный журнал результата: evidence/issue-97-want-desyst.md.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Move the existing desktop UI to the shared design system while preserving financial and authentication flows.

**Status:** Implemented and verified locally; review and publication outcomes are recorded in GitHub #97.

**Dependencies:** `task-7.1`, `task-7.10`, `task-7.11`.

**Kind:** `implementation`.

### Change and contracts

D-49: complete dark theme, Days One/Manrope, public foundation/react 0.2.1 components and project-local want-design. Existing shell, sign-in, invitations, recovery, security, household panel, cash accounts and development catalog use one primitive implementation. Preserve APIs, authority, CSRF, unknown command outcomes, exact money/dates, RU/EN, drafts, accessibility and reduced motion. Upstream calendar fix: want-desyst #7 / PR #9. Component map and boundaries: design-components.en.md. New screens, backend, database, OpenAI, banking and production are excluded.

### Change boundaries

- `web/`
- `spec/001-want-keep-mvp/`

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
make test-web FILTER=all
```

Web checks, build, docs-check, design/access browser scenarios and targeted Chrome/Arc acceptance are evidenced separately.

Migration checks do not establish acceptance of all screens, banking or production. Result evidence: evidence/issue-97-want-desyst.en.md.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
