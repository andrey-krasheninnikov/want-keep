# Task-3.3 — изолированный браузерный сборщик

База реализации: `a4a77365b909080b48093bc5304216489526642c`; ветка: `feat/task-3.3-isolated-browser-collector`. Зависимости task-1.5 и task-3.2 включены. Реальные платформы, пользовательские браузерные профили и production не изменялись.

## Доказанный результат

Collector запускается отдельным Node.js-процессом с Playwright 1.63.0 и слушает только Unix socket с правами `0600`. Протокол содержит `GET /ready`, `POST /v1/capabilities` и `POST /v1/read`. Сервер ограничивает размер и длительность запроса и выполняет один browser job одновременно. Каждый job создаёт новый непостоянный `BrowserContext`; downloads и service workers отключены, context закрывается после любого исхода.

Build-owned runtime-конфигурация задаёт полный D-43 binding, `admissionRevision`, capability manifest, точный origin и разрешённые действия. До запуска браузера collector сравнивает все поля binding и revision. Allowlist допускает один `entry`, один `read` и необязательный `request_statement` с точными `from`/`to`. Повторное чтение передаёт выданный сервером cursor как единственный query-параметр `cursor`; неизвестные origin, method, path, query и payload блокируются. Redirect, popup, download, WebSocket, service worker и payment-маршрут синтетического портала не пересекают границу. Входной контракт не содержит URL, selector, JavaScript, upload или route rule. MFA, CAPTCHA и истёкшая сессия становятся типизированными provider failures без возврата session state.

Go Unix-socket client реализует существующий `contract.RawGateway`. Worker строит его только из сохранённого sync job и действующего admission, временно получает `browser_session` через `credentials.Vault` и очищает копию после job. Маркер `external_started` устанавливается непосредственно перед `/v1/read`; capability check его не устанавливает. Потеря связи после маркера даёт `unresolved` и не запускает автоматический provider replay. Подтверждённый ответ `collector_busy` означает отказ до provider IO: маркер снимается под действующим lease, задание остаётся ожидающим. Page и failure проходят существующие admission, connection generation, lease и cursor fences. Устаревший результат сохраняется только в quarantine.

Raw evidence шифруется существующим connection keyring до PostgreSQL. AAD связывает ciphertext с household, job, page и evidence ID. Миграция 019 хранит batch metadata и неизменяемые evidence items. Batch может перейти только из `staged` в один terminal disposition; приложение имеет минимальные права и не читает plaintext. Staged recovery после рестарта использует существующую terminal receipt и не повторяет browser IO или финансовый эффект.

## Матрица проверок

Обязательные команды: `make check`; `make test-collector FILTER=security`; `make test-integration AREA=collector`; `make test-integration AREA=all`; ingestion/jobs/storage/privacy integration и затронутые race suites; `git diff --check`. PostgreSQL suite использует изолированную PostgreSQL 17.11 под непривилегированной ролью и завершается ошибкой без `WANT_KEEP_TEST_DATABASE_URL`. Browser suite устанавливает закреплённый Chromium и обращается только к локальному синтетическому порталу. Точные результаты кандидата и CI фиксируются в PR.

Тесты проверяют безопасный GET и statement POST, раздельные session cookies, MFA/CAPTCHA, payment/redirect/popup/download/WebSocket/service-worker блокировки и stale binding до browser IO. Go-тесты проверяют Unix-only transport, момент `external_started`, отсутствие session в result и encryption/AAD. PostgreSQL-тесты проверяют ciphertext-only storage, нанoseкундное время, семейную изоляцию, рестарт, idempotent terminal disposition, неизменяемые items и staged recovery.

## Границы критериев

| Критерии | Доказуемая часть task-3.3 | Дальнейшая проверка |
| --- | --- | --- |
| AC-040/041/048 | Read-only runtime, точный statement POST, typed MFA/CAPTCHA/reauth и один job | Реальные provider routes, история и повторный вход владельца |
| AC-050/061/087 | Временная session через vault, изоляция context, отсутствие plaintext в результате/БД/диагностике | Пользовательский поток передачи session и браузерный UI |
| AC-079/090/106 | Server-owned binding/principal, exact pre-read fence, encrypted evidence и commit-time quarantine | Provider/deployment evidence и production admission |
| AC-060/068 | Общая безопасная collector-инфраструктура и ограниченный synthetic egress | Полные live-source и эксплуатационные сценарии |

SDD остаётся **Ready for development**. Реальные кабинеты, личные Chrome/Arc-профили, production egress и provider deployment этой задачей не подтверждены.
