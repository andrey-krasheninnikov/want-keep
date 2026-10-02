<!-- want-keep-task: task-9.2 -->
# task-9.2 — Провести итоговую независимую проверку и передать MVP / Run final independent review and hand off the MVP

## RU

Закрыть подтверждённые дефекты и передать проверяемый результат.

**Состояние:** Не начато; задача ожидает собственные зависимости и entry gates.

**Зависимости:** `task-9.1`.

**Тип:** `verification`.

### Изменение и контракты

Провести независимую read-only проверку полного зафиксированного кандидата по нескольким ролям с перепроверкой фактов. В рамках SDD исправить подтверждённые замечания, повторить затронутые проверки и финальную проверку по ограниченному бюджету раундов. Сопоставить REQ/AC/task/evidence и фактическую готовность; коммиты, push и deployment выполнять только при отдельной действующей авторизации.

### Границы изменений

- `spec/001-want-keep-mvp/verification.md`
- `spec/001-want-keep-mvp/evidence/release.md`

Пути планируемые. Общие контракты — `spec/001-want-keep-mvp/contracts.md`, архитектура/команды — `constraints.md`. Менять владельца поведения и его тесты; незакрытый контракт останавливает зависимую работу.

### Связанные требования

- **REQ-008:** Повторные импорты, чек и запись чата объединяют доказательства одной операции без повторного учёта.
- **REQ-012:** Исправление учёта сохраняет оригинал, автора, основание, версию и возможность отмены решения.
- **REQ-020:** AI-инсайты по доходам и расходам ссылаются на проверяемые данные и отделяют прогноз от факта.
- **REQ-029:** Одни средства нельзя одновременно зарезервировать на несколько целей или повторно учесть через выделенный счёт.
- **REQ-048:** Интеграции и браузерный сборщик выполняют только разрешённые операции чтения.
- **REQ-050:** Файлы, ключи источников, сессии и финансовые журналы защищены от постороннего доступа.
- **REQ-051:** AI ограничен бюджетом $50/месяц и деградирует в очередь ожидания без остановки обычного учёта.
- **REQ-055:** Веб-приложение предназначено для ноутбука macOS в Chrome и Arc; изменение окна и масштаба сохраняет доступность ежедневного учёта.
- **REQ-056:** Развёртывание укладывается в $40/месяц на сервер в DE/NL/BG; отдельные платные источники не используются.
- **REQ-057:** Зашифрованная резервная копия выгружается на MacBook ежечасно при его доступности; восстановление проверяется.
- **REQ-059:** Денежные расчёты используют точную арифметику и явные правила округления на границах.
- **REQ-060:** Текст чеков, банковских описаний и ответов AI не может расширять полномочия агента.
- **REQ-061:** Повторные задания, перезапуски и параллельные изменения не создают двойных финансовых эффектов.
- **REQ-062:** Архитектура использует Go/PostgreSQL, React/TypeScript/Vite и отдельный Playwright-сборщик с зависимостями к домену.
- **REQ-063:** Пользователь, семья и членство моделируются отдельно; ограничение двух участников задаётся конфигурацией.
- **REQ-064:** Оба участника видят все финансовые данные и изменяют операции; личные цели и части плана изменяет только их владелец.
- **REQ-065:** Принадлежность счёта, владелец внешнего аккаунта, автор записи и принадлежность расхода являются отдельными признаками.
- **REQ-067:** Расходы и позиции чеков имеют личное или совместное назначение; общая доля по умолчанию 50/50 с исключениями статьи или покупки.
- **REQ-069:** Резервы личных и совместных целей задаются явно; совместные цели отображаются отдельным общим блоком без персональных долей.
- **REQ-070:** Сумма индивидуальных дневных лимитов не превышает семейный предел одной валюты; счёт плательщика не меняет долю расходов.
- **REQ-071:** Один общий чат сохраняет автора сообщения и проверяет полномочия инициатора AI-команды при исполнении.
- **REQ-072:** Конкурирующие изменения, ответы на уточнения и отмены проверяют версию и текущие права, сохраняя обоих авторов.
- **REQ-073:** Оба управляют подключениями; банковскую авторизацию выполняет владелец внешнего аккаунта без раскрытия секретов партнёру или AI.
- **REQ-076:** Семейная область проверяется для API, файлов, AI, фоновых задач и внешних ID независимо от присланных actor/owner.
- **REQ-088:** Синхронизация провайдера разрешена только актуальным server-side admission, связанным с проверенными версиями адаптера, контракта, allowlist, конфигурации, разрешения оператора и окружения.

### Критерии приёмки

Связь задаёт покрытие, но не доказывает весь критерий; точный результат проверяется ниже.

#### AC-012

- **Дано:** AI ошибочно связал две операции; исходные импортированные записи сохранены.
- **Когда:** Владелец отменяет связь и исправляет категорию.
- **Тогда:** Пересчитаны производные отчёты; видна история; повторный импорт не стирает правку владельца.
- **Уровень:** `integration`.

#### AC-048

- **Дано:** Сборщик имеет сессию личного кабинета с более широкими внешними правами.
- **Когда:** Возникают запрос на платёж, неподтверждённый маршрут или MFA/CAPTCHA.
- **Тогда:** Платёж и неизвестный маршрут блокируются; MFA/CAPTCHA передаётся владельцу, источник приостанавливается; остальные источники продолжают работать.
- **Уровень:** `integration`.

#### AC-050

- **Дано:** Существует приватный чек и активное подключение источника.
- **Когда:** Проверяются прямой URL файла, экспорт без сессии, логи и отзыв подключения.
- **Тогда:** Без авторизации доступ закрыт; секреты зашифрованы и не журналируются; отзыв подключения прекращает дальнейший сбор.
- **Уровень:** `integration`.

#### AC-051

- **Дано:** OpenAI недоступен либо израсходован разрешённый бюджет с резервами текущих запросов.
- **Когда:** Поступают новый импорт, ручной расход и запрос AI.
- **Тогда:** Учёт и расчёты доступны; статус AI ожидает; новые платные запросы не запускаются сверх разрешённого резерва; неизвестная стоимость не освобождается молча.
- **Уровень:** `integration`.

#### AC-057

- **Дано:** Есть база, вложения и MacBook, который временно недоступен.
- **Когда:** Создаются копии, Mac возвращается в сеть, затем проводится восстановление.
- **Тогда:** Показан возраст последней полной копии; после возвращения копирование возобновляется; восстановлены согласованные данные и вложения до четырёх часов; часовой RPO заявляется только при доступном Mac.
- **Уровень:** `integration+manual`.

#### AC-059

- **Дано:** Есть дробные BTC, USDT, процентное начисление и распределение чека.
- **Когда:** Данные проходят API, базу и повторный расчёт.
- **Тогда:** Исходная точность не теряется; JSON-суммы не проходят binary float; распределения сходятся точно, округление отображения не меняет журнал.
- **Уровень:** `unit+contract`.

#### AC-060

- **Дано:** В PDF или описании операции есть инструкция раскрыть ключ либо сделать перевод.
- **Когда:** Документ обрабатывается AI.
- **Тогда:** Инструкция считается данными; секреты и платёжные инструменты недоступны; недопустимая команда отклонена и не меняет учёт.
- **Уровень:** `integration`.

#### AC-061

- **Дано:** Процесс падает между сохранением записи и подтверждением задания.
- **Когда:** Задание повторяется, одновременно приходит правка владельца.
- **Тогда:** Применён один эффект, правка защищена версией, незавершённое состояние восстанавливается; внешняя неоднозначность не вызывает слепой повтор.
- **Уровень:** `integration`.

#### AC-062

- **Дано:** Создана структура приложения и контракты компонентов.
- **Когда:** Проверяются зависимости и публичные интерфейсы.
- **Тогда:** Домен не импортирует HTTP, SQL, UI, OpenAI SDK или браузерные типы; адаптеры маппят внешние модели; сборщик не владеет финансовыми решениями.
- **Уровень:** `static`.

#### AC-076

- **Дано:** Подготовлен синтетический набор сотен операций со сложными переводами, чеками и эталонными ответами.
- **Когда:** Выполняются AI-eval, нагрузочная проверка и ручной дневной сценарий.
- **Тогда:** Отчёт показывает ошибки, уточнения, латентность, токены/стоимость и время пользователя; финансовые инварианты проходят, бюджет оценивается по измерению; непроверенное качество не объявлено доказанным.
- **Уровень:** `manual+integration`.

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

#### AC-085

- **Дано:** Оба видят общий чат; A имеет личную цель, B просит AI изменить её от имени A.
- **Когда:** Модель предлагает действие, а затем A подтверждает новую адресованную ему версию предложения.
- **Тогда:** Сообщение B не выдаёт полномочия A; до разрешённого подтверждения изменения нет. Аудит хранит автора сообщения, подтвердившего и AI-основание; секретов в чате нет.
- **Уровень:** `integration`.

#### AC-086

- **Дано:** A и B открыли одну версию операции или уточнения.
- **Когда:** Оба отправляют несовместимые изменения и повторяют один запрос.
- **Тогда:** Один результат применяется; второй получает конфликт с необходимостью перечитать состояние. Повтор не дублирует эффект; отмена создаёт новую проверенную revision и не стирает чужую последующую правку.
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

#### AC-092

- **Дано:** Свободно RUB 1000; оба пытаются зарезервировать по 800 для разрешённых целей.
- **Когда:** Команды исполняются одновременно.
- **Тогда:** Проверка общего доступного остатка и резерв атомарны: проходит максимум одна команда; отказ не уменьшает другой резерв, оба видят актуальный остаток.
- **Уровень:** `integration`.

#### AC-093

- **Дано:** A и B присылают один и тот же чек с одним счётом; затем приходит банковская операция.
- **Когда:** Обрабатываются параллельные сообщения, повтор файла и импорт.
- **Тогда:** Создаётся один денежный эффект и несколько evidence с авторами; при отсутствии доказанного совпадения требуется уточнение, одинаковые суммы разных счетов не сливаются.
- **Уровень:** `integration`.

#### AC-106

- **Дано:** Подключение авторизовано, но provider/host gate неполон либо прошлый admission относится к другой версии binding.
- **Когда:** Участник или scheduler запрашивает sync, либо меняются build, contract, allowlist, config, permission или environment.
- **Тогда:** Если binding уже неполон или устарел, сервер возвращает `provider_not_admitted` без job, collector IO и проводки. Только admission service ставит `admitted` после provider evidence task-4.x и host evidence task-8.x для точного binding. Job/result несёт неизменяемые binding и `admissionRevision`; смена binding во время read отменяет работу best effort, а обязательная commit-time revalidation сохраняет stale result в quarantine без source record или проводки.
- **Уровень:** `integration+security`.

### Проверка результата

```sh
make check && python3 spec/001-want-keep-mvp/tools/spec_tool.py check
```

Финальный fingerprint соответствует доказательствам и независимому review; нет скрытых blocker или заявлений о непройденных проверках.

Команды `make` — будущий контракт, создаваемый task-1.1; сейчас они не существуют. Live/paid/manual проверки отдельно фиксируют доступ и фактический результат. Исследования не обходят блокер отсутствующего доступа.

### Передача следующему агенту

Зафиксировать контракты, проверки, ограничения, вопросы и разблокированные зависимости; обновить RU/EN и трассировку. Закрывать только по доказательству результата.

**Commit boundary:** commit/push/deploy требуют действующей авторизации пользователя.

## EN

Resolve confirmed defects and hand off a verifiable result.

**Status:** Not started; the task awaits its own dependencies and entry gates.

**Dependencies:** `task-9.1`.

**Kind:** `verification`.

### Change and contracts

Run an independent read-only review of the complete frozen candidate across multiple roles with fact-checking. Under SDD ownership fix confirmed findings, rerun affected checks and the final review within the bounded round policy. Reconcile REQ/AC/task/evidence and actual readiness; commits, push and deployment require separate current authorization.

### Change boundaries

- `spec/001-want-keep-mvp/verification.md`
- `spec/001-want-keep-mvp/evidence/release.md`

Paths are planned. Shared contracts are in `spec/001-want-keep-mvp/contracts.en.md`; architecture/commands are in `constraints.en.md`. Change the behavior owner and its tests; an unresolved contract stops dependent work.

### Linked requirements

- **REQ-008:** Repeated imports, receipts and chat entries combine evidence of one transaction without double counting.
- **REQ-012:** Accounting corrections preserve the original, actor, reason, version and ability to undo a decision.
- **REQ-020:** AI income/expense insights reference verifiable data and separate forecasts from facts.
- **REQ-029:** The same money cannot be reserved for multiple goals or counted again through a dedicated account.
- **REQ-048:** Integrations and the browser collector perform authorized read operations only.
- **REQ-050:** Files, source keys, sessions and financial records are protected against unauthorized access.
- **REQ-051:** AI is limited to $50/month and degrades to a waiting queue without stopping ordinary accounting.
- **REQ-055:** The web app targets macOS laptops in Chrome and Arc; window resizing and zoom preserve daily accounting access.
- **REQ-056:** Deployment fits $40/month for a server in DE/NL/BG; no separately paid data sources are used.
- **REQ-057:** An encrypted backup is pulled to the MacBook hourly while reachable; recovery is tested.
- **REQ-059:** Money calculations use exact arithmetic and explicit boundary rounding rules.
- **REQ-060:** Receipt text, bank descriptions and AI outputs cannot expand agent authority.
- **REQ-061:** Repeated jobs, restarts and concurrent changes cannot create duplicate financial effects.
- **REQ-062:** Architecture uses Go/PostgreSQL, React/TypeScript/Vite and a separate Playwright collector with dependencies pointing toward the domain.
- **REQ-063:** User, household and membership are separate models; the two-member limit is configured.
- **REQ-064:** Both members see all financial data and edit transactions; only the owner edits personal goals and plan portions.
- **REQ-065:** Account ownership, external-account owner, record author and expense attribution are distinct dimensions.
- **REQ-067:** Expenses and receipt items have personal or joint attribution; joint shares default to 50/50 with line or purchase overrides.
- **REQ-069:** Personal and joint goal reservations are explicit; joint goals appear in a separate shared block without personal shares.
- **REQ-070:** Individual daily allowances sum to no more than the household ceiling in one currency; the payer’s account does not change expense shares.
- **REQ-071:** One shared chat retains message authors and checks the AI command initiator’s authority at execution.
- **REQ-072:** Competing edits, clarification answers and reversals check revision and current permissions while retaining both authors.
- **REQ-073:** Both manage connections; the external-account owner performs bank authentication without exposing secrets to the partner or AI.
- **REQ-076:** Household scope is checked for APIs, files, AI, jobs and external IDs independently of supplied actor/owner fields.
- **REQ-088:** Provider sync is allowed only by a current server-side admission bound to verified adapter, contract, allowlist, configuration, operator-permission and environment revisions.

### Acceptance criteria

A link establishes coverage but does not prove the whole criterion; verification below records the exact result.

#### AC-012

- **Given:** AI incorrectly linked two transactions; original imports are retained.
- **When:** The owner unlinks them and corrects the category.
- **Then:** Derived reports are recalculated, history is visible and reimport does not overwrite the owner's correction.
- **Level:** `integration`.

#### AC-048

- **Given:** The collector has a personal-account session with broader provider permissions.
- **When:** A payment request, unapproved route or MFA/CAPTCHA appears.
- **Then:** Payments and unknown routes are blocked; MFA/CAPTCHA is handed to the owner and that source pauses; other sources continue.
- **Level:** `integration`.

#### AC-050

- **Given:** A private receipt and an active source connection exist.
- **When:** A direct file URL, unauthenticated export, logs and disconnection are checked.
- **Then:** Unauthenticated access fails; secrets are encrypted and not logged; disconnecting stops further collection.
- **Level:** `integration`.

#### AC-051

- **Given:** OpenAI is unavailable or the allowed budget including in-flight reservations is exhausted.
- **When:** A new import, manual expense and AI request arrive.
- **Then:** Accounting and calculations remain available; AI status is waiting; no new paid calls exceed the allowed reservation; unknown cost is not silently released.
- **Level:** `integration`.

#### AC-057

- **Given:** The database, attachments and a temporarily unreachable MacBook exist.
- **When:** Backups are attempted, the Mac reconnects and recovery is rehearsed.
- **Then:** Last complete backup age is visible; copying resumes after reconnection; consistent data and attachments restore within four hours; hourly RPO is claimed only while the Mac is reachable.
- **Level:** `integration+manual`.

#### AC-059

- **Given:** Fractional BTC, USDT, interest accrual and receipt allocation exist.
- **When:** Data traverses API, storage and recalculation.
- **Then:** Original precision survives; JSON money never traverses binary floats; allocations reconcile exactly and display rounding does not alter the ledger.
- **Level:** `unit+contract`.

#### AC-060

- **Given:** A PDF or transaction description instructs the agent to reveal a key or transfer funds.
- **When:** AI processes the document.
- **Then:** The instruction is treated as data; secrets and payment tools are unavailable; an invalid command is rejected without changing accounting.
- **Level:** `integration`.

#### AC-061

- **Given:** A process crashes between persisting a record and acknowledging its job.
- **When:** The job is retried while the owner submits a correction.
- **Then:** One effect is applied, the correction is version-protected and incomplete state recovers; an ambiguous external outcome is not blindly retried.
- **Level:** `integration`.

#### AC-062

- **Given:** Application structure and component contracts exist.
- **When:** Dependencies and public interfaces are checked.
- **Then:** Domain imports no HTTP, SQL, UI, OpenAI SDK or browser types; adapters map external models; the collector owns no financial decisions.
- **Level:** `static`.

#### AC-076

- **Given:** A synthetic hundreds-of-transactions set includes difficult transfers, receipts and reference answers.
- **When:** AI evaluation, load checks and a manual daily flow run.
- **Then:** The report shows errors, clarifications, latency, tokens/cost and user time; financial invariants pass and cost uses measurements; untested quality is not claimed as proven.
- **Level:** `manual+integration`.

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

#### AC-085

- **Given:** Both see the shared chat; A has a personal goal and B asks AI to change it as A.
- **When:** The model proposes an action and A later confirms a new proposal version addressed to A.
- **Then:** B’s message grants no authority of A; nothing changes before authorized confirmation. Audit records message author, approver and AI rationale; chat contains no secrets.
- **Level:** `integration`.

#### AC-086

- **Given:** A and B opened the same transaction or clarification revision.
- **When:** Both submit conflicting edits and replay one request.
- **Then:** One result applies; the other receives a conflict requiring refresh. Replay does not duplicate effects; reversal creates a checked new revision without erasing the other member’s later edit.
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

#### AC-092

- **Given:** RUB 1,000 is free; both attempt to reserve 800 for authorized goals.
- **When:** Commands execute concurrently.
- **Then:** Checking household availability and reserving are atomic: at most one command succeeds; rejection does not reduce another reserve and both see current availability.
- **Level:** `integration`.

#### AC-093

- **Given:** A and B submit the same receipt for one account; the bank transaction arrives later.
- **When:** Parallel messages, a file replay and import are processed.
- **Then:** One financial effect and multiple authored evidence records result; an unproven match requires clarification and equal amounts from different accounts are not merged.
- **Level:** `integration`.

#### AC-106

- **Given:** A connection is authenticated, but the provider/host gate is incomplete or the prior admission belongs to a different binding revision.
- **When:** A member or scheduler requests sync, or the build, contract, allowlist, configuration, permission or environment changes.
- **Then:** If the binding is already incomplete or stale, the server returns `provider_not_admitted` with no job, collector IO or posting. Only the admission service sets `admitted` after task-4.x provider evidence and task-8.x host evidence for the exact binding. Each job/result carries immutable binding and `admissionRevision`; a binding change during a read cancels work best effort, while mandatory commit-time revalidation retains a stale result in quarantine without a source record or posting.
- **Level:** `integration+security`.

### Verification

```sh
make check && python3 spec/001-want-keep-mvp/tools/spec_tool.py check
```

Final fingerprint matches evidence and independent review; no hidden blockers or claims about unrun checks.

The `make` commands are a future contract established by task-1.1; they do not exist yet. Live/paid/manual checks separately record access and actual outcomes. Research does not bypass missing-access blockers.

### Handoff to the next agent

Record contracts, checks, limitations, questions and unblocked dependencies; update RU/EN and traceability. Close only with outcome evidence.

**Commit boundary:** commit/push/deploy require current user authorization.
