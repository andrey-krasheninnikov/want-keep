# План реализации Want Keep MVP

[English](plan.en.md)

Статус: **Ready for development**. Дата gate: 2026-09-07. Основание: [task-0.10 evidence](evidence/task-0.10-readiness.md), решения D-37–D-43 и [полный backlog](backlog.md).

План решение-полный на уровне SDD. Он разрешает начинать задачи по графу зависимостей, но не объявляет приложение реализованным. Provider deployment, production и полный MVP имеют отдельные exit gates.

## Правила исполнения

1. Работать по одной карточке `task-*`, её targets и acceptance; сохранять стабильные REQ/AC/task ID.
2. Перед изменением проверить точный branch/base, незакоммиченные изменения, локальные AGENTS и контракт владельца слоя.
3. Не изобретать provider fields. Подтверждённый денежный эффект проводится один раз; неизвестное поле остаётся `unknown`, gap — `source_partial`. Только неподтверждённые identity, status или monetary effect и collision `source_ambiguous` не создают проводку.
4. Внешние интеграции только read-only. По D-43 provider deployment выключен до server-owned admission точного binding; `provider_not_admitted` не запускает collector.
5. После изменения запускать узкие проверки карточки, затем `make check`; внешний readback и CI фиксировать отдельно.
6. Мержить только reviewable diff с RU/EN и traceability при изменении контракта.

## Порядок и параллелизм

| Волна | Задачи | Entry | Параллелизм | Exit |
| --- | --- | --- | --- | --- |
| 0 — SDD | task-0.1–task-0.10 | Интервью и доступное research evidence | Завершено | Ready evidence и этот plan опубликованы |
| 1 — контракты ядра | task-1.1 → task-1.2 → task-1.3 → task-1.4 → task-1.6 → task-1.5 | task-1.1 уже завершена; следующая — task-1.2 | После task-1.4 task-1.6; task-1.5 ждёт обе | Money/API, storage/retention, passkey, family authz и secrets готовы |
| 2 — ledger | task-2.1 → task-2.2 → task-2.3; затем task-2.4/task-2.5/task-2.6; затем task-2.8 → task-2.7/task-2.9 | Storage и membership | 2.4, 2.5, 2.6 параллельно; 2.7 после 2.6/2.8, 2.9 после 2.4/2.8 | Точный журнал, audit, reconciliation, split/refund/debt без двойного эффекта |
| 3 — ingestion | task-3.1 → task-3.2 → task-3.3 | Storage, accounts, ledger, secrets | task-5.1 может стартовать после 3.1; FX после 3.2 | Durable jobs, normalized import contract и изолированный collector |
| 4 — providers | task-4.1–task-4.6 | 3.3, 2.4, 2.5 и research конкретного provider | Все шесть параллельно после общих deps | Contract fixtures + live readback + собственный deployment gate каждого provider |
| 5 — AI | task-5.1 → task-5.2 → task-5.3/task-5.4; task-5.5 после task-6.8 | OpenAI evidence, jobs, secrets и ledger | 5.3/5.4 по своим deps; AI не блокирует обычный учёт | Budget-safe gateway, review every operation, receipts/chat/clarifications, insights |
| 6 — finance | task-6.1; task-6.2; task-6.3 → task-6.4; task-6.5; task-6.6 → task-6.7 → task-6.8 | Ledger; для provider-derived функций — соответствующие task-4.x | 6.1/6.2/6.5 и подготовка 6.6 параллельно по deps | FX/XIRR/credit/savings/budget/goals/daily limits проходят contract tests |
| 7 — desktop | task-7.10 → task-7.11; task-7.16; task-7.1 → task-7.9; task-7.2–task-7.8, task-7.13/task-7.14/task-7.15; task-7.12 | API/auth contracts и соответствующие domain read models | Design system параллельно backend; feature screens после своих deps | SCR-001–SCR-035, RU/EN, Chrome/Arc, 1280×720/1440×900, accessibility и motion acceptance |
| 8 — operations | task-8.1 → task-8.2 → task-8.3 | Collector, AI, notifications; отдельная авторизация на ресурсы | Документацию/скрипты можно готовить раньше, provisioning только по разрешению | Hardened deployment, measured budget/load, encrypted Mac backup и restore rehearsal |
| 9 — acceptance | task-9.1 → task-9.2 | Все перечисленные deps и provider gates | Нет: frozen acceptance candidate | Все mandatory AC, independent final Avida, handoff evidence |

Граф `catalog.json` остаётся источником точных зависимостей. Диапазон в таблице не разрешает обходить зависимость отдельной карточки.

## Provider entry/deployment gates

| Provider task | Предпочтительный transport | До реализации mapping | До включения deployment |
| --- | --- | --- | --- |
| task-4.1 Alfa | Official structured read, иначе разрешённый Playwright | Read connector D-47/D-48; unknown source records без проводок | Финансовый mapping, live-проверки и deployment admission переданы task-4.7 / #94 |
| task-4.2 Raiffeisen | RBO API/CAMT | CAMT 1:N, canonical cross-report fingerprint + optional-ID aliases, CLBD/unknown semantics | OAuth rotation, corrections/reversals, full history, second account, live conformance |
| task-4.3 Ozon | Подтверждённый session read | Sanitized HAR projection, route namespaces, parent fee relation | Session permission/lifecycle, stable account ID, history end, second account, reauth |
| task-4.4 Bybit | Official RSA read-only API | Route IDs, candidate links, hourly collision policy | Precision/history, two accounts, key rotation/revocation, write-route denial |
| task-4.5 Aifory | Structured session read; Playwright только при необходимости | D-33 namespace/unknown rules | Permission, structured fixtures, card lifecycle/fees/FX, pagination, reauth, two accounts |
| task-4.6 EMCD | Structured session read; Playwright только при необходимости | D-34 namespace/unknown rules | Fixtures wallet/Grow/card/P2P, balance/lifecycle/fees, pagination, reauth, two accounts |

Провал provider gate сохраняет источник отключённым. `backend/internal/connections/admission/` и storage adapter task-1.3 владеют aggregate, monotonic `admissionRevision`, атомарным combine/invalidate, admission-check + enqueue и commit-time result fence. task-4.x provider evidence и task-8.x host evidence объединяет только admission service для точного D-43 binding; stale binding закрывает новый sync, а stale result остаётся в quarantine без source/posting. Cancel начатого read — best effort. Это не блокирует ручной учёт, доменные функции или проверенные другие источники. Write actions не входят ни в один fallback.

## Контрактные entry/exit gates

### task-1.2

Entry: task-1.1 в target; contracts version 10. Exit: Money/Asset/Rate/coverage, generated OpenAPI boundary, errors `source_partial`, `source_ambiguous`, `valuation_unavailable`, `quote_unavailable`, `command_expired`, `provider_not_admitted`; connection `deploymentGate.status` и binding; доменная policy запрещает sync до admission, пользовательские DTO не позволяют назначить admission; D-41 recent/detail/tombstone semantics покрыты тестами. Реальное атомарное применение перед job/IO проверяют task-3.3/task-4.x/task-8.1.

### task-1.3

Entry: versioned API/value objects task-1.2. Exit: atomic source/posting/revision/outbox; D-39 unique key and collision evidence; D-41 independent cleanup jobs; persisted D-43 aggregate/repository, monotonic admission revision, atomic provider+host combine/invalidate, admission-check + job enqueue and commit-time stale-result quarantine; restart/revoke-before-and-after-IO/binding-race tests on isolated PostgreSQL.

### task-6.1 и task-6.4

FX exit: CBR/Frankfurter/CoinGecko rules, ≤365-day coverage, `valuation_unavailable`, quote separation, quota/attribution runtime check. XIRR exit: Actual/365, same-day aggregation, одна смена знака, fractional power в 50-digit HALF_EVEN decimal с error bound `1e-24`, bisection от `-1 + 1e-12` до `1 000 000`, irregular/boundary vectors.

### task-9.1/task-9.2

Full MVP exit requires all linked ACs, six admitted provider connectors, retry/duplicate/revision scenarios, family authz, AI injection/cost failures, RU/EN desktop UX, Chrome/Arc, notifications, target deployment and backup/restore. task-9.2 freezes the candidate and leaves no confirmed P0–P3.

## Команды проверки

Базовый набор каждой задачи:

```sh
make docs-check
make check
git diff --check
```

Карточка добавляет целевую команду: `make test-go`, `make test-contract`, `make test-integration`, `make test-web`, `make test-collector`, `make e2e`, `make eval-ai`, `make check-deploy`, `make backup-check` или `make restore-check`. Fail-fast отсутствующей suite — ожидаемый результат до её реализации, не pass.

Проверки разделяются в отчёте: local/synthetic, CI, authorized provider readback, real Chrome/Arc, target-host/runtime и backup/restore. Одно доказательство не подменяет другое.

## Контроль полного охвата

Все 67 задач перечислены явно; порядок определяет граф зависимостей выше:

- `task-0.1` — Проверить контракт чтения Альфа-Банк
- `task-0.2` — Проверить контракт чтения Райффайзенбанк РФ
- `task-0.3` — Проверить контракт чтения Ozon Банк
- `task-0.4` — Проверить контракт чтения Bybit
- `task-0.5` — Проверить контракт чтения Aifory Pro
- `task-0.6` — Проверить контракт чтения EMCD
- `task-0.7` — Проверить бесплатные источники курсов
- `task-0.8` — Измерить качество и стоимость OpenAI
- `task-0.9` — Проверить инфраструктуру и бюджет сервера
- `task-0.10` — Закрыть контракты и проверить готовность SDD
- `task-1.1` — Создать структуру проекта и команды проверки
- `task-1.2` — Определить денежные типы и API-контракт
- `task-1.3` — Создать хранилище и транзакционные границы
- `task-1.4` — Реализовать passkey и восстановление доступа
- `task-1.5` — Защитить секреты и приватные вложения
- `task-1.6` — Создать семью, членство и права на ресурсы
- `task-2.1` — Реализовать счета и начальные остатки
- `task-2.2` — Реализовать журнал операций и статусы
- `task-2.3` — Добавить версии, исправления и аудит
- `task-2.4` — Связать переводы и исключить дубликаты
- `task-2.5` — Сверять журнал с балансом источника
- `task-2.6` — Разделить категории, продавцов и товары
- `task-2.7` — Реализовать возвраты и распределение расходов
- `task-2.8` — Распределять семейные расходы и позиции по участникам
- `task-2.9` — Учитывать явные долги и возмещения внутри семьи
- `task-3.1` — Создать долговечные фоновые задания
- `task-3.2` — Определить входной контракт коннекторов
- `task-3.3` — Создать изолированный браузерный сборщик
- `task-4.1` — Реализовать коннектор Альфа-Банк
- `task-4.2` — Реализовать коннектор Райффайзенбанк РФ
- `task-4.3` — Реализовать коннектор Ozon Банк
- `task-4.4` — Реализовать коннектор Bybit
- `task-4.5` — Реализовать коннектор Aifory Pro
- `task-4.6` — Реализовать коннектор EMCD
- `task-4.7` — Подтвердить финансовый mapping и live-допуск Alfa (#94); зависит от read-коннектора task-4.1, блокирует production admission и полную приёмку task-9.1.
- `task-5.1` — Создать OpenAI gateway и контроль расходов
- `task-5.2` — Проверять каждую операцию через AI-команды
- `task-5.3` — Обрабатывать чеки и позиции
- `task-5.4` — Реализовать чат и очередь уточнений
- `task-5.5` — Формировать обоснованные AI-инсайты
- `task-6.1` — Реализовать курсы и валютную оценку
- `task-6.2` — Учитывать кредитки и грейс-период
- `task-6.3` — Считать начисления и прогноз накоплений
- `task-6.4` — Сравнивать доходность денежных потоков
- `task-6.5` — Сводить торговый результат и майнинг
- `task-6.6` — Планировать месячный бюджет и доходы
- `task-6.7` — Резервировать деньги на цели
- `task-6.8` — Считать дневные лимиты и прогноз ликвидности
- `task-7.1` — Создать desktop-оболочку и вход RU/EN
- `task-7.2` — Показать счета, операции и исправления
- `task-7.3` — Создать чат с выбором счёта и файлами
- `task-7.4` — Создать редактор месячного бюджета
- `task-7.5` — Показать цели и резервирование
- `task-7.6` — Собрать дашборд, лимиты и инсайты
- `task-7.7` — Показать кредитки, накопления и доходность
- `task-7.8` — Добавить уведомления и web-push
- `task-7.9` — Добавить семейный контекст и принадлежность в интерфейс
- `task-7.10` — Создать токены, типографику и брендовые ресурсы
- `task-7.11` — Создать каталог компонентов на Base UI
- `task-7.12` — Проверить desktop UX и визуальную приёмку
- `task-7.13` — Создать экраны подключений и повторного входа
- `task-7.14` — Создать настройки и состояние учёта
- `task-7.15` — Добавить контекстные анимации финансовых событий
- `task-8.1` — Подготовить развёртывание и состояние системы
- `task-8.2` — Выгружать зашифрованные копии на MacBook
- `task-8.3` — Проверить восстановление из локальной копии
- `task-9.1` — Провести сквозную приёмку полного MVP
- `task-9.2` — Провести итоговый Avida review и передать MVP
