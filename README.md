# Лабораторные работы №2–9: REST API + Kubernetes

REST API на Go (Gin + MongoDB + Redis + MinIO + RabbitMQ) с JWT/OAuth2-аутентификацией, CRUD-ресурсами, файловым хранилищем, асинхронной обработкой событий и Kubernetes-деплоем.

---

## Стек технологий

| Компонент | Технология |
|-----------|-----------|
| Язык / фреймворк | Go 1.22, Gin |
| База данных | MongoDB 7 |
| Кеш / сессии | Redis 7 |
| Объектное хранилище | MinIO |
| Брокер сообщений | RabbitMQ 3.12 |
| Оркестрация | Kubernetes (Docker Desktop) |
| Документация API | Swagger (swaggo) |

---

## Что реализовано по лабораторным

| ЛР | Что добавлено |
|----|--------------|
| ЛР2 | CRUD категорий и продуктов, пагинация, soft delete |
| ЛР3 | JWT (access + refresh), OAuth2 Яндекс, bcrypt, logout / logout-all, сброс пароля |
| ЛР4 | Swagger UI (`/api/docs`), генерация OpenAPI через swaggo |
| ЛР5 | Redis: cache-aside для списков, JTI-инвалидация access-токенов, AOF |
| ЛР6 | Переход с PostgreSQL на MongoDB, Repository Pattern, индексы через код |
| ЛР7 | MinIO: загрузка/скачивание файлов, метаданные в MongoDB, кеш в Redis |
| ЛР8 | RabbitMQ: публикация события при регистрации, SMTP welcome-email, DLQ, идемпотентность |
| ЛР9 | K8s health-зонды, манифесты для всех сервисов, Redis distributed lock, горизонтальное масштабирование |

---

## Быстрый старт (Docker Compose)

```bash
cp .env.example .env
# заполнить .env реальными значениями
docker-compose up --build
```

API: **http://localhost:4200**  
Swagger: **http://localhost:4200/api/docs/index.html** (при `APP_ENV=development`)  
RabbitMQ UI: **http://localhost:15672**

### Остановка

```bash
docker-compose down        # остановить
docker-compose down -v     # остановить + удалить тома (данные будут потеряны)
```

---

## Переменные окружения

```env
# === MongoDB ===
MONGO_URI=mongodb://admin:secret@mongodb:27017/wp_labs?authSource=admin
MONGO_DB_NAME=wp_labs
MONGO_ROOT_USER=admin
MONGO_ROOT_PASSWORD=secret

# === JWT ===
JWT_ACCESS_SECRET=your_access_secret_key_min_32_chars
JWT_REFRESH_SECRET=your_refresh_secret_key_min_32_chars
JWT_ACCESS_EXPIRATION=15m
JWT_REFRESH_EXPIRATION=168h

# === OAuth2 Yandex ===
YANDEX_CLIENT_ID=your_yandex_client_id
YANDEX_CLIENT_SECRET=your_yandex_client_secret
YANDEX_CALLBACK_URL=http://localhost:4200/auth/oauth/yandex/callback

# === Redis ===
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=redis_secure_password_change_in_prod
CACHE_TTL_DEFAULT=300
CACHE_ENABLED=true

# === MinIO ===
MINIO_ENDPOINT=minio:9000
MINIO_ACCESS_KEY=minio_admin
MINIO_SECRET_KEY=minio_secure_password_change_in_prod
MINIO_BUCKET=wp-labs-files
MINIO_USE_SSL=false
MAX_FILE_SIZE=10485760

# === RabbitMQ ===
RABBITMQ_USER=student
RABBITMQ_PASS=student_secure_rabbit_pass_change_in_prod
QUEUE_USER_REGISTERED=wp.auth.user.registered

# === SMTP ===
SMTP_HOST=smtp.yandex.com
SMTP_PORT=465
SMTP_USER=your_email@yandex.ru
SMTP_PASS=your_app_password
SMTP_FROM=your_email@yandex.ru
SMTP_SECURE=true
APP_PUBLIC_URL=http://localhost:4200

# === App ===
APP_ENV=development
PORT=4200
```

> Не коммитьте `.env` с реальными секретами.

---

## ЛР9: Kubernetes

### Требования

- Docker Desktop с включённым Kubernetes (Settings → Kubernetes → Enable Kubernetes)
- kubectl
- Локальный Docker-реестр запущен: `docker run -d -p 5000:5000 --restart=always --name registry registry:2`

### Сборка и публикация образа

```bash
# Собрать и запушить в локальный реестр (доступен K8s-ноде через host.docker.internal)
docker build -t host.docker.internal:5000/api:1.0.0 .
docker push host.docker.internal:5000/api:1.0.0
```

Для образов из Docker Hub, которые не тянутся напрямую, используем тот же приём:

```bash
docker pull rabbitmq:3.12-management-alpine
docker tag rabbitmq:3.12-management-alpine host.docker.internal:5000/rabbitmq:3.12-management-alpine
docker push host.docker.internal:5000/rabbitmq:3.12-management-alpine
```

### Настройка секретов

Перед деплоем обновите пароли в `k8s/*/secret.yaml` и `k8s/05-api/secret.yaml`:

| Параметр | Файл |
|----------|------|
| `MONGO_ROOT_PASSWORD` / `MONGO_URI` | `k8s/01-mongodb/secret.yaml` |
| `REDIS_PASSWORD` | `k8s/02-redis/secret.yaml` |
| `MINIO_ROOT_PASSWORD` | `k8s/03-minio/secret.yaml` |
| `RABBITMQ_DEFAULT_PASS` + erlang cookie | `k8s/04-rabbitmq/secret.yaml` |
| JWT, SMTP, OAuth, MinIO key и др. | `k8s/05-api/secret.yaml` |

### Деплой

```bash
kubectl apply -f k8s/00-namespace.yaml
kubectl apply -f k8s/01-mongodb/ \
             -f k8s/02-redis/ \
             -f k8s/03-minio/ \
             -f k8s/04-rabbitmq/ \
             -f k8s/05-api/

# Ожидание готовности
kubectl rollout status deployment/api -n wp-labs

# Проброс порта
kubectl port-forward svc/api 4200:4200 -n wp-labs
```

### Проверка

```bash
# Health-зонды
curl http://localhost:4200/health/live
# {"status":"ok","timestamp":"..."}

curl http://localhost:4200/health/ready
# {"status":"ok","checks":{"mongodb":{"status":"ok"},"rabbitmq":{"status":"ok"},"redis":{"status":"ok"}}}

# Swagger
open http://localhost:4200/api/docs/index.html
```

### Горизонтальное масштабирование

```bash
kubectl scale deployment/api --replicas=4 -n wp-labs
kubectl get pods -n wp-labs -o wide

# Автомасштабирование по CPU
kubectl autoscale deployment/api --min=2 --max=6 --cpu-percent=70 -n wp-labs
```

### Distributed Lock — проверка

При `replicas=4` зарегистрируйте нового пользователя. В логах подов увидите:

```
# Под, захвативший блокировку:
консьюмер RabbitMQ: попытка отправки письма ... instanceID=<hostname>:<uuid>

# Остальные поды:
консьюмер RabbitMQ: блокировка занята другим экземпляром, пропуск (eventId=...)
```

Событие обработает ровно один под.

```bash
# Просмотр логов всех подов api с префиксом
kubectl logs -l app=api -n wp-labs --prefix

# RabbitMQ Management UI
kubectl port-forward svc/rabbitmq 15672:15672 -n wp-labs
# http://localhost:15672 (student / из secret.yaml)
```

### Очистка

```bash
kubectl delete namespace wp-labs
```

---

## Эндпоинты

| Метод | Путь | Описание |
|-------|------|----------|
| `GET` | `/health/live` | K8s liveness зонд |
| `GET` | `/health/ready` | K8s readiness зонд (MongoDB + Redis + RabbitMQ) |
| `GET` | `/health/redis` | Статус Redis и метрики кеша |
| `GET` | `/health/diagnosis` | Сравнение латентности MongoDB vs Redis |
| `POST` | `/auth/register` | Регистрация (→ событие в RabbitMQ → welcome email) |
| `POST` | `/auth/login` | Вход, выдача JWT в cookies |
| `POST` | `/auth/refresh` | Обновление пары токенов |
| `POST` | `/auth/logout` | Выход (инвалидация JTI) |
| `POST` | `/auth/logout-all` | Выход со всех устройств |
| `GET` | `/auth/whoami` | Профиль из JWT + кеш |
| `GET/POST` | `/profile` | Просмотр / обновление профиля |
| `POST` | `/files` | Загрузка файла (multipart/form-data) |
| `GET` | `/files/:id` | Скачивание файла |
| `DELETE` | `/files/:id` | Удаление файла |
| `GET/POST/PUT/PATCH/DELETE` | `/categories` | CRUD категорий |
| `GET/POST/PUT/PATCH/DELETE` | `/products` | CRUD продуктов |

Полная схема — в Swagger UI: **http://localhost:4200/api/docs/index.html**

---

## Структура K8s-манифестов

```
k8s/
├── 00-namespace.yaml
├── 01-mongodb/        secret + service (headless) + statefulset
├── 02-redis/          secret + service + deployment
├── 03-minio/          secret + service (headless) + statefulset
├── 04-rabbitmq/       secret + service (headless, publishNotReadyAddresses) + statefulset
└── 05-api/            configmap + secret + deployment (replicas=2) + service
```

---

## Перегенерация Swagger

```bash
go run github.com/swaggo/swag/cmd/swag@latest init -g cmd/server/main.go -o docs
```

---

## Яндекс SMTP (ЛР8)

Ошибка `535 access rights` при AUTH означает проблему на стороне Яндекса, не в коде:

1. В настройках ящика включить **"Почтовые программы"** → IMAP + пароли приложений
2. Создать пароль приложения в [Яндекс ID → Безопасность](https://id.yandex.ru/security)
3. `SMTP_USER` и `SMTP_FROM` должны совпадать (или содержать один домен)
4. `SMTP_HOST=smtp.yandex.com`, `SMTP_PORT=465`, `SMTP_SECURE=true`
5. После правки `.env` пересоздать контейнер: `docker compose up -d --build --force-recreate app`

---

## MongoDB: проверка данных

```bash
docker exec -it wp_labs_mongo mongosh -u admin -p secret --authenticationDatabase admin
```

```js
use wp_labs
db.categories.find({ deleted_at: null })
db.users.getIndexes()
```

В MongoDB Compass: `mongodb://admin:secret@localhost:27017/?authSource=admin`

---

## Redis: проверка кеша

```bash
docker exec -it wp_labs_redis redis-cli -a "<REDIS_PASSWORD>"
KEYS wp:*
GET wp:categories:list:page:1:limit:10
TTL wp:auth:user:<uuid>:access:<jti>
```
