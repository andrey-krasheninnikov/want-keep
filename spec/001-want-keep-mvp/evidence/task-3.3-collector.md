# Task-3.3 — изолированный браузерный сборщик

База реализации: `a4a77365b909080b48093bc5304216489526642c`; ветка: `feat/task-3.3-isolated-browser-collector`. Зависимости task-1.5 и task-3.2 включены. Реальные платформы, пользовательские браузерные профили и production не изменялись.

## Доказанный результат

Collector запускается отдельным Node.js-процессом с Playwright 1.63.0 и слушает только Unix socket с правами `0600`. Протокол содержит `GET /ready`, `POST /v1/capabilities` и `POST /v1/read`. Сервер ограничивает размер и длительность запроса и выполняет один browser job одновременно. Каждый job создаёт новый непостоянный `BrowserContext`; JavaScript, downloads и service workers отключены, context закрывается после любого исхода.

Build-owned runtime-конфигурация задаёт полный D-43 binding, `admissionRevision`, capability manifest, точный origin и разрешённые действия. До запуска браузера collector сравнивает все поля binding и revision. Allowlist допускает один `entry`, один `read` и необязательный `request_statement` с точными `from`/`to`. Повторное чтение передаёт выданный сервером cursor как единственный query-параметр `cursor`; неизвестные origin, method, path, query и payload блокируются. Чтение выполняется вне кода страницы с отфильтрованными cookies и ограниченным потоком ответа; страница не может подменить результат или повторить statement POST. Redirect, popup, download, WebSocket, service worker и payment-маршрут синтетического портала не пересекают границу. Входной контракт не содержит URL, selector, JavaScript, upload или route rule. MFA, CAPTCHA и истёкшая сессия становятся типизированными provider failures без возврата session state.

Go Unix-socket client реализует существующий `contract.RawGateway`. Worker строит его только из сохранённого sync job и действующего admission, временно получает `browser_session` через `credentials.Vault` и очищает копию после job. Маркер `external_started` устанавливается непосредственно перед `/v1/read`; capability check его не устанавливает. Подтверждённые `collector_busy`, `collector_session_invalid` и `collector_preflight_rejected` означают отказ до provider IO: маркер снимается под действующим lease; job соответственно ожидает collector, повторного входа владельца или завершается ошибкой запроса. HTTP 429 создаёт типизированный `rate_limited` с ограниченным сроком повтора. Некорректные байты тела 401/403 не маскируют запрос входа. Потеря связи после начала IO даёт `unresolved` без автоматического provider replay. Page и failure проходят существующие admission, connection generation, lease и cursor fences. Устаревший результат сохраняется только в quarantine.

Raw evidence шифруется существующим connection keyring до PostgreSQL. AAD связывает ciphertext с household, job, page и evidence reference. Миграция 021 хранит batch metadata и неизменяемые evidence items; provider evidence ID остаётся зашифрованным. Batch может перейти только из `staged` в один terminal disposition; приложение имеет минимальные права и не читает plaintext. Staged recovery после рестарта использует существующую terminal receipt и не повторяет browser IO или финансовый эффект.

Длительность ограничена также при запуске Chromium и создании context: отмена освобождает слот, закрывает остановленный браузер и поздно созданный context. И входная HTML-страница, и финансовый ответ читаются потоком с пределом 32 MiB после декомпрессии; redirect не отправляет запрос в запрещённое назначение. Последовательные страницы продолжаются в одном lease и одной попытке с чтением сохранённого checkpoint; admission проверяется для каждой страницы. Регрессии покрывают шесть страниц в одной попытке, отмену setup, большой chunked/gzip-ответ и сохранение cookies обычного gzip-ответа.

## Матрица проверок

Команды базовой матрицы: `make check`; `make test-collector FILTER=security`; `make test-integration AREA=collector`; `make test-integration AREA=all`; ingestion/jobs/storage/privacy integration и затронутые race suites; `git diff --check`. PostgreSQL suite использует изолированную PostgreSQL 17.11 под непривилегированной ролью и завершается ошибкой без `WANT_KEEP_TEST_DATABASE_URL`. Browser suite устанавливает закреплённый Chromium и обращается только к локальному синтетическому порталу. Точные результаты кандидата и CI фиксируются в PR.

При завершении по решению пользователя повторены только затронутые проверки; полный локальный набор не повторялся. Актуальный CI проверяется отдельно.

Тесты проверяют безопасный GET и statement POST, раздельные session cookies, MFA/CAPTCHA, HTTP 429, некорректное тело 401, восстановление Chromium, нормализацию route, подмену page fetch, отсутствие повторного statement POST, отключение WebRTC, payment/redirect/popup/download/WebSocket/service-worker блокировки и stale binding до browser IO. Go-тесты проверяют Unix-only transport, момент `external_started`, три подтверждённых отказа до IO, отсутствие session в result и encryption/AAD. PostgreSQL-тесты проверяют ciphertext-only storage, отсутствие provider ID в метаданных, наносекундное время, семейную изоляцию, рестарт, idempotent terminal disposition, неизменяемые items и staged recovery.

## Границы критериев

| Критерии | Доказуемая часть task-3.3 | Дальнейшая проверка |
| --- | --- | --- |
| AC-040/041/048 | Read-only runtime, точный statement POST, typed MFA/CAPTCHA/reauth и один job | Реальные provider routes, история и повторный вход владельца |
| AC-050/061/087 | Временная session через vault, изоляция context, отсутствие plaintext в результате/БД/диагностике | Пользовательский поток передачи session и браузерный UI |
| AC-079/090/106 | Server-owned binding/principal, exact pre-read fence, encrypted evidence и commit-time quarantine | Provider/deployment evidence и production admission |
| AC-060/068 | Общая безопасная collector-инфраструктура и ограниченный synthetic egress | Полные live-source и эксплуатационные сценарии |

SDD остаётся **Ready for development**. Для реального кабинета с JavaScript страницы нужен отдельно проверенный сценарий. Личные Chrome/Arc-профили, production egress и provider deployment этой задачей не подтверждены.

После слияния task-6.1 её миграция `020_rates_valuation.sql` сохранена; неприменённая collector-миграция перенумерована в `021_collector_evidence.sql` без изменения SQL. Production-схема не изменялась.
