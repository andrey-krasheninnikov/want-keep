# Архитектурные ограничения

[English](constraints.en.md)

Стек выбран пользователем (D-17). task-1.1 создаёт воспроизводимые manifests и buildable foundation без продуктового runtime/API. Остальная часть документа описывает целевую архитектуру.

Закреплены Go 1.26.5 и Node.js 24.19.0 LTS. Web использует React 19.2.8, Vite 8.2.2, TypeScript 6.0.3, Base UI 1.8.0 и shadcn 4.21.0; collector использует Playwright 1.63.0. Прямые npm-зависимости закреплены точными версиями, каждый пакет имеет собственный lockfile.

## Владельцы поведения и структура

Модульный монолит Go с API и worker-процессами, managed PostgreSQL, React/TypeScript/Vite и отдельным Playwright TypeScript collector. Production: один VPS 2 vCPU/4 ГБ/50 ГБ в Германии для приложения и managed PostgreSQL 1 vCPU/2 ГБ/20 ГБ в той же private VPC без публичного IP; Docker Compose управляет приложением, но не production-БД. Development и integration используют изолированный PostgreSQL-контейнер. Redis, Kafka, Kubernetes, vector DB и Python-сервис не обязательны. Конфигурация и незакрытые runtime gates: [исследование task-0.9](evidence/hosting.md).

| Планируемая область | Ответственность |
| --- | --- |
| `backend/internal/<feature>/` | Money/accounts/ledger/budget/goals/forecast/credit/savings/returns/trading/mining и их application-контракты. Новая структура по владельцам, не глобальные helpers/utils. |
| `backend/internal/delivery/`, `api/openapi.yaml` | HTTP, auth boundary, schema validation и transport mapping. OpenAPI — source; generated bindings не редактировать вручную. |
| `backend/internal/integrations/<provider>/` | Проверенный контракт платформы, DTO, gateway/mapper, replay и error classification. Никаких импортов UI. |
| `backend/internal/storage/`, `backend/migrations/` | Транзакции, persistence mapping, migrations; бизнес-решения не прятать в SQL. |
| `backend/internal/ai/`, `gateways/openai/` | Application-команды и ограниченная оркестрация / внешний OpenAI SDK. Домен не зависит от SDK. |
| `web/src/features/<feature>/` | Локальные экраны, формы и состояние; пересечение features через публичные API, узкие query subscriptions. |
| `collector/src/providers/`, `collector/contracts/` | Изолированные read-сценарии кабинетов и транспортный контракт сбора. Не владеет категоризацией или финансовым учётом. |
| `deploy/`, `ops/` | Развёртывание, резервирование и восстановление; ключи и данные не входят в Git. |

Money — небольшой самостоятельный доменный контракт. Остальные shared-модули не создавать без доказанного общего владельца. Межмодульную координацию выполнять в application-слое через публичные контракты; не импортировать чужие private/data/presentation реализации.

## Поток и сохранение

Источник/чат → сохранённое доказательство → mapping и доменная валидация → атомарные операции/проводки → AI-review конкретной версии → проверенная команда или clarification → производные отчёты.

Original source records неизменяемы; изменение источника — новая версия. Операция связывает несколько доказательств. Financial posting и outbox/job сохраняются в одной транзакции. Source ID уникален внутри household + stable external-account identity + product/log namespace. Connection/сессия хранится как происхождение, а не постоянная identity счёта. Точная идемпотентность не заменяет экономическое сопоставление разных источников.

Деньги хранятся как точные decimal и передаются строками; source precision сохраняется. Native ledger и исторические rate snapshots отделены от форматирования. Односторонние/неполные переводы, неподдерживаемые активы, unknown balances и pending события остаются явно неполными.

## Безопасность и доверие

- Отдельные пользователи с членством в семье; bootstrap первого участника однократный и управляется оператором. Passkey требует проверенных origin/RP/challenge/purpose/expiry. Recovery-коды хешированы, одноразовы; восстановление отзывает только старые сессии/подписки восстанавливаемого пользователя.
- Сессия браузера хранится в Secure HttpOnly SameSite cookie; изменяющие HTTP-запросы защищены от CSRF. Actor берётся из проверенного серверного контекста; принадлежность хранится у ресурса, членство проверяется независимо от JSON клиента.
- Ключи источников и банковские сессии шифруются отдельно от данных; master/recovery keys вне БД, логов и Git. Download/preview каждого вложения авторизован.
- Collector изолирован процессом, профилями, сетью и allowlist проверенных действий/маршрутов/payload. Одного GET/POST allowlist недостаточно: проверяется семантика действия. Неизвестное действие блокируется, MFA/CAPTCHA передаётся владельцу. AI не управляет браузером.
- Описания операций, файлы и ответы модели недоверенные. AI видит разрешённые данные без секретов и вызывает только типизированные application-команды; не получает SQL, raw HTTP, shell, платёжные или торговые tools.
- Файлы проверяются до AI: MIME/signature, размер/число страниц, безопасное чтение. Активный контент и произвольные ссылки не исполняются. Финансовые тексты/документы не попадают в telemetry.
- Корректировки и связывание имеют expectedRevision, evidence и audit. Предложения изменить утверждённый план/цели требуют подтверждения той же версии.

## Отказы и наблюдаемость

Импорт и AI — независимые очереди. У source sync есть leases, bounded retries, checkpoint и время успешного обновления. При отказе одного источника другие продолжают работу. Восстановление повторяет идемпотентные read jobs; неоднозначные платные/внешние эффекты сначала сверяются. Неполный источник получает `source_partial`, коллизия identity — `source_ambiguous`; подтверждённый monetary effect сохраняется, а неподтверждённый эффект не проводится. D-43 admission проверяется до постановки sync job; stale/missing binding даёт `provider_not_admitted` без provider IO.

AI-reviewed не равен posted. При недоступности AI подтверждённые source/manual операции сохраняются, расчёты работают; unknown classification показывается отдельно. Неустановленная сумма/счёт/валюта остаётся draft. Stale AI не применяется.

Логи: код, correlation ID, source/job/operation ID, длительность и категория ошибки. Метрики: свежесть/полнота импорта, backlog AI, уточнения, ошибки сумм/связей, usage/reserved spend, возраст backup. Никаких ключей, чеков или полных сообщений.

## Контракт команд

Корневой Makefile реализует общий интерфейс проверок. `make check` запускает только существующие проверки foundation; команда будущей suite завершается ошибкой до появления реализации, а не сообщает пустой успех.

| Команда | Обязательное поведение |
| --- | --- |
| `make check` | Форматирование в check-mode, lint/typecheck, unit checks Go/web/collector, docs и contract-generation check. |
| `make test-go PKG=./internal/<feature>/...` | Из `backend/` запустить тесты указанного пакета с repo-pinned Go. |
| `make test-web FILTER=<feature>` | Запустить соответствующие тесты web в headless режиме. |
| `make test-collector FILTER=<suite>` | Запустить ограниченные синтетические collector сценарии без внешних аккаунтов. |
| `make test-integration AREA=<suite|all>` | Изолированный PostgreSQL/сервисы и fault/concurrency сценарии; не production. |
| `make test-contract PROVIDER=<name>` | Проверить versioned synthetic provider fixtures и нормализацию. |
| `make check-contracts` | Проверить OpenAPI/schema и воспроизводимость generated output. |
| `make e2e SCENARIO=<name|all>` | Браузерные сценарии на тестовых данных. Реальное устройство отдельно. |
| `make eval-ai SUITE=<name|all>` | Offline fixtures по умолчанию; live-run требует ключа, явного лимита прогона и отчёта usage. |
| `make check-deploy` | Compose/config/resource/security validation без provisioning. |
| `make backup-check MODE=synthetic`, `make restore-check MODE=synthetic` | Проверки целостности/восстановления в изоляции с измеренным временем. |

Названия suites из карточек задач — часть контракта runner. Отсутствующая suite завершает команду ошибкой, не успешным пустым прогоном.

## Готовность, миграции и rollout

task-1.1 выполнена как независимая техническая основа. task-0.10 закрыла фундаментальные решения D-37–D-43 и выпустила [Ready-план](plan.md). Реализация начинается с task-1.2 и следует собственным зависимостям. Неопределённые provider fields не заполняются вымышленными endpoint или значениями: safe unknown/partial/ambiguous является частью готового контракта.

SDD Ready и эксплуатационный допуск разделены. Каждый provider deployment выключен по умолчанию. `backend/internal/connections/admission/` — application owner aggregate/repository interface; task-1.3 реализует storage adapter и атомарные transitions. task-4.x подтверждает provider evidence, task-8.x — host/deployment evidence; только server-owned admission service объединяет оба результата для точного D-43 binding build/contract/allowlist/config/permission/environment. Смена binding или failed/revoked check повышает `admissionRevision`, возвращает `pending|blocked`, инвалидирует не начатые jobs и проверяется collector перед provider IO. Уже начатый result обязан пройти commit-time revalidation в транзакции source/posting/outbox; stale result остаётся в quarantine. Провал gate блокирует только соответствующий connector deployment или production, а не разработку домена и других адаптеров.

Схемы появляются новыми миграциями; применённые миграции не переписываются. Обратная совместимость проверяется для действующего API/данных; expand → backfill → switch → contract применяется только при реальной необходимости. Rollback не удаляет журнал, файлы или пользовательские правки.

CI, synthetic integration, live source readback, physical-device push и restore rehearsal — разные доказательства. Только полный набор обязательных AC закрывает MVP. Финальный review выполняется read-only с независимым fact-check; SDD-владелец исправляет подтверждённые findings.

## Семейная область и права

`backend/internal/household/` владеет User/Household/Membership и публичной policy проверки членства/прав. Владельцы budget/goals/ledger/integrations применяют её в своих use cases совместно с доменными инвариантами. Роль текущих участников — member; принадлежность ресурса personal/household и personalOwnerId не подменяются ролью. Не создавать редактор произвольных ролей или поля первого/второго партнёра.

Все финансовые объекты, файлы, chat retrieval, задания, idempotency и поиск совпадений ограничены householdId. Principal устанавливает сессия или проверенный контекст фонового задания; пользовательский фильтр отчёта его не меняет. Проверка family scope, права, expectedRevision и мутация выполняются в одной транзакционной границе. Права UI не заменяют backend. Тестовая вторая семья нужна для проверки изоляции, а не как новая функция публичного SaaS.

Оба правят операции и подключения, только владелец — личную цель и личную часть плана; общие изменяет любой member. Право на исправление факта расхода не даёт права менять чужой план. Повтор команды и конфликт версий различаются. Отключение коннектора меняет его generation/lease; результат предыдущего поколения не применяется. Ввод банковских секретов изолирован от общего чата и другого участника.

Финансовая identity источника включает семью, провайдера и проверенный реальный внешний аккаунт/продукт/log. Пересоздание connection не обнуляет дедупликацию; одинаковые source ID разных внешних аккаунтов не сливаются. При недостатке identity сбор сохраняет evidence, но не создаёт второй финансовый счёт вслепую.

## Desktop и представление

Дизайн, экраны и навигация являются контрактами UI, дополняющими API: [design](design.md), [screens](screens.md), [navigation](navigation.md). Design system — узкий владелец токенов/примитивов/motion; features владеют задачами пользователя. Сервер возвращает суммы, объяснения и статусы, клиент не повторяет денежные формулы. Состояния и animation events не управляют финансовым журналом. Foundation закрепляет Base UI/shadcn и тёмные базовые токены; шрифты, компоненты и пользовательские экраны принадлежат последующим UI-задачам.

## PostgreSQL task-1.3

pgx v5.10.0 остаётся в storage; domain/application не импортируют драйвер или pgx.Tx. Application boundary connections/admission распознаётся архитектурным тестом. Тестовая БД PostgreSQL 17.11 закреплена digest; production major подтверждает task-8.1. READ COMMITTED + admission-before-household lock order; maintenance отделён от application. [Контракт хранения и запуск](evidence/task-1.3-storage.md).

## Identity task-1.4

Auth использует domain/application, WebAuthn adapter, delivery/identity и storage. Правила D-45 и границы приёмки: [контракт](contracts.md#task-14-вход-и-восстановление-d-45), [evidence](evidence/task-1.4-identity.md).

Task-1.5 (D-46) добавляет отдельные keyring и изолированный processor. Domain/application не импортируют crypto storage, SQL, HTTP или декодеры; секреты доступны только credentials adapter. Runtime processor лишён сети/ключей/БД/общего каталога. Отказ защищённых функций не останавливает identity/учёт. [Контракт](contracts.md), [проверки и эксплуатационная передача](evidence/task-1.5-privacy.md).

## Household task-1.6

Приглашение не назначает principal и не даёт финансового доступа до атомарной регистрации. Identity lock предшествует family/invitation lock; family scope и joining scope не смешиваются. Раздельные доменные policies сохраняют текущего владельца, actor, payer и external owner. Миграция 005 расширяет схему, secret responses не replay-ятся. [Контракт](contracts.md#task-16-приглашения-и-семейные-права), [evidence](evidence/task-1.6-household.md).

### Граница хранения счетов (task-2.1)

API счетов регистрирует команды до исполнения с проверкой сессии и использует порядок блокировок identity → household. Чтение выполняется в repeatable-read снимке без финансовых write locks. Миграция 007 сохраняет 001–006 и помечает старые недоказанные остатки как legacy. Только admitted импорт создаёт импортные продукты, наблюдения источника и алиасы; пользовательские команды не могут их подделать. Проекции журнала и неизменяемые наблюдения платформ хранятся и интерпретируются раздельно. Принадлежность счёта не назначает владельца банковской сессии. См. [контракт счетов](contracts.md#task-21--счета-и-начальные-остатки) и [границы проверки](evidence/task-2.1-accounts.md).

## Ingestion task-3.2

`collector/contracts/v10/ingestion.openapi.yaml` — единственный wire source внутреннего контракта версии 10. `make generate-contracts` обновляет Go и TypeScript, `make check-contracts` сравнивает оба результата с временной генерацией. Generated DTO остаются в `integrations/contract` и collector; `integrations/domain` не зависит от transport, SQL, HTTP, Playwright или provider SDK.

`integrations/application` координирует `ProviderGateway`, `EvidenceStore`, accounts importer, ledger source writer и `connections/admission`. Gateway создаётся из server-owned конфигурации с полным неизменяемым D-43 binding; application сравнивает его с binding задания до manifest/provider IO. Provider IO выполняется только после pre-read fence и вне финансовой транзакции. Raw evidence сохраняется до `CommitPage`, сразу связывается с server-derived household/job и получает durable disposition даже при последующей ошибке валидации; callback не выполняет внешний IO. Счёт может быть разрешён без balance observation; любая ссылка balance/posting требует account descriptor той же самостоятельной страницы. Серверные principal, external owner, internal IDs, revisions и timestamps не берутся из payload.

Контракт fail closed: только read capability, строгий JSON с обязательными required-полями и запретом явного `null` в non-nullable полях, каноническое UTC-время с `Z`, согласованные Unicode code-point/byte limits и точное эхо job/binding/revision/cursor для страницы и provider failure. Cursor проверяется при сохранении sync-result отдельно от неизменяемой lease identity: запоздалый outcome прежней страницы не меняет состояние задания, а продвижение checkpoint не ломает heartbeat. NUL и lone surrogate запрещены до identity/storage; составные ключи кодируются структурно. Gaps уникальны и ограничены, а необязательный lookback не смешивается с явным нулём. Manifest точной admitted-сборки связывается с provider binding и ограничивает product, log namespace, record kind и необходимое read-действие до сохранения evidence. Подтверждённые provider mappings, включая Raiffeisen/Ozon `RUR → RUB`, применяются в accounts-owned policy с сохранением исходного кода; неизвестные symbols не преобразуются эвристически. Карточный alias допускает только безопасное название и собственные последние четыре ASCII-цифры. Отдельная fee-проводка поддерживается, но свободный `feeId` не пересекает v10 boundary без типизированной correspondence. Server-side account/source ambiguity предварительно группирует полные D-39 keys страницы, дедуплицирует одинаковые факты, сохраняет evidence, делает coverage partial и не проводит конфликтующую группу; независимые записи страницы сохраняются. Evidence получает durable `staged` до финансового commit. Финансовый эффект и неизменяемая receipt из миграции 018 сохраняются атомарно; bounded readback завершает доказанный commit, а недоказанный исход оставляет `staged`. Отдельный lifecycle context закрывает только доказанный rejected page или provider outcome. Типизированный provider outcome атомарно связывает evidence с household/job и сохраняет retry/reauth/terminal lifecycle под sync-result fence. Stale или отвергнутый result пересекает только evidence/quarantine boundary. Реальные provider routes, network isolation и live permission доказываются следующими задачами; наличие synthetic gateway не является admission в production.

## Browser collector task-3.3

Collector — отдельный Node.js-процесс без TCP listener. Go связывается с ним только через Unix socket. Runtime использует build-owned exact binding/origin/action allowlist и новый непостоянный Playwright context для каждого job. В этом context отключён JavaScript; авторизованное чтение использует отфильтрованные cookies вне кода страницы, точный адрес и запрет redirect. Пользовательские Chrome/Arc-профили, URL, selectors, JavaScript и route rules не входят во входной контракт. Только чтение и проверенный POST заказа выписки разрешены; остальные действия и неизвестные сетевые переходы закрываются. Для реального провайдера, требующего JavaScript страницы, нужны отдельно проверенный сценарий и допуск к эксплуатации.

Сессия расшифровывается существующим vault только на время job. `external_started` фиксируется непосредственно перед IO, а неизвестный outcome не повторяется автоматически. Evidence шифруется существующим connection keyring до записи в PostgreSQL; идентификаторы исходных доказательств остаются в ciphertext, AAD включает household/job/page/item. Миграция 022 разрешает только однонаправленное изменение disposition, не выдаёт приложению plaintext и сохраняет staged recovery. Provider-specific allowlist, egress и разрешение task-4.x/task-8.x остаются обязательными перед эксплуатацией.

## OpenAI task-5.1

`ai/domain` владеет точной стоимостью, usage и состояниями; `ai/application` — резервированием, исполнением и сверкой; `gateways/openai` — единственное место импорта SDK; storage скрывает SQL. Worker связывает эти границы, но не применяет AI-предложение. Архитектурная проверка запрещает SDK, HTTP и pgx в domain/application.

Миграция 016 добавляет только append-only попытки и их состояния. Household lock сериализует семейный бюджет и два слота. Маркер external-started сохраняется до IO; provider IO всегда вне транзакции. Любой известный provider outcome и соответствующее terminal/retry job-state записываются в одной household-транзакции; неизвестный outcome атомарно оставляет job в `unresolved`. Доверенный resumer только переводит подходящие budget-waiting jobs обратно в `ready`; после Input Tokens подсчёта резерв, слот и глобальный unknown-barrier повторно проверяются под той же блокировкой. Maintenance role не читает и не изменяет таблицы AI/jobs напрямую: ей доступна только проверяемая `SECURITY DEFINER`-операция сверки истёкшего либо unresolved вызова. Application role не обновляет неизменяемую историю. [Контракт и проверка](evidence/task-5.1-openai-gateway.md).

## Transaction review task-5.2

Новая runtime-схема команд и prompt не наследуют допуск исторической квалификации task-0.8. Model/effort, тарификация и лимит task-5.1 сохраняются; `production_admitted=false`. Проверки task-5.2 выполняются offline на синтетических ответах и PostgreSQL. Live qualification, OpenAI IO, извлечение чеков, чат, продуктовые бюджет/цели и браузерная приёмка этим результатом не подтверждаются. Новых зависимостей и публичных универсальных инструментов исполнения нет. Автоматическая классификация подчиняется сохранённой защите конкретных полей; подтверждение пользователем создаёт обычное аудированное решение с возможностью отмены.
