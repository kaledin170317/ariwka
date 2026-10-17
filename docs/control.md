# Control, Telegram и PostgreSQL

Все три сервиса — в `/etc/control/compose.yml`. База остаётся в существующем
томе `control_postgres-data`; `task control-deploy` её не перезапускает.

Первый запуск с машины разработчика (нужны Docker/Buildx и `docker login` с
правом записи в `kaledin170317/vpn`):

```sh
task control-publish
task telegram-publish
task control-init
```

До запуска проверьте локальные `.env` (секреты control и бота) и `.env.node`
(`DOCKERHUB_TOKEN` с правом чтения). Оба файла должны быть доступны только вам
(`chmod 600 .env .env.node`). **На сервере ничего останавливать вручную не надо**:
`control-init` скачает образы, применит миграции, остановит и отключит старые
`control.service` и `telegram.service` и запустит контейнеры. Старые файлы
на сервере автоматически не удаляются. PostgreSQL и данные сохраняются.
Перед миграциями рекомендуется сделать резервную копию БД.

Проверка на control-сервере:

```sh
docker ps --filter label=com.docker.compose.project=control
```

Следующие обновления: опубликуйте изменившийся образ (`task control-publish`
или `task telegram-publish`) и выполните `task control-deploy`. Контейнер с
неизменившимся образом и конфигурацией не пересоздаётся. Для новых версий
образы всегда публикуются под отдельными тегами `control-latest` и
`telegram-latest` одного приватного репозитория (один общий `latest` для
разных образов использовать нельзя).

Общие команды: `task publish` параллельно собирает все четыре образа, `task init`
последовательно запускает сначала ноды, затем control, а `task deploy`
параллельно обновляет ноды и control-сервер. Если ноды уже настроены и нужен
только переход control с systemd, используйте **`task control-init`**, а не
общий `task init`.

Локальная `.env` не копируется на сервер: Ansible передаёт переменные в Compose
при запуске. Уже созданные контейнеры сохраняют их при перезапуске Docker;
администратор Docker может прочитать их через `docker inspect`.
