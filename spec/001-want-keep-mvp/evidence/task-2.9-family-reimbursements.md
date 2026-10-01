# Task-2.9 — явные долги и семейные возмещения

Реализован backend/API явных долгов между участниками семьи. Исходная база: `a4a77365b909080b48093bc5304216489526642c`; ветка: `feat/task-2.9-family-reimbursements`. Зависимости task-2.4 и task-2.8 включены. Новых библиотек нет; миграции 001–018 и production не менялись.

Ledger хранит creditor и debtor как разные активные `MembershipID`, точную native-сумму, остаток, состояние, автора и неизменяемые revisions. Долг появляется только после явной команды. Распределение расхода, payer и обычный перевод не создают его. Необязательная ссылка на расход фиксирует текущую posted revision, но не выводит сумму долга. `open|settled|attention_required|voided` не входят в семейные активы, остатки, доходы или расходы.

Погашение использует существующий posted перевод между личным счётом debtor и личным счётом creditor. Комиссия исключена из доступной principal-суммы. Stable transaction/matching key учитывает один денежный носитель независимо от выбранной стороны доказанной группы. Один перевод можно распределить между несколькими долгами только в пределах ещё не использованной received principal. Для одного актива `transferAmount` равна `settledAmount`; для разных активов обе суммы задаются явно без выведенного курса или паритета.

Финансовая правка, exclusion, cancellation, reversal связанного перевода или смена владельца его счёта переводит active settlement в `stale` и восстанавливает остаток. Текстовая правка перевода сохраняет связь. Изменение исходного расхода переводит долг в `attention_required`; undo не восстанавливает ссылку на устаревшую revision расхода. Corrections не позволяют сменить стороны или актив при active settlements и не уменьшают principal ниже уже погашенной суммы. Selective undo сохраняет независимые поздние решения и обнаруживает пересечение полей, включая A–B–A; undo settlement восстанавливает долг.

Миграция `019_family_reimbursements.sql` добавляет current records, immutable revisions, field versions, decisions, settlements, transfer usage, operation links и audit. Составные household FK, точный `NUMERIC`, immutable triggers и минимальные grants сохраняют изоляцию. Command `pending` фиксируется отдельно; reimbursement revision, settlement, audit, outbox и terminal result сохраняются атомарно под семейной блокировкой. После очистки D-41 неизменяемый audit запрещает повтор команды с прежним ключом. Command result дополнительно проверяет текущий доступ к долгу.

OpenAPI реализует `GET/POST /reimbursements`, чтение карточки и истории, corrections, settlements и selective undo. Mutation использует session actor, Origin/CSRF, `Idempotency-Key`, `expectedRevision`, command recovery и `no-store`. List/history используют session-bound keyset cursor, размер страницы 50 по умолчанию и максимум 100. Чужой и отсутствующий долг не различаются.

## Проверки и границы

Обязательные проверки: `make check`, `make test-integration AREA=all`, `make test-family-reimbursements-race`, затронутые race suites, privacy suite и `git diff --check`. PostgreSQL 17.11 закреплён digest; отсутствие БД завершает integration-suite ошибкой. Family-reimbursements integration/race включены в CI.

Suite проверяет точность RUB, USD, USDT, USDC, BTC и ETH; погашение RUB 300 как 100+200; один перевод между несколькими долгами; запрет перепогашения; cross-asset с двумя суммами; отсутствие дохода/расхода у principal перевода; replay даже после очистки D-41; конкурентные settlement; correction/undo и A–B–A. Отдельно проверяются обе стороны matched-перевода, восстановление долга после финансовой правки или смены владельца счёта, сохранение связи после текстовой правки, запрет undo устаревшей ссылки на расход, `attention_required` после изменения расхода, семейная изоляция, command visibility, CSRF, pagination, migration, immutable history и grants непривилегированной роли.

Доказаны backend-части AC-006, AC-082 и AC-086. UI SCR-013, наличное возмещение без существующего ledger-перевода, реальные банковские данные, браузерная и production-приёмка остаются последующим задачам. SDD остаётся **Ready for development**.
