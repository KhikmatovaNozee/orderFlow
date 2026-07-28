# OrderFlow

REST API интернет-магазина на Go. Покупатель смотрит каталог и оформляет заказы,
продавец ведёт товары и обрабатывает заказы. Ключевое ограничение продукта —
пайплайн статусов заказа: разрешены только переходы `new → paid → shipped`,
отмена (`cancelled`) возможна только из `new`.

Учебный проект курса backend-разработки на Go. Пишется с нуля, без стартера.

## Стек

| Что | Чем |
|---|---|
| Язык | Go 1.26 |
| HTTP | Gin |
| БД | PostgreSQL 16 |
| Драйвер | pgx/v5 (пул соединений) |
| Авторизация | JWT (access + refresh), bcrypt |
| Логи | log/slog, JSON |
| Запуск | Docker Compose |
| CI | GitHub Actions |

## Быстрый старт

Нужен установленный Docker.

```bash
git clone https://github.com/KhikmatovaNozee/orderFlow.git
cd orderFlow
cp .env.example .env
docker compose up --build
```

Поднимутся два контейнера: приложение на `:8080` и PostgreSQL на `:5432`.
Таблицы создаются автоматически при старте, миграции запускать не нужно.

Проверить, что всё живо:

```bash
curl http://localhost:8080/health
```

```json
{"status":"ok","db":"up"}
```

Если база недоступна, `/health` вернёт `503` и `{"status":"unavailable","db":"down"}` —
это настоящая проверка соединения, а не заглушка.

Остановить:

```bash
docker compose down
```

## Переменные окружения

| Переменная | Обязательна | Значение по умолчанию | Описание |
|---|---|---|---|
| `DATABASE_URL` | да | — | Строка подключения к PostgreSQL |
| `JWT_SECRET` | да | — | Секрет для подписи access-токенов |
| `LOG_LEVEL` | нет | `info` | `debug` / `info` / `warn` / `error` |

Приложение не стартует, если `DATABASE_URL` или `JWT_SECRET` не заданы —
падает сразу с понятной ошибкой, а не позже посреди запроса.

Пример в `.env.example`. Для docker-compose значения уже прописаны в
`docker-compose.yml`, отдельный `.env` нужен только для запуска без докера.

## Роли

Роль выбирается при регистрации и хранится в JWT.

| Роль | Кто это | Что может |
|---|---|---|
| `user` | Покупатель | Смотреть каталог, оформлять и оплачивать заказы |
| `seller` | Продавец | Всё то же плюс управление своими товарами и заказами |

Покупательские действия доступны **любому** залогиненному пользователю,
включая продавца — отдельной проверки роли там нет, хватает аутентификации.
Проверка роли (`RequireRole`) стоит только на продавцовской группе `/manage`.

## Эндпоинты

Базовый префикс — `/api/v1`, кроме `/health`.

| Метод | Путь | Авторизация | Описание |
|---|---|---|---|
| `GET` | `/health` | — | Проверка сервиса и БД |
| `POST` | `/api/v1/auth/register` | — | Регистрация |
| `POST` | `/api/v1/auth/login` | — | Вход, выдаёт пару токенов |
| `POST` | `/api/v1/auth/refresh` | — | Обновление пары токенов |
| `POST` | `/api/v1/auth/logout` | — | Отзыв refresh-токена |
| `GET` | `/api/v1/protected` | Bearer | Проверка аутентификации |
| `GET` | `/api/v1/manage/test` | Bearer + `seller` | Проверка доступа продавца |

### Регистрация

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"login":"sevara","password":"password123","role":"user"}'
```

`201 Created`:

```json
{"id":1,"login":"sevara","role":"user","created_at":"2026-07-28T10:00:00Z"}
```

Пароль хранится как bcrypt-хэш и наружу не отдаётся никогда.
Занятый логин → `409`, короткий пароль или неизвестная роль → `400`.

### Вход

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"login":"sevara","password":"password123"}'
```

`200 OK`:

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "a1b2c3...",
  "token_type": "Bearer",
  "expires_in": 900
}
```

### Запрос с токеном

```bash
curl http://localhost:8080/api/v1/protected \
  -H "Authorization: Bearer <access_token>"
```

### Обновление и выход

```bash
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh":"<refresh_token>"}'
```

```bash
curl -X POST http://localhost:8080/api/v1/auth/logout \
  -H "Content-Type: application/json" \
  -d '{"refresh":"<refresh_token>"}'
```

Logout возвращает `204 No Content`.

Готовые запросы для всех ручек лежат в `test.http` — открываются прямо
из GoLand или VS Code с расширением REST Client.

## Как устроены токены

Токенов два, и это не усложнение ради усложнения.

**Access** живёт 15 минут, проверяется по подписи и нигде не хранится на
сервере. Если он утечёт (логи, XSS, история запросов) — окно ущерба узкое.

**Refresh** живёт дольше и лежит в таблице `refresh_tokens`, причём в виде
**хэша**, а не самого токена. Утечка дампа базы не даёт угнать сессии.
При каждом обновлении происходит ротация: старый токен отзывается
(`revoked_at`), выдаётся новый. Logout помечает refresh отозванным — по
чистому stateless-JWT выход из системы сделать нельзя, поэтому refresh
и держим в базе.

## Формат ответов

Все ошибки приходят в одинаковой форме:

```json
{"error": "описание"}
```

Соответствие доменных ошибок и HTTP-кодов задано в одном месте
(`internal/respond`), а не раскидано по хендлерам:

| Ошибка | Код |
|---|---|
| `ErrInvalid` | 400 |
| `ErrForbidden` | 403 |
| `ErrNotFound` | 404 |
| `ErrNoStock` | 409 |
| `ErrConflict` | 409 |
| всё остальное | 500 |

На `500` наружу всегда уходит нейтральное «internal server error» —
детали (SQL, имена таблиц) пишутся в логи, но не отдаются клиенту.

## Архитектура

```
cmd/server          точка входа, сборка зависимостей, graceful shutdown
internal/
  router            маршруты и подключение middleware
  handler           разбор запроса, коды ответов
  service           бизнес-логика
  repository        SQL, работа с БД
  middleware        Auth, RequireRole, логирование с request_id
  model             сущности и доменные ошибки
  respond           единый формат ответов и ошибок
  logger            общий JSON-логгер
```

Зависимости идут в одну сторону: `handler → service → repository`.
Хендлер не знает про SQL, сервис не знает про HTTP.

Репозитории спрятаны за интерфейсами (`UserRepository`,
`RefreshTokenRepository`), объявленными на стороне сервиса. Благодаря этому
в тестах база подменяется фейками — сервисный слой тестируется без докера
и без сети.

## Логи и остановка

Логи структурные, JSON, в stdout. Каждому запросу присваивается
`request_id`, он же возвращается в заголовке `X-Request-Id`. Если id пришёл
снаружи — переиспользуется, так запрос прослеживается через несколько
сервисов.

```json
{"time":"...","level":"INFO","msg":"request handled","request_id":"4899353d...","method":"GET","path":"/health","status":200,"duration":1063606}
```

Хендлеры и сервисы логируют через `logger.From(ctx)` — их строки
автоматически получают тот же `request_id`.

По `SIGINT`/`SIGTERM` сервис завершается корректно и в правильном порядке:

```
shutdown signal received
http server stopped accepting connections
closing database pool
server stopped cleanly
```

Сначала сервер перестаёт принимать новые соединения и дослуживает те
запросы, что уже в работе, и только потом закрывается пул БД. Наоборот
нельзя: недослуженный запрос полезет в базу, которой уже нет.

Проверить:

```bash
docker compose up -d
docker compose logs -f app
# в другом окне
docker compose stop app
```

## Тесты

```bash
go test ./...
```

Покрытие:

```bash
go test $(go list ./... | grep -Ev '/cmd/|/repository/') -coverprofile=coverage.out
go tool cover -func=coverage.out
```

Текущее покрытие — **81%**.

Из подсчёта исключены `cmd/` и `repository/`. Причина: в `cmd/` только
проводка приложения, а в `repository/` — голый SQL, который юнит-тестами
без живой БД не покрыть. По условию задачи репозитории и не тестируются
сами: они подменяются фейками, чтобы можно было проверить слой сервисов.
Работа репозиториев проверяется интеграционно, запуском в docker-compose.

Тесты табличные, живой БД не требуют — база везде заменена фейковыми
репозиториями.

## Разработка

```bash
go build ./...
go test ./...
golangci-lint run
```

Каждая задача делается в отдельной ветке (`feat/us-07`, `chore/mw-respond`)
и вливается через Pull Request. Прямой push в `main` не используется, PR
мержит не автор, а напарница после ревью.

На каждый PR автоматически запускаются:

- **Tests & Coverage** — build, vet, тесты, race detector, отчёт покрытия с порогом 70%
- **Lint** — golangci-lint
- **Nilaway** — поиск потенциальных nil-разыменований (информативно, не блокирует)

## Команда

| Участница | Зона ответственности |
|---|---|
| Нозима Хикматова | Каркас проекта, схема БД, регистрация, JWT + refresh, middleware |
| Севинч Иброхимова | Единый формат ответов, логирование и graceful shutdown, тесты, CI |
