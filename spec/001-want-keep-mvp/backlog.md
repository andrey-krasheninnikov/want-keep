# Backlog Want Keep MVP

Собрано из [catalog.json](catalog.json). Редактировать каталог, затем выполнить `python3 spec/001-want-keep-mvp/tools/spec_tool.py render`.

Спецификация прошла task-0.10 и готова к разработке. Решение-полный порядок, параллелизм и entry/exit gates опубликованы в [plan.md](plan.md). Каждая задача по-прежнему ждёт собственные зависимости и runtime gates; Ready SDD не означает реализованный или принятый MVP. Карточки самодостаточны и содержат RU/EN.

| Task | Результат | Зависимости | GitHub |
| --- | --- | --- | --- |
| [task-0.1](tasks/task-0.1.md) | Проверить контракт чтения Альфа-Банк | — | [#1](https://github.com/pchkauu/want-keep/issues/1) |
| [task-0.2](tasks/task-0.2.md) | Проверить контракт чтения Райффайзенбанк РФ | — | [#2](https://github.com/pchkauu/want-keep/issues/2) |
| [task-0.3](tasks/task-0.3.md) | Проверить контракт чтения Ozon Банк | — | [#3](https://github.com/pchkauu/want-keep/issues/3) |
| [task-0.4](tasks/task-0.4.md) | Проверить контракт чтения Bybit | — | [#4](https://github.com/pchkauu/want-keep/issues/4) |
| [task-0.5](tasks/task-0.5.md) | Проверить контракт чтения Aifory Pro | — | [#5](https://github.com/pchkauu/want-keep/issues/5) |
| [task-0.6](tasks/task-0.6.md) | Проверить контракт чтения EMCD | — | [#6](https://github.com/pchkauu/want-keep/issues/6) |
| [task-0.7](tasks/task-0.7.md) | Проверить бесплатные источники курсов | — | [#7](https://github.com/pchkauu/want-keep/issues/7) |
| [task-0.8](tasks/task-0.8.md) | Измерить качество и стоимость OpenAI | — | [#8](https://github.com/pchkauu/want-keep/issues/8) |
| [task-0.9](tasks/task-0.9.md) | Проверить инфраструктуру и бюджет сервера | — | [#9](https://github.com/pchkauu/want-keep/issues/9) |
| [task-0.10](tasks/task-0.10.md) | Закрыть контракты и проверить готовность SDD | task-0.1, task-0.2, task-0.3, task-0.4, task-0.5, task-0.6, task-0.7, task-0.8, task-0.9 | [#10](https://github.com/pchkauu/want-keep/issues/10) |
| [task-1.1](tasks/task-1.1.md) | Создать структуру проекта и команды проверки | task-0.10 | [#11](https://github.com/pchkauu/want-keep/issues/11) |
| [task-1.2](tasks/task-1.2.md) | Определить денежные типы и API-контракт | task-1.1 | [#12](https://github.com/pchkauu/want-keep/issues/12) |
| [task-1.3](tasks/task-1.3.md) | Создать хранилище и транзакционные границы | task-1.2 | [#13](https://github.com/pchkauu/want-keep/issues/13) |
| [task-1.4](tasks/task-1.4.md) | Реализовать passkey и восстановление доступа | task-1.3 | [#14](https://github.com/pchkauu/want-keep/issues/14) |
| [task-1.5](tasks/task-1.5.md) | Защитить секреты и приватные вложения | task-1.4 | [#15](https://github.com/pchkauu/want-keep/issues/15) |
| [task-1.6](tasks/task-1.6.md) | Создать семью, членство и права на ресурсы | task-1.4 | [#16](https://github.com/pchkauu/want-keep/issues/16) |
| [task-2.1](tasks/task-2.1.md) | Реализовать счета и начальные остатки | task-1.3, task-1.6 | [#17](https://github.com/pchkauu/want-keep/issues/17) |
| [task-2.2](tasks/task-2.2.md) | Реализовать журнал операций и статусы | task-2.1, task-1.2 | [#18](https://github.com/pchkauu/want-keep/issues/18) |
| [task-2.3](tasks/task-2.3.md) | Добавить версии, исправления и аудит | task-2.2 | [#19](https://github.com/pchkauu/want-keep/issues/19) |
| [task-2.4](tasks/task-2.4.md) | Связать переводы и исключить дубликаты | task-2.3 | [#20](https://github.com/pchkauu/want-keep/issues/20) |
| [task-2.5](tasks/task-2.5.md) | Сверять журнал с балансом источника | task-2.3 | [#21](https://github.com/pchkauu/want-keep/issues/21) |
| [task-2.6](tasks/task-2.6.md) | Разделить категории, продавцов и товары | task-2.3 | [#22](https://github.com/pchkauu/want-keep/issues/22) |
| [task-2.7](tasks/task-2.7.md) | Реализовать возвраты и распределение расходов | task-2.2, task-2.6, task-2.8 | [#23](https://github.com/pchkauu/want-keep/issues/23) |
| [task-2.8](tasks/task-2.8.md) | Распределять семейные расходы и позиции по участникам | task-2.6, task-1.6 | [#24](https://github.com/pchkauu/want-keep/issues/24) |
| [task-2.9](tasks/task-2.9.md) | Учитывать явные долги и возмещения внутри семьи | task-2.4, task-2.8 | [#25](https://github.com/pchkauu/want-keep/issues/25) |
| [task-3.1](tasks/task-3.1.md) | Создать долговечные фоновые задания | task-1.3, task-2.3 | [#26](https://github.com/pchkauu/want-keep/issues/26) |
| [task-3.2](tasks/task-3.2.md) | Определить входной контракт коннекторов | task-3.1, task-1.2, task-2.1, task-2.2 | [#27](https://github.com/pchkauu/want-keep/issues/27) |
| [task-3.3](tasks/task-3.3.md) | Создать изолированный браузерный сборщик | task-3.2, task-1.5 | [#28](https://github.com/pchkauu/want-keep/issues/28) |
| [task-4.1](tasks/task-4.1.md) | Реализовать коннектор Альфа-Банк | task-0.1, task-3.3, task-2.4, task-2.5 | [#29](https://github.com/pchkauu/want-keep/issues/29) |
| [task-4.2](tasks/task-4.2.md) | Реализовать коннектор Райффайзенбанк РФ | task-0.2, task-3.3, task-2.4, task-2.5 | [#30](https://github.com/pchkauu/want-keep/issues/30) |
| [task-4.3](tasks/task-4.3.md) | Реализовать коннектор Ozon Банк | task-0.3, task-3.3, task-2.4, task-2.5 | [#31](https://github.com/pchkauu/want-keep/issues/31) |
| [task-4.4](tasks/task-4.4.md) | Реализовать коннектор Bybit | task-0.4, task-3.3, task-2.4, task-2.5 | [#32](https://github.com/pchkauu/want-keep/issues/32) |
| [task-4.5](tasks/task-4.5.md) | Реализовать коннектор Aifory Pro | task-0.5, task-3.3, task-2.4, task-2.5 | [#33](https://github.com/pchkauu/want-keep/issues/33) |
| [task-4.6](tasks/task-4.6.md) | Реализовать коннектор EMCD | task-0.6, task-3.3, task-2.4, task-2.5 | [#34](https://github.com/pchkauu/want-keep/issues/34) |
| [task-5.1](tasks/task-5.1.md) | Создать OpenAI gateway и контроль расходов | task-0.8, task-3.1, task-1.5 | [#35](https://github.com/pchkauu/want-keep/issues/35) |
| [task-5.2](tasks/task-5.2.md) | Проверять каждую операцию через AI-команды | task-5.1, task-2.4, task-2.6, task-2.8, task-2.9 | [#36](https://github.com/pchkauu/want-keep/issues/36) |
| [task-5.3](tasks/task-5.3.md) | Обрабатывать чеки и позиции | task-5.2, task-1.5, task-2.7 | [#37](https://github.com/pchkauu/want-keep/issues/37) |
| [task-5.4](tasks/task-5.4.md) | Реализовать чат и очередь уточнений | task-5.3, task-5.2 | [#38](https://github.com/pchkauu/want-keep/issues/38) |
| [task-5.5](tasks/task-5.5.md) | Формировать обоснованные AI-инсайты | task-5.1, task-6.8 | [#39](https://github.com/pchkauu/want-keep/issues/39) |
| [task-6.1](tasks/task-6.1.md) | Реализовать курсы и валютную оценку | task-0.7, task-2.2, task-3.2 | [#40](https://github.com/pchkauu/want-keep/issues/40) |
| [task-6.2](tasks/task-6.2.md) | Учитывать кредитки и грейс-период | task-4.1, task-4.2, task-4.3, task-2.2 | [#41](https://github.com/pchkauu/want-keep/issues/41) |
| [task-6.3](tasks/task-6.3.md) | Считать начисления и прогноз накоплений | task-4.1, task-4.2, task-4.3, task-4.4, task-4.6, task-6.1 | [#42](https://github.com/pchkauu/want-keep/issues/42) |
| [task-6.4](tasks/task-6.4.md) | Сравнивать доходность денежных потоков | task-6.3 | [#43](https://github.com/pchkauu/want-keep/issues/43) |
| [task-6.5](tasks/task-6.5.md) | Сводить торговый результат и майнинг | task-6.1, task-2.4 | [#44](https://github.com/pchkauu/want-keep/issues/44) |
| [task-6.6](tasks/task-6.6.md) | Планировать месячный бюджет и доходы | task-2.7, task-2.6, task-6.1, task-6.2, task-2.8 | [#45](https://github.com/pchkauu/want-keep/issues/45) |
| [task-6.7](tasks/task-6.7.md) | Резервировать деньги на цели | task-6.6, task-2.1 | [#46](https://github.com/pchkauu/want-keep/issues/46) |
| [task-6.8](tasks/task-6.8.md) | Считать дневные лимиты и прогноз ликвидности | task-6.6, task-6.7, task-6.3, task-2.5 | [#47](https://github.com/pchkauu/want-keep/issues/47) |
| [task-7.1](tasks/task-7.1.md) | Создать desktop-оболочку и вход RU/EN | task-1.4, task-1.2, task-7.11, task-1.6, task-2.1 | [#48](https://github.com/pchkauu/want-keep/issues/48) |
| [task-7.2](tasks/task-7.2.md) | Показать счета, операции и исправления | task-7.1, task-2.5, task-2.6, task-2.7, task-5.2, task-7.9, task-2.9 | [#49](https://github.com/pchkauu/want-keep/issues/49) |
| [task-7.3](tasks/task-7.3.md) | Создать чат с выбором счёта и файлами | task-7.1, task-5.4, task-7.9 | [#50](https://github.com/pchkauu/want-keep/issues/50) |
| [task-7.4](tasks/task-7.4.md) | Создать редактор месячного бюджета | task-7.1, task-6.6, task-7.9 | [#51](https://github.com/pchkauu/want-keep/issues/51) |
| [task-7.5](tasks/task-7.5.md) | Показать цели и резервирование | task-7.1, task-6.7, task-7.9 | [#52](https://github.com/pchkauu/want-keep/issues/52) |
| [task-7.6](tasks/task-7.6.md) | Собрать дашборд, лимиты и инсайты | task-7.2, task-7.4, task-7.5, task-6.8, task-5.5, task-7.9 | [#53](https://github.com/pchkauu/want-keep/issues/53) |
| [task-7.7](tasks/task-7.7.md) | Показать кредитки, накопления и доходность | task-7.2, task-6.2, task-6.3, task-6.4, task-6.5, task-7.9 | [#54](https://github.com/pchkauu/want-keep/issues/54) |
| [task-7.8](tasks/task-7.8.md) | Добавить уведомления и web-push | task-7.1, task-6.8, task-5.5, task-1.4, task-3.1, task-7.9 | [#55](https://github.com/pchkauu/want-keep/issues/55) |
| [task-7.9](tasks/task-7.9.md) | Добавить семейный контекст и принадлежность в интерфейс | task-7.1, task-1.6 | [#56](https://github.com/pchkauu/want-keep/issues/56) |
| [task-7.10](tasks/task-7.10.md) | Создать токены, типографику и брендовые ресурсы | task-1.1 | [#57](https://github.com/pchkauu/want-keep/issues/57) |
| [task-7.11](tasks/task-7.11.md) | Создать каталог компонентов на Base UI | task-7.10 | [#58](https://github.com/pchkauu/want-keep/issues/58) |
| [task-7.12](tasks/task-7.12.md) | Проверить desktop UX и визуальную приёмку | task-7.2, task-7.3, task-7.4, task-7.5, task-7.6, task-7.7, task-7.8, task-7.9, task-7.13, task-7.14, task-7.15 | [#59](https://github.com/pchkauu/want-keep/issues/59) |
| [task-7.13](tasks/task-7.13.md) | Создать экраны подключений и повторного входа | task-7.1, task-7.9, task-3.3, task-4.1, task-4.2, task-4.3, task-4.4, task-4.5, task-4.6 | [#60](https://github.com/pchkauu/want-keep/issues/60) |
| [task-7.14](tasks/task-7.14.md) | Создать настройки и состояние учёта | task-7.9, task-7.8, task-8.2, task-2.6 | [#61](https://github.com/pchkauu/want-keep/issues/61) |
| [task-7.15](tasks/task-7.15.md) | Добавить контекстные анимации финансовых событий | task-7.11, task-7.2, task-7.4, task-7.5, task-7.14 | [#62](https://github.com/pchkauu/want-keep/issues/62) |
| [task-8.1](tasks/task-8.1.md) | Подготовить развёртывание и состояние системы | task-0.9, task-3.3, task-5.1, task-7.8 | [#63](https://github.com/pchkauu/want-keep/issues/63) |
| [task-8.2](tasks/task-8.2.md) | Выгружать зашифрованные копии на MacBook | task-8.1, task-1.5 | [#64](https://github.com/pchkauu/want-keep/issues/64) |
| [task-8.3](tasks/task-8.3.md) | Проверить восстановление из локальной копии | task-8.2 | [#65](https://github.com/pchkauu/want-keep/issues/65) |
| [task-9.1](tasks/task-9.1.md) | Провести сквозную приёмку полного MVP | task-4.1, task-4.2, task-4.3, task-4.4, task-4.5, task-4.6, task-7.3, task-7.6, task-7.7, task-7.8, task-8.3, task-7.12 | [#66](https://github.com/pchkauu/want-keep/issues/66) |
| [task-9.2](tasks/task-9.2.md) | Провести итоговую независимую проверку и передать MVP | task-9.1 | [#67](https://github.com/pchkauu/want-keep/issues/67) |
