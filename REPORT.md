# REPORT.md — Отчёт по лабораторной работе №9

## Тема: Знакомство с масштабированием веб-приложений на примере Kubernetes

---

## 1. Цель работы

Познакомиться с процессом горизонтального масштабирования веб-приложений на примере Kubernetes: изучить понятия liveness/readiness probes, реализовать health-эндпоинты, создать K8s-манифесты для всех сервисов и реализовать распределённую блокировку на Redis.

---

## 2. Выполненные задачи

### 2.1 Health-эндпоинты

Добавлены два эндпоинта в `internal/health/probes.go`:

**`GET /health/live`** — liveness зонд. Минимальная проверка: процесс жив, цикл событий работает. Не обращается к внешним зависимостям — чтобы перезапуск контейнера не происходил из-за временной недоступности БД.

```json
{"status":"ok","timestamp":"2026-05-21T12:21:10Z"}
```

**`GET /health/ready`** — readiness зонд. Проверяет все зависимости с таймаутом 5 секунд:
- **MongoDB**: `db.RunCommand({ping: 1})` + измерение латентности
- **Redis**: `PING` + измерение латентности; если Redis отключён (`rdb == nil`) — статус `"disabled"`, не ломает readiness
- **RabbitMQ**: проверка `!amqpConn.IsClosed()`

```json
{
  "status": "ok",
  "timestamp": "2026-05-21T12:21:10Z",
  "checks": {
    "mongodb": {"status": "ok", "latency_ms": 0.306},
    "redis":   {"status": "ok", "latency_ms": 0.136},
    "rabbitmq": {"status": "ok"}
  }
}
```

При недоступности любой зависимости возвращается **503 Service Unavailable**.

### 2.2 Kubernetes-манифесты

Создана директория `k8s/` со следующей структурой:

```
k8s/
├── 00-namespace.yaml           # Namespace wp-labs
├── 01-mongodb/                 # StatefulSet: mongo:7, PVC 1Gi
├── 02-redis/                   # Deployment: redis:7-alpine
├── 03-minio/                   # StatefulSet: minio/minio, PVC 2Gi
├── 04-rabbitmq/                # StatefulSet: rabbitmq:3.12-management
└── 05-api/                     # Deployment: 2 реплики, configmap, secret
```

Особенности конфигурации:
- Все сервисы в одном Namespace `wp-labs`
- Stateful-сервисы (MongoDB, MinIO, RabbitMQ) — StatefulSet с PersistentVolumeClaim
- Redis и API — Deployment (stateless)
- Headless Services (`clusterIP: None`) для StatefulSet с `publishNotReadyAddresses: true`
- Cluster-local DNS вместо внешних адресов (`redis.wp-labs.svc.cluster.local`, и т.д.)
- ConfigMap для нечувствительной конфигурации, Secret — для паролей и ключей

### 2.3 Redis Distributed Lock

Реализован в `internal/cache/lock.go`. Используется в консьюмере RabbitMQ для защиты от дублированной обработки событий при нескольких репликах API.

**Алгоритм:**

1. Попытка захвата: `SET wp:lock:event:{eventId} {instanceID} NX EX 30` — атомарная операция Redis, успешна только если ключ не существует (NX = Not eXists, EX 30 = TTL 30 секунд)
2. Если не захвачена → другой под уже обрабатывает → `Ack` + skip
3. Если захвачена → double-check идемпотентности → SMTP → `Ack`
4. Освобождение: атомарный Lua-скрипт (удаляет ключ только если владелец совпадает)

```lua
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("DEL", KEYS[1])
else
    return 0
end
```

Lua-скрипт выполняется атомарно — нет race condition между проверкой и удалением.

---

## 3. Ответы на контрольные вопросы

### 1. Что такое Kubernetes и какую проблему он решает?

Kubernetes (K8s) — платформа оркестрации контейнеров с открытым исходным кодом. Решает проблемы:

- **Масштабирование**: автоматическое добавление/удаление реплик по нагрузке
- **Самовосстановление**: перезапуск упавших контейнеров, замена нездоровых подов
- **Rolling updates**: обновление без даунтайма (один за одним, с откатом при ошибке)
- **Service discovery**: автоматическая DNS-регистрация сервисов внутри кластера
- **Балансировка нагрузки**: распределение трафика между репликами

До K8s приходилось вручную управлять VM, настраивать load balancer, писать скрипты деплоя. K8s декларирует желаемое состояние ("хочу 3 реплики API") — платформа сама его поддерживает.

---

### 2. В чём разница между Pod, Deployment и StatefulSet?

**Pod** — минимальная единица развёртывания. Группа контейнеров с общим сетевым пространством и томами. Поды эфемерны: при сбое создаётся новый. Напрямую поды почти не используют.

**Deployment** — контроллер для **stateless**-приложений:
- Управляет набором одинаковых, взаимозаменяемых подов
- Rolling update, rollback, масштабирование
- При пересоздании — новый pod с новым именем и IP
- Используется для: API-серверов, фронтенда, worker'ов

**StatefulSet** — контроллер для **stateful**-приложений (БД, очереди):
- Стабильные имена подов: `rabbitmq-0`, `rabbitmq-1`, ...
- Каждый под привязан к **своему** PersistentVolumeClaim
- Упорядоченный запуск и остановка (0 → 1 → 2)
- Pod-специфичный DNS: `rabbitmq-0.rabbitmq.namespace.svc.cluster.local`
- Используется для: MongoDB, Redis Cluster, RabbitMQ, MinIO, ZooKeeper

> **Важно:** StatefulSet не делает автоматическую репликацию данных. `replicas: 3` для MongoDB создаст 3 независимых экземпляра, не replica set. Репликация данных — задача самого приложения (MongoDB Replica Set, Redis Sentinel, и т.д.).

---

### 3. Какую роль играет Service? Типы и отличия.

Service — абстракция, обеспечивающая стабильный сетевой доступ к группе подов. Поды появляются и умирают, меняют IP — Service остаётся постоянным.

| Тип | Описание | Когда использовать |
|-----|----------|--------------------|
| `ClusterIP` (default) | Виртуальный IP внутри кластера | Внутренние сервисы (API → Redis) |
| `ClusterIP: None` (Headless) | Нет виртуального IP, DNS возвращает IP подов напрямую | StatefulSet, прямой DNS |
| `NodePort` | Открывает порт на каждом узле кластера (30000-32767) | Разработка, доступ без ingress |
| `LoadBalancer` | Создаёт внешний балансировщик (в облаке) | Продакшен в AWS/GCP/Azure |
| `ExternalName` | CNAME на внешний DNS | Доступ к внешним сервисам |

В проекте:
- API, Redis — `ClusterIP` (внутренние)
- MongoDB, MinIO, RabbitMQ — Headless (`ClusterIP: None`) для pod-специфичного DNS

---

### 4. Как представлены секреты в Kubernetes? Меры безопасности.

**Secret** — объект K8s для хранения чувствительных данных. Значения хранятся в base64-кодировании (не шифровании — base64 легко декодируется).

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: api-secret
type: Opaque
data:
  JWT_ACCESS_SECRET: eW91cl9zZWNyZXQ=   # base64("your_secret")
```

**Меры безопасности:**

1. **Не коммитить Secret-файлы с реальными данными в Git** — использовать `.gitignore`, хранить в менеджере секретов (HashiCorp Vault, AWS Secrets Manager)
2. **Включить шифрование etcd at rest** — по умолчанию etcd хранит Secrets в открытом виде
3. **RBAC** — ограничить `get`/`list` для Secret только нужным сервис-аккаунтам
4. **External Secrets Operator** — синхронизация из внешних хранилищ (рекомендуется для продакшена)
5. **Rotation** — регулярная смена секретов

В учебном проекте Secrets содержат закодированные значения, но это допустимо для локальной разработки.

---

### 5. Горизонтальное vs вертикальное масштабирование. Что поддерживает kubectl scale?

**Вертикальное масштабирование (Scale Up)** — увеличение ресурсов одного экземпляра (CPU, RAM). Требует перезапуска, ограничено возможностями железа.

**Горизонтальное масштабирование (Scale Out)** — увеличение числа экземпляров. Не требует перезапуска существующих, теоретически не ограничено. Требует, чтобы приложение было stateless.

`kubectl scale` поддерживает **горизонтальное** масштабирование:

```bash
kubectl scale deployment/api --replicas=4 -n wp-labs
```

Для автоматического горизонтального масштабирования по метрикам — HorizontalPodAutoscaler (HPA):

```bash
kubectl autoscale deployment/api --min=2 --max=6 --cpu-percent=70 -n wp-labs
```

---

### 6. Разница между livenessProbe и readinessProbe. Почему их не следует делать идентичными?

| | livenessProbe | readinessProbe |
|--|--------------|----------------|
| **Вопрос** | Процесс жив? | Готов принимать трафик? |
| **Действие при провале** | Перезапуск контейнера | Исключение из балансировки Service |
| **Что проверяет** | Минимум: процесс не завис | Все зависимости: БД, кеш, очередь |
| **Частота** | Редко, дёшево | Часто, допускает нагрузку |

**Почему не следует делать их идентичными:**

Если liveness проверяет подключение к MongoDB — при временной недоступности БД Kubernetes начнёт **перезапускать поды**. Это усугубит ситуацию: пока поды перезапускаются, они создают дополнительную нагрузку. Правильно — исключить под из балансировки (readiness) и дождаться восстановления БД, а не перезапускать рабочий процесс.

В проекте:
- `/health/live` — только проверка HTTP (процесс живой)
- `/health/ready` — MongoDB ping + Redis PING + RabbitMQ IsClosed()

---

### 7. Как Kubernetes определяет, что под недоступен для трафика?

1. readinessProbe выполняется периодически (каждые `periodSeconds`)
2. При `failureThreshold` последовательных провалах — под помечается как **NotReady**
3. Service снимает IP пода из списка Endpoints — трафик больше не маршрутизируется на этот под
4. После `successThreshold` успехов подряд — под возвращается в ротацию

Также под никогда не получает трафик, если:
- Контейнер ещё не прошёл startupProbe
- Pod не прошёл readinessProbe хотя бы один раз после старта

Scheme: `kubelet → readinessProbe → kube-proxy / iptables → Service Endpoints`.

---

### 8. Что такое Namespace и зачем он нужен в многопользовательском кластере?

**Namespace** — логическая изоляция ресурсов внутри одного кластера K8s.

Применения:
- **Разделение окружений**: `dev`, `staging`, `production` на одном кластере
- **Мультитенантность**: разные команды или клиенты в одном кластере
- **Квоты**: ResourceQuota — ограничить CPU/RAM на namespace
- **RBAC**: разные права для разных команд в разных namespace
- **Изоляция DNS**: `redis.dev.svc.cluster.local` и `redis.prod.svc.cluster.local` — разные сервисы

В проекте все ресурсы в `wp-labs`:

```bash
kubectl get all -n wp-labs
```

По умолчанию поды в одном namespace могут обращаться к сервисам в другом namespace через полный DNS: `service.other-namespace.svc.cluster.local`.

---

### 9. Почему масштабирование stateful-сервисов через репликацию подов не гарантирует консистентность данных?

StatefulSet с `replicas: 3` создаёт 3 **независимых** экземпляра одного приложения, каждый со **своим** PVC. Они не образуют кластер автоматически.

**Пример с MongoDB:**
- `mongodb-0`, `mongodb-1`, `mongodb-2` — три отдельных MongoD-процесса
- Запись в `mongodb-0` **не реплицируется** в `mongodb-1` и `mongodb-2`
- Это три независимые БД, не MongoDB Replica Set

Для настоящей репликации нужна специфичная конфигурация самого приложения:
- **MongoDB**: Replica Set (`rs.initiate()`, `rs.add()`)
- **Redis**: Redis Sentinel или Redis Cluster
- **RabbitMQ**: Quorum Queues + clustering через erlang cookie и rabbitmq-plugins
- **MinIO**: MinIO Distributed Mode (erasure coding)

В лабораторной работе используется 1 реплика каждого stateful-сервиса — это допустимо для локальной разработки и упрощённого деплоя.

---

### 10. Условия для корректной работы распределённой блокировки в Redis

**Базовые условия:**

1. **Атомарность захвата**: `SET key value NX EX ttl` — одна команда, неделима. Нельзя делать `EXISTS` + `SET` отдельными командами: race condition.

2. **Уникальный идентификатор владельца**: значение блокировки должно быть уникальным для каждого клиента (`instanceID = hostname + uuid`). Нужно, чтобы не освободить чужую блокировку.

3. **Атомарное освобождение**: Lua-скрипт `GET → compare → DEL`. Нельзя делать `GET` + `DEL` отдельными командами: между ними TTL может истечь и другой клиент успеет захватить блокировку.

4. **TTL (Time-To-Live)**: блокировка должна автоматически истекать. Если процесс упал — блокировка освободится через TTL, не навсегда.

5. **Идемпотентность операции**: даже если блокировка истекла и другой под начал обработку — повторная обработка не должна нарушать инварианты (double-check через Redis idempotency key).

**Ограничения в нашей реализации:**

- Используется один инстанс Redis (не Redis Cluster). При падении Redis распределённая блокировка перестаёт работать, но graceful degradation: `rdb == nil` → no-op, обработка продолжается без блокировки.
- Для продакшена с требованиями к durability — Redlock (алгоритм для нескольких независимых Redis-инстансов).

---

## 4. Итоги

В ходе работы:

1. Реализованы health-эндпоинты `/health/live` и `/health/ready` для интеграции с K8s probes
2. Созданы K8s-манифесты для 5 сервисов (MongoDB, Redis, MinIO, RabbitMQ, API) + Namespace
3. API задеплоен с 2 репликами, RollingUpdate, startup/liveness/readiness probes
4. Реализован Redis distributed lock (SET NX EX + Lua CAS unlock) в консьюмере RabbitMQ
5. Проверена корректность health-зондов и горизонтального масштабирования

Приложение успешно работает в Kubernetes с возможностью горизонтального масштабирования API без изменения кода — благодаря stateless-архитектуре и идемпотентной обработке событий.

---

## 5. Структура изменённых файлов

| Файл | Изменения |
|------|-----------|
| `internal/health/probes.go` | Новый файл: liveness и readiness handlers |
| `internal/cache/lock.go` | Новый файл: Redis distributed lock |
| `internal/cache/keys.go` | +`EventLockKey()` для lock-ключей |
| `internal/messaging/consumer.go` | +distributed lock в handler, +instanceID |
| `cmd/server/main.go` | +health routes, +instanceID, +distLock |
| `k8s/` | 17 новых YAML-манифестов |
| `README.md` | Обновлён под ЛР9 |
| `DEPLOYMENT.md` | Новый: руководство по деплою + разбор ошибок |
| `REPORT.md` | Новый: отчёт и ответы на вопросы |
