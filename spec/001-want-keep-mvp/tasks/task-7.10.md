<!-- want-keep-task: task-7.10 -->
# task-7.10 — Создать токены, типографику и брендовые ресурсы / Create tokens, typography and brand assets

## RU

Единый визуальный фундамент desktop Want Keep.

**Состояние:** Реализована визуальная основа и dev/test-образцы; task-1.1 включена в базу. Продуктовые экраны и общий каталог компонентов остаются последующим задачам.

**Зависимости:** `task-1.1`.

**Тип:** `implementation`.

### Изменение и контракты

Историческая основа task-7.10 сохранена. D-49 и issue #97 переводят текущую тему на опубликованные foundation/react 0.2.1 целиком, включая Days One, Manrope, цвета, фокус и motion. Продуктовые layout-алиасы ссылаются на общие роли, не заменяя их. Логотип, точные строки сумм, табличные цифры, reduced motion и исключение dev-каталога из production сохраняются. Текущий контракт: design.md; прежние доказательства остаются историческими.

### Границы изменений

- `web/src/design-system/`
- `web/public/fonts/`
- `web/public/brand/`

Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-077:** Полная тёмная тема want-desyst используется без локальной замены палитры, геометрии или фокуса.
- **REQ-078:** Days One используется для display, Manrope для повседневного интерфейса; шрифты поставляет want-desyst.

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

### Проверка результата

```sh
make test-web FILTER=design-tokens
```

Токены воспроизводимы, шрифты RU/EN и длинные суммы читаемы, лицензии включены, contrast matrix приложена.

Команды существуют: make check; make test-web FILTER=design-tokens; make e2e SCENARIO=design-tokens; git diff --check. Матрица: design-contrast.md; покрытие шрифтов и границы: evidence/task-7.10-design.md. По уточнению пользователя ручная проверка этой задачи использует Chrome; Arc остаётся последующей продуктовой приёмке.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

A consistent desktop Want Keep visual foundation.

**Status:** Visual foundation and dev/test specimens implemented; task-1.1 is included in the base. Product screens and the full component catalog remain downstream.

**Dependencies:** `task-1.1`.

**Kind:** `implementation`.

### Change and contracts

Retain the historical task-7.10 foundation. D-49 and issue #97 move the current theme entirely to published foundation/react 0.2.1, including Days One, Manrope, colors, focus and motion. Product layout aliases reference shared roles without replacing them. Preserve the logo, exact monetary strings, tabular figures, reduced motion and production exclusion of the development catalog. Current contract: design.en.md; earlier evidence remains historical.

### Change boundaries

- `web/src/design-system/`
- `web/public/fonts/`
- `web/public/brand/`

Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-077:** Use the complete want-desyst dark theme without local palette, geometry or focus replacements.
- **REQ-078:** Use Days One for display and Manrope for everyday UI, supplied by want-desyst.

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

### Verification

```sh
make test-web FILTER=design-tokens
```

Tokens are reproducible, RU/EN fonts and long amounts are readable, licenses included and contrast matrix attached.

Commands exist: make check; make test-web FILTER=design-tokens; make e2e SCENARIO=design-tokens; git diff --check. Matrix: design-contrast.md; font coverage and boundaries: evidence/task-7.10-design.en.md. Per user refinement this task uses Chrome for manual verification; Arc remains later product acceptance.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
