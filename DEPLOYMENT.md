# DEPLOYMENT.md — Развёртывание в Kubernetes: разбор ошибок

Этот документ описывает реальные проблемы, возникшие при развёртывании проекта в локальном Kubernetes (Docker Desktop), и способы их решения.

---

## Окружение

- **Docker Desktop** с включённым Kubernetes (single-node)
- **Windows 11**, PowerShell
- **kubectl**, **k9s** для мониторинга

---

## Ошибка 1: kubectl: connection refused (localhost:8080)

### Симптом

```
E0521 ... dial tcp 127.0.0.1:8080: connect: connection refused
```

Любая команда `kubectl` падает с отказом соединения.

### Причина

Kubernetes не был включён в Docker Desktop. По умолчанию Docker Desktop устанавливает только Docker Engine, K8s-нода не поднимается.

### Решение

1. Открыть Docker Desktop
2. **Settings → Kubernetes → Enable Kubernetes**
3. Нажать **Apply & Restart**
4. Дождаться появления зелёного индикатора Kubernetes в трее

После перезапуска `kubectl config current-context` должен вернуть `docker-desktop`.

---

## Ошибка 2: ErrImageNeverPull для API-пода

### Симптом

```
Failed to pull image "wp-labs/api:1.0.0": rpc error: ... ErrImageNeverPull
```

API-поды зависают в статусе `ErrImageNeverPull`.

### Причина

Docker Desktop использует **два отдельных хранилища образов**:

- `moby` namespace — для Docker CLI (`docker build`, `docker run`)
- `k8s.io` namespace — для Kubernetes

Образ, собранный через `docker build`, **не виден** Kubernetes-ноде. `imagePullPolicy: Never` запрещает попытку скачать образ извне, но своя copy в `k8s.io` отсутствует.

> Попытка решить через отключение "Use containerd for pulling and storing images" в Docker Desktop — **не помогает** и может сломать другие вещи.

### Решение: локальный реестр как мост

```bash
# 1. Запустить локальный Docker Registry
docker run -d -p 5000:5000 --restart=always --name registry registry:2

# 2. Собрать образ сразу с тегом для локального реестра
docker build -t host.docker.internal:5000/api:1.0.0 .

# 3. Запушить в реестр
docker push host.docker.internal:5000/api:1.0.0
```

В манифесте `k8s/05-api/deployment.yaml` указать:

```yaml
image: host.docker.internal:5000/api:1.0.0
imagePullPolicy: Always
```

`host.docker.internal` — специальное DNS-имя Docker Desktop, которое резолвится как адрес хост-машины изнутри K8s-ноды. Kubernetes тянет образ из реестра на хосте через этот адрес.

Также добавить реестр в `daemon.json` как insecure (HTTP):

```json
"insecure-registries": ["host.docker.internal:5000", "localhost:5000"]
```

---

## Ошибка 3: ImagePullBackOff для RabbitMQ

### Симптом

```
Failed to pull image "rabbitmq:3.12-management-alpine": ... registry-mirror:1273 returned 500
```

RabbitMQ-под зависает в `ImagePullBackOff`. Внутренний зеркальный реестр Docker Desktop возвращает 500.

### Причина

Docker Desktop имеет встроенный кеширующий прокси (`registry-mirror`) для образов Docker Hub. Из-за нестабильного соединения с Docker Hub (или ограничений сети) прокси вернул ошибку 500.

### Решение

Скачать образ через Docker CLI (используя рабочие зеркала) и пробросить через локальный реестр:

```bash
# Добавить зеркала в daemon.json и применить
# "registry-mirrors": ["https://dockerhub.timeweb.cloud", "https://mirror.gcr.io"]

# Скачать образ через Docker (зеркала работают)
docker pull rabbitmq:3.12-management-alpine

# Переименовать и запушить в локальный реестр
docker tag rabbitmq:3.12-management-alpine host.docker.internal:5000/rabbitmq:3.12-management-alpine
docker push host.docker.internal:5000/rabbitmq:3.12-management-alpine
```

В `k8s/04-rabbitmq/statefulset.yaml` изменить:

```yaml
image: host.docker.internal:5000/rabbitmq:3.12-management-alpine
```

---

## Ошибка 4: Init:ImagePullBackOff для RabbitMQ

### Симптом

RabbitMQ-под застрял в `Init:ImagePullBackOff`.

### Причина

В исходном манифесте был init-контейнер на основе `busybox` для исправления прав на том:

```yaml
initContainers:
  - name: fix-permissions
    image: busybox
    command: ["chown", "-R", "999:999", "/var/lib/rabbitmq"]
```

`busybox` также не мог скачаться из-за недоступности Docker Hub.

### Решение

Убрать init-контейнер полностью. Права на том устанавливаются через `fsGroup` в `securityContext` пода — без дополнительного контейнера:

```yaml
spec:
  securityContext:
    fsGroup: 999   # GID пользователя rabbitmq внутри образа
  containers:
    - name: rabbitmq
      ...
```

Kubernetes автоматически меняет владельца тома на указанный `fsGroup` при монтировании — init-контейнер для этого не нужен.

---

## Ошибка 5: CrashLoopBackOff для API — RabbitMQ DNS не резолвится

### Симптом

```
dial tcp: lookup rabbitmq.wp-labs.svc.cluster.local on 10.96.0.10:53: no such host
```

или

```
dial tcp: lookup rabbitmq-0.rabbitmq.wp-labs.svc.cluster.local: no such host
```

API-под запускается, пытается подключиться к RabbitMQ, получает ошибку DNS и крашится.

### Причина

Headless Service в Kubernetes (`clusterIP: None`) регистрирует DNS-записи для подов только **после того, как под проходит readiness probe**. Пока RabbitMQ-под ещё стартует (Running, но не Ready), его DNS-имя не существует в кластере.

Попытка исправить через pod-специфичный DNS (`rabbitmq-0.rabbitmq.wp-labs.svc.cluster.local`) не помогла по той же причине — StatefulSet pod DNS тоже требует готовности пода.

### Решение

Добавить флаг `publishNotReadyAddresses: true` в Service RabbitMQ:

```yaml
# k8s/04-rabbitmq/service.yaml
spec:
  clusterIP: None
  publishNotReadyAddresses: true   # DNS регистрируется сразу, не ждёт readiness
  selector:
    app: rabbitmq
```

Теперь DNS-запись `rabbitmq-0.rabbitmq.wp-labs.svc.cluster.local` появляется сразу после создания пода — API может начать подключение, а RabbitMQ ещё продолжает инициализацию.

---

## Ошибка 6: Swagger не работает (404 или пустая страница)

### Симптом

Swagger UI по адресу `http://localhost:4200/api/docs/index.html` недоступен или возвращает 404.

### Причина

В приложении Swagger включён только при `APP_ENV != "production"`. В ConfigMap было:

```yaml
APP_ENV: "production"
```

### Решение

Изменить в `k8s/05-api/configmap.yaml`:

```yaml
APP_ENV: "development"
```

Применить изменения и перезапустить деплоймент:

```bash
kubectl apply -f k8s/05-api/configmap.yaml
kubectl rollout restart deployment/api -n wp-labs
kubectl rollout status deployment/api -n wp-labs
```

---

## Ошибка 7: Docker Hub недоступен (context deadline exceeded)

### Симптом

```
error response from daemon: Head "https://registry-1.docker.io/v2/...": context deadline exceeded
```

`docker build` или `docker pull` не могут скачать базовые образы.

### Решение

Добавить зеркала Docker Hub в `~/.docker/daemon.json`:

```json
{
  "registry-mirrors": [
    "https://dockerhub.timeweb.cloud",
    "https://mirror.gcr.io"
  ],
  "insecure-registries": [
    "host.docker.internal:5000",
    "localhost:5000"
  ]
}
```

Применить: Docker Desktop → Settings → Docker Engine → вставить JSON → **Apply & Restart**.

После перезапуска `docker pull golang:1.22-alpine` и другие образы должны скачиваться через зеркало.

---

## Общая последовательность деплоя (рабочая)

```bash
# 1. Убедиться, что K8s включён
kubectl cluster-info

# 2. Локальный реестр
docker run -d -p 5000:5000 --restart=always --name registry registry:2

# 3. Образы, которые не тянутся через K8s напрямую — через Docker + push
docker pull rabbitmq:3.12-management-alpine
docker tag rabbitmq:3.12-management-alpine host.docker.internal:5000/rabbitmq:3.12-management-alpine
docker push host.docker.internal:5000/rabbitmq:3.12-management-alpine

# 4. Сборка API
docker build -t host.docker.internal:5000/api:1.0.0 .
docker push host.docker.internal:5000/api:1.0.0

# 5. Деплой
kubectl apply -f k8s/00-namespace.yaml
kubectl apply -f k8s/01-mongodb/ -f k8s/02-redis/ -f k8s/03-minio/ -f k8s/04-rabbitmq/ -f k8s/05-api/

# 6. Мониторинг
kubectl get pods -n wp-labs -w

# 7. Проброс порта
kubectl port-forward svc/api 4200:4200 -n wp-labs

# 8. Проверка
curl http://localhost:4200/health/live
curl http://localhost:4200/health/ready
```

---

## Полезные команды отладки

```bash
# Статус всех ресурсов
kubectl get all -n wp-labs

# Детали пода (события, причина краша)
kubectl describe pod <pod-name> -n wp-labs

# Логи (последние 100 строк)
kubectl logs <pod-name> -n wp-labs --tail=100

# Предыдущий контейнер (при CrashLoop)
kubectl logs <pod-name> -n wp-labs --previous

# Войти в контейнер
kubectl exec -it <pod-name> -n wp-labs -- /bin/sh

# DNS-проверка изнутри пода
kubectl exec -it <api-pod-name> -n wp-labs -- nslookup rabbitmq-0.rabbitmq.wp-labs.svc.cluster.local

# Применить одно изменение и перезапустить
kubectl apply -f k8s/05-api/configmap.yaml
kubectl rollout restart deployment/api -n wp-labs
```
