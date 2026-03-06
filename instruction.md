# Инструкция по настройке и запуску API «Блеск» в Docker

Полная пошаговая инструкция по развёртыванию REST API магазина косметики «Блеск» с помощью Docker и Docker Compose.

---

## 1. Требования

Перед началом убедитесь, что установлены:

- **Docker** (Desktop или Engine). Рекомендуемая версия: 20.10 или новее.  
  Скачать: [https://docs.docker.com/desktop/](https://docs.docker.com/desktop/)
- **Docker Compose** (v2 или отдельная утилита `docker-compose`).  
  Установка: [https://docs.docker.com/compose/install/](https://docs.docker.com/compose/install/)

**Важно (Windows):** перед запуском `docker compose` должен быть **запущен Docker Desktop**. Если он не запущен, появится ошибка вида:  
`The system cannot find the file specified` / `open //./pipe/dockerDesktopLinuxEngine`.  
Откройте Docker Desktop и дождитесь полной загрузки (иконка в трее перестанет анимироваться), затем снова выполните команду.

Проверка в терминале:

```bash
docker --version
docker compose version
```

(или `docker-compose --version`, если используете отдельный бинарник). Команда `docker info` должна выполниться без ошибки — это подтвердит, что демон Docker запущен.

---

## 2. Получение кода проекта

Если проект ещё не склонирован:

```bash
git clone https://github.com/BugLivesMatter/lab_2.git
cd lab_2
```

Если вы уже в каталоге проекта (например, `lab_2`), переходите к следующему шагу.

---

## 3. Настройка переменных окружения

Переменные окружения задаются в файле `.env` в корне проекта. Docker Compose подставляет их в контейнеры.

### 3.1. Создание файла `.env`

В корне проекта (рядом с `docker-compose.yml`) создайте файл `.env`. Можно скопировать пример:

**Windows (PowerShell):**
```powershell
Copy-Item .env.example .env
```

**Windows (cmd):**
```cmd
copy .env.example .env
```

**Linux / macOS:**
```bash
cp .env.example .env
```

### 3.2. Редактирование `.env`

Откройте `.env` в редакторе и задайте значения. Минимальный набор:

```env
DB_USER=student
DB_PASSWORD=ваш_надёжный_пароль
DB_NAME=wp_labs
```

- **DB_USER** — пользователь PostgreSQL.  
- **DB_PASSWORD** — пароль пользователя (обязательно смените с примера).  
- **DB_NAME** — имя базы данных (можно оставить `wp_labs`).

Файл `.env` не должен попадать в Git (он в `.gitignore`). Не публикуйте его и не коммитьте пароли.

---

## 4. Запуск в Docker

### 4.1. Сборка и запуск

Из корня проекта выполните:

```bash
docker compose up --build
```

Или, если у вас установлен старый вариант CLI:

```bash
docker-compose up --build
```

- `--build` — перед запуском пересобрать образ приложения (нужно при первом запуске и после изменения кода).

Сервисы запустятся в текущем терминале; логи постgres и приложения будут выводиться в консоль.

### 4.2. Ожидаемый вывод

1. Сборка образа приложения (Go).
2. Запуск контейнера **postgres** (образ `postgres:16`).
3. Healthcheck PostgreSQL: контейнер БД считается готовым после успешной проверки `pg_isready`.
4. Запуск контейнера **app** (ваше API). Приложение подключится к БД, выполнит миграции (GORM AutoMigrate) и начнёт слушать порт 4200.

В логах должно появиться что-то вроде:

```
wp_labs_app  | server listening on :4200
```

### 4.3. Запуск в фоне

Чтобы контейнеры работали в фоне (без занятия терминала):

```bash
docker compose up --build -d
```

Логи тогда смотреть так:

```bash
docker compose logs -f app
```

---

## 5. Проверка работы

### 5.1. Доступность API

- **Базовый URL:** [http://localhost:4200](http://localhost:4200)

Проверка списка категорий (должен вернуться JSON с `data` и `meta`):

```bash
curl -s http://localhost:4200/categories
```

Или откройте в браузере: `http://localhost:4200/categories`.

### 5.2. Создание категории и товара

**Создать категорию:**

```bash
curl -X POST http://localhost:4200/categories ^
  -H "Content-Type: application/json" ^
  -d "{\"name\": \"Уход за лицом\", \"description\": \"Кремы и сыворотки\", \"status\": \"active\"}"
```

В ответе будет объект с полем `id` (UUID). Подставьте его в следующий запрос вместо `CATEGORY_UUID`.

**Создать товар** (подставьте свой `CATEGORY_UUID`):

```bash
curl -X POST http://localhost:4200/products ^
  -H "Content-Type: application/json" ^
  -d "{\"categoryId\": \"CATEGORY_UUID\", \"name\": \"Крем увлажняющий\", \"description\": \"50 мл\", \"price\": 990.50, \"status\": \"available\"}"
```

На Linux/macOS в командах `curl` используйте обратный слэш `\` вместо `^` для переноса строки и одинарные кавычки при необходимости.

### 5.3. База данных

PostgreSQL доступен на **localhost:5432**. Подключение из хоста:

- **Host:** localhost  
- **Port:** 5432  
- **User:** значение `DB_USER` из `.env`  
- **Password:** значение `DB_PASSWORD` из `.env`  
- **Database:** значение `DB_NAME` из `.env`

Можно использовать PgAdmin, DBeaver, DataGrip или `psql`. Таблицы `categories` и `products` создаются при первом старте приложения (миграции выполняются автоматически).

---

## 6. Остановка и очистка

### 6.1. Остановка контейнеров

Если контейнеры запущены в текущем терминале — нажмите `Ctrl+C`.

Если запущены в фоне:

```bash
docker compose down
```

### 6.2. Удаление данных БД (volume)

Чтобы удалить и данные PostgreSQL (volume), выполните:

```bash
docker compose down -v
```

После этого при следующем `docker compose up --build` база будет создана заново и миграции применятся снова.

---

## 7. Возможные проблемы и решения

### Ошибка «The system cannot find the file specified» / «pipe dockerDesktopLinuxEngine»

На Windows это обычно значит, что **Docker Desktop не запущен**. Запустите Docker Desktop из меню Пуск, дождитесь готовности (в трее отображается «Docker Desktop is running») и снова выполните `docker compose up --build`.

### Порт 4200 или 5432 уже занят

- Измените маппинг портов в `docker-compose.yml`, например:
  - для приложения: `"4201:4200"` — тогда API будет на [http://localhost:4201](http://localhost:4201);
  - для БД: `"5433:5432"` — тогда подключаться к PostgreSQL нужно на порт 5433.
- Или остановите процесс, который использует нужный порт.

### Приложение не подключается к БД при старте

- Убедитесь, что в `docker-compose.yml` у сервиса `app` указано `depends_on: postgres` с `condition: service_healthy`.
- Проверьте переменные в `.env`: `DB_HOST=postgres`, `DB_PORT=5432`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` — они передаются в контейнер `app` и должны совпадать с настройками контейнера `postgres`.

### Ошибка при сборке образа (go build)

- Убедитесь, что в проекте есть `go.mod` и `go.sum`.
- Выполните локально `go mod download` и снова запустите `docker compose up --build`.

### После изменения кода изменения не применяются

- Пересоберите образ без кэша:  
  `docker compose build --no-cache app`  
  затем `docker compose up -d`.

---

## 8. Краткая шпаргалка команд

| Действие | Команда |
|----------|---------|
| Первый запуск (сборка + запуск) | `docker compose up --build` |
| Запуск в фоне | `docker compose up --build -d` |
| Остановка | `docker compose down` |
| Остановка и удаление данных БД | `docker compose down -v` |
| Просмотр логов приложения | `docker compose logs -f app` |
| Просмотр логов БД | `docker compose logs -f postgres` |
| Список контейнеров | `docker compose ps` |

---

Миграции выполняются автоматически при старте приложения (GORM AutoMigrate). Отдельно запускать миграции не нужно.
