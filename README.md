# Лабораторные работы №2–4: REST API с авторизацией и документацией Swagger

## Описание проекта

RESTful API на Go (Gin + PostgreSQL) с полным циклом аутентификации, CRUD-ресурсами и автоматической документацией OpenAPI (Swagger).

### Что реализовано

**Лабораторная работа №2 — CRUD REST API:**
- Ресурсы: категории (`/categories`) и продукты (`/products`)
- Полный CRUD с поддержкой пагинации (`page`, `limit`)
- Мягкое удаление (Soft Delete)
- Миграции базы данных через `golang-migrate`

**Лабораторная работа №3 — Авторизация и аутентификация:**
- Регистрация, вход, выход, сброс пароля
- JWT Access (15 мин) + Refresh (7 дней) токены с подписью HS256
- Передача токенов через `HttpOnly`, `SameSite` cookies
- Хеширование паролей: `bcrypt` с уникальной солью
- Refresh-токены в БД с возможностью отзыва (logout / logout-all)
- Авторизация через Яндекс (Authorization Code Grant, реализован вручную)
- CSRF-защита через параметр `state` в OAuth
- Все эндпоинты `/categories` и `/products` защищены JWT-middleware

**Лабораторная работа №4 — Swagger / OpenAPI документация:**
- Автоматическая генерация спецификации через `swaggo/swag` (Code-First подход)
- Все эндпоинты аннотированы: теги, описания, коды ответов, примеры
- Swagger UI доступен **только в режиме `development`** — при `APP_ENV=production` маршрут `/api/docs` возвращает 404
- Настроена схема безопасности `CookieAuth` (apiKey в cookie `access_token`)
- Чувствительные поля (пароли, соли, хеши токенов) исключены из схем ответов

---

## Быстрый старт

### 1. Клонирование репозитория

```bash
git clone https://github.com/BugLivesMatter/lab_2.git
cd lab_2
git checkout lab4/swagger-openapi
```

### 2. Настройка переменных окружения

Скопируйте `.env.example` в `.env` и при необходимости отредактируйте значения:

```bash
cp .env.example .env
```

Минимально необходимое содержимое `.env`:

```env
# === Database ===
DB_HOST=postgres
DB_PORT=5432
DB_USER=student
DB_PASSWORD=student
DB_NAME=wp_labs

# === JWT Secrets (мин. 32 символа) ===
JWT_ACCESS_SECRET=your_access_secret_key_min_32_chars
JWT_REFRESH_SECRET=your_refresh_secret_key_min_32_chars
JWT_ACCESS_EXPIRATION=15m
JWT_REFRESH_EXPIRATION=168h

# === OAuth2 Yandex ===
YANDEX_CLIENT_ID=your_yandex_client_id
YANDEX_CLIENT_SECRET=your_yandex_client_secret
YANDEX_CALLBACK_URL=http://localhost:4200/auth/oauth/yandex/callback

# === App environment ===
APP_ENV=development
```

> **Важно:** Никогда не коммитьте `.env` с реальными секретами. Используйте `.env.example` как шаблон.

### 3. Запуск через Docker

```bash
docker-compose up --build
```

Приложение будет доступно по адресу: `http://localhost:4200`

### 4. Остановка

```bash
docker-compose down
```

### 5. Полная очистка (удаление данных БД)

```bash
docker-compose down -v
```

---

## Swagger UI

Документация API доступна при `APP_ENV=development`:

**`http://localhost:4200/api/docs/index.html`**

> При `APP_ENV=production` маршрут `/api/docs` возвращает `404 Not Found`.

Для тестирования защищённых эндпоинтов:
1. Выполните `POST /auth/login` — cookies установятся автоматически в браузере
2. Нажмите **Authorize** в Swagger UI и введите значение `access_token` из cookie
3. Запросы к защищённым ресурсам (`/categories`, `/products`, `/auth/whoami`) будут проходить успешно

---

## Описание API

### Авторизация и аутентификация

| Метод | Эндпоинт | Описание | Доступ | Статус успеха |
|-------|----------|----------|--------|---------------|
| `POST` | `/auth/register` | Регистрация нового пользователя | Public | `201 Created` |
| `POST` | `/auth/login` | Вход (установка cookies) | Public | `200 OK` |
| `POST` | `/auth/refresh` | Обновление пары токенов | Public (Refresh Cookie) | `200 OK` |
| `GET` | `/auth/whoami` | Данные текущего пользователя | Private | `200 OK` |
| `POST` | `/auth/logout` | Завершение текущей сессии | Private | `200 OK` |
| `POST` | `/auth/logout-all` | Завершение всех сессий | Private | `200 OK` |
| `GET` | `/auth/oauth/:provider` | Инициация входа через OAuth | Public | `302 Redirect` |
| `GET` | `/auth/oauth/:provider/callback` | Обработка ответа от OAuth | Public | `200 OK` |
| `POST` | `/auth/forgot-password` | Запрос на сброс пароля | Public | `200 OK` |
| `POST` | `/auth/reset-password` | Установка нового пароля | Public | `200 OK` |

### Категории (требуют авторизации)

| Метод | Эндпоинт | Описание | Статус успеха |
|-------|----------|----------|---------------|
| `GET` | `/categories` | Список категорий с пагинацией | `200 OK` |
| `GET` | `/categories/:id` | Категория по ID | `200 OK` |
| `POST` | `/categories` | Создать категорию | `201 Created` |
| `PUT` | `/categories/:id` | Полное обновление категории | `200 OK` |
| `PATCH` | `/categories/:id` | Частичное обновление категории | `200 OK` |
| `DELETE` | `/categories/:id` | Мягкое удаление категории | `204 No Content` |

### Продукты (требуют авторизации)

| Метод | Эндпоинт | Описание | Статус успеха |
|-------|----------|----------|---------------|
| `GET` | `/products` | Список продуктов с пагинацией | `200 OK` |
| `GET` | `/products/:id` | Продукт по ID | `200 OK` |
| `POST` | `/products` | Создать продукт | `201 Created` |
| `PUT` | `/products/:id` | Полное обновление продукта | `200 OK` |
| `PATCH` | `/products/:id` | Частичное обновление продукта | `200 OK` |
| `DELETE` | `/products/:id` | Мягкое удаление продукта | `204 No Content` |

### Параметры пагинации

| Параметр | Тип | По умолчанию | Описание |
|----------|-----|--------------|----------|
| `page` | integer | `1` | Номер страницы (начинается с 1) |
| `limit` | integer | `10` | Записей на странице (макс. 100) |

```json
{
  "data": [ ... ],
  "meta": {
    "total": 100,
    "page": 1,
    "limit": 10,
    "totalPages": 10
  }
}
```

---

## Миграции

Применяются **автоматически** при запуске через `docker-compose up --build`.

Файлы миграций в `internal/migrations/`:
- `000001_create_categories_table`
- `000002_create_products_table`
- `000003_create_users_table`
- `000004_create_refresh_tokens_table`
- `000005_create_password_reset_tokens_table`

Отдельная команда не требуется — миграции запускаются в `runMigrations()` при старте сервера.

---

## Тестирование API через curl

### Регистрация
```bash
curl -X POST http://localhost:4200/auth/register \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"test@example.com\",\"password\":\"SecurePass123!\",\"phone\":\"+79991234567\"}"
```

### Вход
```bash
curl -X POST http://localhost:4200/auth/login \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"test@example.com\",\"password\":\"SecurePass123!\"}" \
  -c cookies.txt
```

### Проверка авторизации
```bash
curl http://localhost:4200/auth/whoami -b cookies.txt
```

### Список категорий
```bash
curl http://localhost:4200/categories -b cookies.txt
```
