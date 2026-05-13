# Лабораторные работы №2–8: REST API с авторизацией, Swagger, Redis, MongoDB, MinIO и RabbitMQ

## Описание проекта

RESTful API на Go (Gin + **MongoDB** + Redis + **RabbitMQ**) с аутентификацией, CRUD-ресурсами, кешированием и автоматической документацией OpenAPI (Swagger).

> **ЛР6:** PostgreSQL полностью заменён на MongoDB 7. Подробнее об изменениях и отличиях двух СУБД — в [differences.md](differences.md).

### Что реализовано

**Лабораторная работа №8 — RabbitMQ и приветственное письмо:**
- Сервис RabbitMQ 3.12 с Management Plugin в `docker-compose` (порты `5672`, `15672`)
- Модуль `internal/messaging`: exchange `app.events` (direct), `app.dlx`, очередь `wp.auth.user.registered` с DLX, очередь `wp.auth.user.registered.dlq`
- После успешной регистрации (`POST /auth/register`) и при первой регистрации через OAuth публикуется JSON-событие `user.registered` (routing key `user.registered`), сообщения **persistent**, очереди **durable**
- В том же процессе что и HTTP-сервер запускается **фоновый консьюмер**: SMTP-отправка приветственного письма (plain + HTML), **Ack** только после успеха
- Повтор при ошибке SMTP: до **3** попыток с увеличением `metadata.attempt`, затем сообщение уходит в **DLQ** через `Nack(requeue=false)`
- **Идемпотентность** по `eventId`: ключ в Redis `wp:events:processed:{eventId}` с TTL 24 часа
- Модуль `internal/email`: конфигурация SMTP из `.env`, без логирования паролей
- При старте приложения выполняется **валидация** обязательных переменных RabbitMQ и SMTP

**Лабораторная работа №7 — MinIO Object Storage + файлы:**
- Добавлен сервис MinIO в `docker-compose` и конфигурация в `.env`
- Новый модуль `internal/file` (Controller/Service/Repository/Domain/DTO)
- Потоковая загрузка/скачивание файлов через MinIO (`io.Reader`, без полного буфера в памяти)
- Хранение только метаданных в MongoDB (`files`), бинарные данные — только в MinIO
- Защищенные эндпоинты `/files` (только авторизованный владелец)
- Кеш метаданных файла в Redis по ключу `wp:files:{fileId}:meta` (TTL 300 сек)
- Профиль пользователя вынесен в `/profile`, добавлены `displayName`, `bio`, `avatarFileId`

**Лабораторная работа №6 — MongoDB:**
- PostgreSQL заменён на MongoDB 7 (`mongo:7` в Docker)
- Хранение данных в коллекциях (документоориентированная модель)
- UUID в поле `_id` вместо SQL-первичного ключа
- Soft Delete через поле `deleted_at` (фильтр `{"deleted_at": null}`)
- Индексы создаются программно в `internal/database/mongodb.go` (заменяет SQL-миграции)
- Подключение через MongoDB URI (`MONGO_URI`)
- Минимальные изменения бизнес-логики за счёт Repository Pattern
- Диагностика: `GET /health/diagnosis` сравнивает латентность MongoDB vs Redis

**Лабораторная работа №2 — CRUD REST API:**
- Ресурсы: категории (`/categories`) и продукты (`/products`)
- Полный CRUD с поддержкой пагинации (`page`, `limit`)
- Мягкое удаление (Soft Delete)

**Лабораторная работа №3 — Авторизация и аутентификация:**
- Регистрация, вход, выход, сброс пароля
- JWT Access + Refresh с подписью HS256
- Передача токенов через `HttpOnly`, `SameSite` cookies
- Хеширование паролей: `bcrypt` с уникальной солью
- Refresh-токены в MongoDB (хеши), отзыв сессий (`logout` / `logout-all`)
- OAuth Яндекс (Authorization Code Grant)
- Защищённые маршруты `/categories` и `/products` через middleware

**Лабораторная работа №4 — Swagger / OpenAPI:**
- Генерация спецификации через `swaggo/swag`
- Swagger UI при `APP_ENV=development` (`/api/docs`)
- Схема `CookieAuth` (cookie `access_token`)

**Лабораторная работа №5 — Redis: кеш и сессии:**
- Redis в `docker-compose` с паролем (`REDIS_PASSWORD`) и **AOF** (`--appendonly yes`, том `wp_labs_redis`)
- Модуль `internal/cache`: клиент, **`cache.Service`** (`Get` / `Set` / `Del` / `DelByPattern` / `Exists`), JSON-сериализация, опциональное отключение (`CACHE_ENABLED`)
- **Cache-Aside:** `GET /categories`, `GET /products` — ключи с префиксом `wp:`, TTL из `CACHE_TTL_DEFAULT` (по умолчанию 300 с)
- Кеш профиля для `GetUserByID` / `whoami` — ключ `wp:users:profile:{userId}`
- **Инвалидация** списков при `POST` / `PUT` / `PATCH` / `DELETE` по соответствующим ресурсам (`DelByPattern` по спискам)
- **JTI access** в Redis: ключ `wp:auth:user:{userId}:access:{jti}`, значение `"valid"`, TTL = срок жизни access JWT
- В таблице `refresh_tokens` хранится **`access_jti`** для явного **`Del`** ключа JTI в Redis при **refresh** (до выдачи новой пары)
- Middleware: проверка подписи JWT → **`Exists` по JTI** (при ошибке Redis — переход к проверке по БД) → поиск активной сессии по **хэшу access** в БД

**Диагностика (лабораторная / мониторинг):**
- **`GET /health/redis`** — PING, INFO, DBSIZE, метрики обращений приложения к кешу (`RedisStatusResponse`)
- **`GET /health/diagnosis`** — сравнение латентности MongoDB и Redis на том же пути данных, что **`GET /categories`** (параметры `page`, `limit`; перед замером выполняется `Del` ключа страницы — см. поле `notes` в ответе)

---

## Быстрый старт

### 1. Клонирование репозитория

```bash
git clone https://github.com/BugLivesMatter/Networking.git
cd Networking
git checkout main
```

### 2. Переменные окружения

```bash
cp .env.example .env
```

Минимальный набор (см. также `.env.example`):

```env
# === MongoDB ===
MONGO_URI=mongodb://admin:secret@mongodb:27017/wp_labs?authSource=admin
MONGO_DB_NAME=wp_labs
MONGO_ROOT_USER=admin
MONGO_ROOT_PASSWORD=secret

# === JWT (секреты не короче 32 символов) ===
JWT_ACCESS_SECRET=your_access_secret_key_min_32_chars
JWT_REFRESH_SECRET=your_refresh_secret_key_min_32_chars
JWT_ACCESS_EXPIRATION=15m
JWT_REFRESH_EXPIRATION=168h

# === OAuth2 Yandex ===
YANDEX_CLIENT_ID=your_yandex_client_id
YANDEX_CLIENT_SECRET=your_yandex_client_secret
YANDEX_CALLBACK_URL=http://localhost:4200/auth/oauth/yandex/callback

# === Redis / Cache ===
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=redis_secure_password_change_in_prod
CACHE_TTL_DEFAULT=300
CACHE_ENABLED=true

# === MinIO / Object Storage ===
MINIO_ENDPOINT=minio:9000
MINIO_ACCESS_KEY=minio_admin
MINIO_SECRET_KEY=minio_secure_password_change_in_prod
MINIO_BUCKET=wp-labs-files
MINIO_USE_SSL=false
MAX_FILE_SIZE=10485760

# === RabbitMQ (ЛР8) ===
RABBITMQ_USER=student
RABBITMQ_PASS=student_secure_rabbit_pass_change_in_prod
QUEUE_USER_REGISTERED=wp.auth.user.registered

# === SMTP (ЛР8) ===
SMTP_HOST=smtp.yandex.ru
SMTP_PORT=465
SMTP_USER=your_email@yandex.ru
SMTP_PASS=your_app_password
SMTP_FROM=your_email@yandex.ru
SMTP_SECURE=true
APP_PUBLIC_URL=http://localhost:4200

# === App ===
APP_ENV=development
```

> Не коммитьте `.env` с реальными секретами.

### 3. Запуск

```bash
docker-compose up --build
```

API: **`http://localhost:4200`**

### 4. Остановка / полная очистка

```bash
docker-compose down
docker-compose down -v   # удалит тома MongoDB, Redis, MinIO и RabbitMQ
```

---

## Redis: проверка кеша и сессий

Подключение к Redis в контейнере:

```bash
docker exec -it wp_labs_redis redis-cli -a "<REDIS_PASSWORD>"
```

(пароль из `.env` / `docker-compose`.)

Примеры:

```bash
# В тестах допустимо; в продакшене на больших данных KEYS блокирует Redis — лучше SCAN
KEYS wp:*
GET wp:categories:list:page:1:limit:10
TTL wp:auth:user:<uuid>:access:<jti>
DEL wp:categories:list:page:1:limit:10
```

Массовое удаление по шаблону в приложении делается через **`SCAN` + `UNLINK`** (см. `cache.Service.DelByPattern`).

Проверка **logout / JTI:**
1. Войти, убедиться в наличии ключа `wp:auth:user:<userId>:access:<jti>`
2. `POST /auth/logout` с cookies
3. Запрос к защищённому ресурсу со старым access → **401**

После **перезапуска** контейнера Redis данные частично восстанавливаются за счёт **AOF** и тома; при `docker-compose down -v` кеш и JTI теряются.

---

## RabbitMQ (ЛР8)

### Management UI

- URL: **http://localhost:15672**
- Логин и пароль: значения `RABBITMQ_USER` и `RABBITMQ_PASS` из `.env`.

### Схема обмена сообщениями

```mermaid
flowchart LR
  subgraph http [HTTP]
    Register[POST /auth/register]
  end
  subgraph app [Процесс app]
    Producer[Publisher internal/messaging]
    Consumer[Consumer горутина]
    SMTP[SMTP welcome email]
  end
  subgraph broker [RabbitMQ]
    Ex[app.events direct]
    Q[wp.auth.user.registered]
    DLX[app.dlx]
    DLQ[wp.auth.user.registered.dlq]
  end
  subgraph cache [Redis]
    Idem[wp:events:processed:eventId]
  end
  Register --> Producer
  Producer -->|"user.registered persistent"| Ex
  Ex --> Q
  Q --> Consumer
  Consumer --> SMTP
  Consumer --> Idem
  Q -->|Nack no requeue после 3 попыток| DLX
  DLX --> DLQ
```

### Очереди и ключи

| Имя | Назначение |
|-----|------------|
| `app.events` | Direct exchange для доменных событий |
| `user.registered` | Routing key → очередь регистраций |
| `wp.auth.user.registered` | Основная очередь обработки |
| `app.dlx` | Dead letter exchange |
| `wp.auth.user.registered.dlq` | Очередь «ядовитых» / исчерпавших попытки сообщений |

### Ручная проверка ЛР8 (чек-лист)

1. Заполнить в `.env` **реальные** SMTP-параметры (пароль приложения для почтового ящика).
2. `docker compose up --build` — дождаться healthy у `rabbitmq` и `app`.
3. **Happy path:** `POST /auth/register` с новым email → в логах `app` строка об публикации `eventId` → в UI очереди `wp.auth.user.registered` сообщение исчезает после обработки → на почте **одно** приветственное письмо.
4. **Идемпотентность:** в Management UI опубликовать в `app.events` с routing key `user.registered` копию JSON с тем же `eventId`, что уже обработан → второе письмо **не** приходит.
5. **Retry и DLQ:** задать неверный `SMTP_PASS`, выполнить одну регистрацию → в логах три попытки для одного события → сообщение оказывается в `wp.auth.user.registered.dlq`.
6. **Накопление:** сломать `SMTP_HOST`, сделать несколько регистраций → счётчик Ready в основной очереди растёт; починить SMTP и перезапустить `app` → сообщения обрабатываются.

---

## Swagger

При **`APP_ENV=development`**:

**`http://localhost:4200/api/docs/index.html`**

Перегенерация спецификации (из корня репозитория):

```bash
go run github.com/swaggo/swag/cmd/swag@latest init -g cmd/server/main.go -o docs
```

---

## Эндпоинты (кратко)

| Метод | Путь | Описание |
|-------|------|----------|
| | **Health** | |
| `GET` | `/health/redis` | Статус Redis и метрики кеша |
| `GET` | `/health/diagnosis` | Диагностика: та же цепочка, что `GET /categories` (query: `page`, `limit`) |
| | **Auth** | |
| `POST` | `/auth/register`, `/auth/login`, `/auth/refresh`, … | Как в Swagger |
| `GET` | `/auth/whoami` | Профиль (с кешированием) |
| | **Profile** (JWT) | |
| `GET` | `/profile` | Получение текущего профиля |
| `POST` | `/profile` | Обновление профиля и `avatarFileId` |
| | **Files** (JWT) | |
| `POST` | `/files` | Загрузка файла (multipart/form-data) |
| `GET` | `/files/:fileId` | Скачивание файла (только владелец) |
| `DELETE` | `/files/:fileId` | Soft Delete метаданных + удаление объекта из MinIO |
| | **Ресурсы** (JWT) | |
| `GET` … `DELETE` | `/categories`, `/products` | CRUD + пагинация на списках |

Подробные схемы запросов/ответов — в **`docs/swagger.json`** / Swagger UI.

---

## MongoDB: проверка данных

Подключение к MongoDB в контейнере через `mongosh`:

```bash
docker exec -it wp_labs_mongo mongosh -u admin -p secret --authenticationDatabase admin
```

Пример команд:

```js
use wp_labs
db.categories.find({ deleted_at: null })
db.products.find({ category_id: <uuid-binary> })
db.users.getIndexes()
```

В MongoDB Compass: подключиться по `mongodb://admin:secret@localhost:27017/?authSource=admin`.

---

## Индексы (заменяют SQL-миграции)

Создаются автоматически при старте приложения (`database.EnsureIndexes` в `internal/database/mongodb.go`).

| Коллекция | Поле(я) | Тип |
|-----------|---------|-----|
| `categories` | `deleted_at` | sparse |
| `products` | `category_id`, `deleted_at` | compound |
| `users` | `email` | unique |
| `users` | `phone`, `yandex_id`, `vk_id` | unique + sparse |
| `users` | `avatar_file_id` | sparse |
| `files` | `user_id`, `deleted_at` | compound |
| `files` | `object_key` | index |
| `refresh_tokens` | `token_hash` | unique |
| `refresh_tokens` | `access_token_hash` | unique + sparse |
| `refresh_tokens` | `user_id` | index |
| `password_reset_tokens` | `token` | unique |
| `password_reset_tokens` | `user_id` | index |

---

## Примеры curl

### Вход и whoami

```bash
curl -X POST http://localhost:4200/auth/login \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"test@example.com\",\"password\":\"SecurePass123!\"}" \
  -c cookies.txt

curl http://localhost:4200/auth/whoami -b cookies.txt
```

### Список категорий

```bash
curl "http://localhost:4200/categories?page=1&limit=10" -b cookies.txt
```

### Загрузка и скачивание файла

```bash
curl -X POST http://localhost:4200/files \
  -b cookies.txt \
  -F "file=@avatar.png"

curl -X GET http://localhost:4200/files/<fileId> \
  -b cookies.txt \
  -o downloaded_avatar.png
```

### Обновление профиля с аватаром

```bash
curl -X POST http://localhost:4200/profile \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d "{\"displayName\":\"Иван Иванов\",\"bio\":\"Backend разработчик\",\"avatarFileId\":\"<fileId>\"}"
```

### Регистрация (публикация события в RabbitMQ, ЛР8)

После успешного ответа **201** в логах контейнера `app` появится строка об опубликованном `eventId`; консьюмер отправит приветственное письмо на указанный email.

```bash
curl -X POST http://localhost:4200/auth/register \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"newuser@example.com\",\"password\":\"SecurePass123!\",\"phone\":\"\"}"
```

### Health

```bash
curl http://localhost:4200/health/redis
curl "http://localhost:4200/health/diagnosis?page=1&limit=10"
```
