<!-- want-keep-task: task-8.2 -->
# task-8.2 — Выгружать зашифрованные копии на MacBook / Pull encrypted backups to the MacBook

## RU

Получать согласованные независимые от сервера копии при доступном Mac.

**Состояние:** Открыта; ready-for-agent. Описание готово к реализации; начало зависит от незакрытых задач и условий ниже. Метка не подтверждает выполнение, live-допуск или разрешение на production; SDD Ready не является блокером.

**Зависимости:** `task-8.1`, `task-1.5`.

**Тип:** `implementation`.

### Изменение и контракты

### Текущее поведение

Согласованы исследовательские решения по инфраструктуре и резервированию; доступны модульные контракты хранения и фоновой обработки. Развёртывание, получение копии и измеренное восстановление требуют собственных доказательств.

### Требуемое поведение

Реализовать через launchd инициируемый Mac hourly pull без входящего порта: restricted non-root export principal, согласованный cutoff, потоковый logical pg_dump managed PostgreSQL через VPS/private VPC, immutable attachment inventory и manifest с checksum/version/timestamps. Шифровать на стороне получателя; recovery key хранить в Mac Keychain с независимой аварийной копией. Набор завершать атомарно после проверки. Retention: 48 hourly/30 daily/8 weekly/12 monthly; cap 20 GiB, warning 15 GiB или <25 GiB свободно, последний complete set не удалять. Проверять volume encryption/capacity и показывать fresh/stale/failed, age и условный RPO; документы RU/EN.

### Ключевые контракты

Mac launchd pull, restricted export principal, согласованный cutoff, encrypted backup manifest/inventory и retention.

### Вне объёма

Не менять production, не создавать платные ресурсы и не удалять пользовательские данные без отдельной действующей авторизации. Не считать синтетический прогон доказательством live-допуска.

### Блокирующие зависимости

- [#63](https://github.com/andrey-krasheninnikov/want-keep/issues/63) (`task-8.1`): открыта; блокирует выполнение.
- [#15](https://github.com/andrey-krasheninnikov/want-keep/issues/15) (`task-1.5`): выполнена.

### Границы изменений

- `ops/backup/`
- `backend/internal/backup/`
- `docs/operations/backup.md`

Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-050:** Файлы, ключи источников, сессии и финансовые журналы защищены от постороннего доступа.
- **REQ-057:** Зашифрованная резервная копия выгружается на MacBook ежечасно при его доступности; восстановление проверяется.
- **REQ-058:** Операционные статусы показывают ошибки импорта, AI, курсов, резервирования и расходы без утечки финансового содержимого.
- **REQ-061:** Повторные задания, перезапуски и параллельные изменения не создают двойных финансовых эффектов.
- **REQ-075:** Восстановление данных сохраняет пользователей, членство, принадлежность, роли, историю и общий семейный учёт.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-050

- **Дано:** Существует приватный чек и активное подключение источника.
- **Когда:** Проверяются прямой URL файла, экспорт без сессии, логи и отзыв подключения.
- **Тогда:** Без авторизации доступ закрыт; секреты зашифрованы и не журналируются; отзыв подключения прекращает дальнейший сбор.
- **Уровень:** `integration`.

#### AC-057

- **Дано:** Есть база, вложения и MacBook, который временно недоступен.
- **Когда:** Создаются копии, Mac возвращается в сеть, затем проводится восстановление.
- **Тогда:** Показан возраст последней полной копии; после возвращения копирование возобновляется; восстановлены согласованные данные и вложения до четырёх часов; часовой RPO заявляется только при доступном Mac.
- **Уровень:** `integration+manual`.

#### AC-058

- **Дано:** Сломан один коннектор, задержан AI и устарела копия.
- **Когда:** Открывается состояние системы и читаются диагностические логи.
- **Тогда:** Видны отдельные проблемы и действия восстановления; логи содержат идентификаторы/коды, а не чеки, ключи или тексты финансовых сообщений.
- **Уровень:** `integration`.

#### AC-061

- **Дано:** Процесс падает между сохранением записи и подтверждением задания.
- **Когда:** Задание повторяется, одновременно приходит правка владельца.
- **Тогда:** Применён один эффект, правка защищена версией, незавершённое состояние восстанавливается; внешняя неоднозначность не вызывает слепой повтор.
- **Уровень:** `integration`.

#### AC-073

- **Дано:** MacBook недоступен 10 часов; последующая загрузка вложений обрывается.
- **Когда:** Проверяется статус копии и выполняется восстановление последнего полного набора.
- **Тогда:** Не заявляется часовой RPO; неполный набор не помечен успешным; последний полный набор восстановим и имеет проверяемый manifest.
- **Уровень:** `integration+manual`.

#### AC-089

- **Дано:** Копия содержит двух участников, личные/общие цели и операции с разными авторами.
- **Когда:** Полный набор восстанавливается на изолированном сервере.
- **Тогда:** Суммы, связи и права обоих сохранены; входы не объединены, банковские сессии автоматически не оживают. Восстановление укладывается в измеренную цель RTO.
- **Уровень:** `integration+manual`.

### Проверка результата

```sh
make test-integration AREA=backup && make backup-check MODE=synthetic
```

Недоступный Mac, оборванный download и повреждённый manifest не дают ложный successful backup; полноценный набор проверяется.

Команды Make являются интерфейсами проверки; наличие target не доказывает реализацию соответствующей suite. Если требуемой suite ещё нет, реализовать её в рамках задачи; отсутствие suite не считается успешной проверкой. Фиксировать фактический результат каждого прогона. make docs-check проверяет документацию и исследовательские инструменты, но не финансовый runtime, браузеры или production. Live/manual-проверки, доступ и расходы фиксируются отдельно.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Obtain consistent server-independent copies while the Mac is reachable.

**Status:** Open; ready-for-agent. The description is ready for implementation; open prerequisites and the entry conditions below govern execution. The label does not establish completion, live admission or production authorization; SDD Ready is not a blocker.

**Dependencies:** `task-8.1`, `task-1.5`.

**Kind:** `implementation`.

### Change and contracts

### Current behavior

Infrastructure and backup research decisions are recorded; storage and background-processing contracts are available. Deployment, backup collection and measured recovery require their own evidence.

### Desired behavior

Implement a launchd-driven hourly Mac pull with no inbound Mac port: restricted non-root export principal, consistent cutoff, streaming logical pg_dump of managed PostgreSQL through VPS/private VPC, immutable attachment inventory and checksum/version/timestamp manifest. Encrypt on the recipient; keep the recovery key in Mac Keychain with an independent emergency copy. Complete a set atomically after verification. Retention: 48 hourly/30 daily/8 weekly/12 monthly; 20 GiB cap, warning at 15 GiB or <25 GiB free, never delete the last complete set. Preflight volume encryption/capacity and show fresh/stale/failed, age and conditional RPO; document RU/EN.

### Key interfaces

Mac launchd pull, restricted export principal, consistent cutoff, encrypted backup manifest/inventory and retention.

### Out of scope

Do not change production, provision paid resources or delete user data without separate current authorization. A synthetic run does not establish live admission.

### Blocked by

- [#63](https://github.com/andrey-krasheninnikov/want-keep/issues/63) (`task-8.1`): open; blocks execution.
- [#15](https://github.com/andrey-krasheninnikov/want-keep/issues/15) (`task-1.5`): completed.

### Change boundaries

- `ops/backup/`
- `backend/internal/backup/`
- `docs/operations/backup.md`

Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-050:** Files, source keys, sessions and financial records are protected against unauthorized access.
- **REQ-057:** An encrypted backup is pulled to the MacBook hourly while reachable; recovery is tested.
- **REQ-058:** Operational status exposes import, AI, FX, backup failures and spend without leaking financial content.
- **REQ-061:** Repeated jobs, restarts and concurrent changes cannot create duplicate financial effects.
- **REQ-075:** Data recovery preserves users, memberships, ownership, roles, history and shared household accounting.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-050

- **Given:** A private receipt and an active source connection exist.
- **When:** A direct file URL, unauthenticated export, logs and disconnection are checked.
- **Then:** Unauthenticated access fails; secrets are encrypted and not logged; disconnecting stops further collection.
- **Level:** `integration`.

#### AC-057

- **Given:** The database, attachments and a temporarily unreachable MacBook exist.
- **When:** Backups are attempted, the Mac reconnects and recovery is rehearsed.
- **Then:** Last complete backup age is visible; copying resumes after reconnection; consistent data and attachments restore within four hours; hourly RPO is claimed only while the Mac is reachable.
- **Level:** `integration+manual`.

#### AC-058

- **Given:** A connector fails, AI is delayed and a backup is stale.
- **When:** System health and diagnostic logs are inspected.
- **Then:** Separate failures and recovery actions are visible; logs contain identifiers/codes, not receipts, keys or financial message text.
- **Level:** `integration`.

#### AC-061

- **Given:** A process crashes between persisting a record and acknowledging its job.
- **When:** The job is retried while the owner submits a correction.
- **Then:** One effect is applied, the correction is version-protected and incomplete state recovers; an ambiguous external outcome is not blindly retried.
- **Level:** `integration`.

#### AC-073

- **Given:** The MacBook is unreachable for 10 hours and the subsequent attachment download is interrupted.
- **When:** Backup status is inspected and the last complete set is restored.
- **Then:** Hourly RPO is not claimed; the partial set is not marked successful; the last complete set is recoverable with a verifiable manifest.
- **Level:** `integration+manual`.

#### AC-089

- **Given:** A backup contains two members, personal/joint goals and transactions by different authors.
- **When:** The complete set is restored onto an isolated server.
- **Then:** Amounts, relationships and both users’ permissions are preserved; sign-ins are not merged and bank sessions do not revive automatically. Recovery meets the measured RTO target.
- **Level:** `integration+manual`.

### Verification

```sh
make test-integration AREA=backup && make backup-check MODE=synthetic
```

Unreachable Mac, interrupted download and corrupt manifest never produce false success; complete sets verify.

Make commands are verification interfaces; the existence of a target does not establish an implemented suite. If a required suite is missing, implement it as part of the task; a missing suite is not a passing check. Record the actual outcome of each run. make docs-check validates documentation and research tooling, not financial runtime, browsers or production. Record live/manual checks, access and costs separately.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
