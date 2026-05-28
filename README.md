# E-Commerce Platform

Учебный проект интернет-магазина на микросервисах. В нем есть корзина, управление заказами и товарами, уведомления о статусах заказа, PostgreSQL, Kafka и фронтенд.

## Что внутри

- `cart/` - сервис корзины. Хранит товары пользователя, оформляет checkout и обращается в `loms`.
- `loms/` - сервис заказов, товаров и остатков. Создает заказы, меняет статусы, резервирует остатки и публикует события в Kafka через outbox.
- `notifications/` - сервис уведомлений. Читает события из Kafka и отправляет callback во внешнюю систему.
- `frontend/` - React + Vite интерфейс: каталог, корзина, оформление, оплата и отмена заказа.
- `pkg/` - общий Go-код и сгенерированные protobuf/gRPC/gateway файлы.
- `docs/` - swagger-спеки и картинки для документации.
- `integration-tests/` - интеграционные тесты.
- `docker-compose*.yml` - окружение для локального запуска, CI и тестов.
- `Taskfile.yml` - основные команды проекта.

## Технологии

- Go `1.26.2`
- PostgreSQL `17`
- Kafka `7.6.1`
- gRPC + grpc-gateway
- React, TypeScript, Vite, Tailwind CSS
- Docker Compose

## Быстрый запуск

Нужны Docker, Docker Compose, Go, Node.js/npm и `go-task`.

Установить `task`, если его нет:

```bash
go install github.com/go-task/task/v3/cmd/task@latest
```

Запустить backend:

```bash
task backend
```

В отдельном терминале запустить frontend:

```bash
task frontend
```

Открыть frontend:

```text
http://localhost:5173
```

## Основные адреса

- Frontend: `http://localhost:5173`
- Cart HTTP gateway: `http://localhost:8080`
- Cart gRPC: `localhost:50051`
- LOMS HTTP gateway: `http://localhost:8081`
- LOMS gRPC: `localhost:50052`
- Notifications gRPC: `localhost:50053`
- PostgreSQL: `localhost:5432`
- Kafka для локальных клиентов: `localhost:9093`
- Kafdrop: `http://localhost:9000`

В `docker-compose.yml` также поднимаются вторые инстансы `cart-2` и `loms-2`, они нужны для проверок и сценариев с несколькими сервисами.

## Полезные команды

Запустить backend в Docker:

```bash
task backend
```

Остановить контейнеры и удалить volumes:

```bash
task down
```

Запустить frontend:

```bash
task frontend
```

Запустить интеграционные тесты:

```bash
task test
```

Запустить линтер:

```bash
task lint
```

Перегенерировать protobuf/gRPC/swagger/sqlc-код:

```bash
task generate
```

Если зависимости генерации уже установлены:

```bash
task fast-generate
```

## Конфигурация

Основная конфигурация задается переменными окружения в `docker-compose.yml`.

- `POSTGRES_HOST`, `POSTGRES_PORT`, `POSTGRES_DB`, `POSTGRES_USER`, `POSTGRES_PASSWORD` - подключение к PostgreSQL.
- `LOMS_GRPC_ADDR` - адрес LOMS для сервиса корзины.
- `KAFKA_BROKERS` - брокеры Kafka внутри Docker-сети.
- `KAFKA_NOTIFICATIONS_TOPIC` - topic уведомлений, по умолчанию `order_status_notifications`.
- `KAFKA_CONSUMER_GROUP` - consumer group сервиса `notifications`.
- `CALLBACK_ADDR` - внешний адрес для отправки уведомлений.
- `OUTBOX_WORKERS`, `OUTBOX_BATCH_SIZE`, `OUTBOX_FETCH_PERIOD`, `OUTBOX_IN_PROGRESS_TTL` - настройки outbox worker в `loms`.

## Как это работает

1. Пользователь добавляет товары в корзину через `frontend`.
2. `cart` хранит корзину и при checkout отправляет заказ в `loms`.
3. `loms` создает заказ, резервирует остатки и пишет outbox-событие.
4. Outbox worker публикует изменение статуса заказа в Kafka.
5. `notifications` читает Kafka topic и отправляет callback по `CALLBACK_ADDR`.
