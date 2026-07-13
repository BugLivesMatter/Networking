# NeuroOps — план развития

## Статус разработки

- [x] Интерактивная 3D-сцена и liquid-glass dashboard.
- [x] Трёхмерный куб кластера с workload-ячейками и близкими репликами.
- [x] WebGL compatibility view для устройств без аппаратного 3D.
- [x] Браузерный demo cluster с четырьмя chaos-сценариями.
- [x] Go-контракт `ClusterSource`, REST snapshot, SSE и scenario API.
- [x] Переключение frontend между local simulation и Go API.
- [x] CI и публикация автономной демки на GitHub Pages.
- [ ] Авторизация и incident workspace в новом интерфейсе.
- [ ] `KubernetesClusterSource` и реальные health/metrics adapters.
- [ ] Публичный backend deployment и observability stack.

## Цель

Превратить учебный REST API в **NeuroOps** — интерактивный центр наблюдения за микросервисным кластером. Публичная главная страница визуализирует состояние кластера в 3D; после входа пользователь получает рабочее пространство для инцидентов, bug reports и расследований.

## Пользовательский опыт

На главной странице — управляемая WebGL-сцена в чёрно-белом стиле с панелями «жидкого стекла». Логический сервис — ядро, его pod'ы — нейроны, а зависимости — пульсирующие связи. Цвет используется только для состояния: серый/белый — норма, жёлтый — деградация, красный — ошибка, синий — запуск новой реплики. По клику открываются readiness, latency, uptime, рестарты и последние события; снизу остаётся компактная таблица сервисов.

В demo-режиме посетитель может запускать безопасные сценарии: **Simulate Redis latency**, **Crash API pod**, **Scale API** и **Recover cluster**. Это должно менять сцену и health-данные через тот же поток событий, что и живой кластер.

## Архитектура: demo без переписывания для Kubernetes

Backend получает единый контракт источника состояния:

```go
type ClusterSource interface {
    Topology(ctx context.Context) (Topology, error)
    Subscribe(ctx context.Context) (<-chan ClusterEvent, error)
}
```

- `DemoClusterSource` генерирует реалистичный кластер и управляемые сбои.
- `KubernetesClusterSource` позднее использует Kubernetes API (`client-go`) для Pod, Deployment и Service.
- Оба источника отдают одинаковые `Service`, `Instance`, `Dependency`, `HealthSnapshot` и `ClusterEvent`.
- Источник выбирается через `CLUSTER_SOURCE=demo|kubernetes`; frontend не зависит от его типа.

Состояние передаётся UI через REST для начального snapshot и SSE для обновлений. MongoDB хранит конфигурацию сервисов, историю и инциденты; Redis — быстрый snapshot, кеш и блокировки; RabbitMQ — доменные события и уведомления; MinIO — логи и другие вложения.

## Доменная модель

Постепенно заменить учебные `categories`/`products` на:

- `Service` — API, MongoDB, Redis, RabbitMQ или MinIO;
- `Instance` — pod/реплика сервиса;
- `HealthCheck` и `ClusterEvent` — проверки и смены состояния;
- `Incident`, `IncidentEvent` и `Report` — расследования, timeline и bug reports;
- `Attachment` — постмортемы, логи, скриншоты;
- роли: `viewer`, `responder`, `incident-manager`, `admin`.

## Этапы реализации

1. **Основа backend.** Создать модели сервисов и инстансов, REST endpoints `topology`, service details и события; не удалять работающую аутентификацию.
2. **Demo cluster.** Добавить API/Redis/MongoDB/RabbitMQ/MinIO с зависимостями, SLA-порогами и детерминированными сценариями деградации, падения, восстановления и масштабирования.
3. **Realtime.** Реализовать SSE, чтобы изменения состояния сразу отражались в UI. Новая реплика сначала синяя и становится healthy только после успешной проверки.
4. **3D dashboard.** В `frontend/` подключить React Three Fiber/Three.js: вращение и zoom камеры, нейроны, связи, частицы, hover/click, боковую карточку и нижнюю health-панель. Предусмотреть 2D fallback.
5. **Incident workspace.** Создание инцидента из события, severity, назначение responder, timeline, SLA, MinIO-вложения, email-уведомления и аудит.
6. **Live cluster.** Реализовать `KubernetesClusterSource`, read-only ServiceAccount и документированное переключение `CLUSTER_SOURCE`.
7. **Observability.** Структурные логи, correlation ID, Prometheus-метрики и Grafana dashboard; при необходимости подключить реальные метрики Prometheus вместо базовых health checks.

## Публичное демо и GitHub Actions

GitHub Actions не является постоянно работающим хостингом для Go API. Он должен запускать CI/CD: `gofmt` check, `go vet`, `go test ./...`, frontend build, Docker build, Trivy scan, проверку Kubernetes YAML и публикацию образа в GitHub Container Registry (`ghcr.io`).

Frontend публикуется на GitHub Pages. На первом шаге он может работать с локальным симулятором в браузере; для полного demo с аккаунтами, SSE и инцидентами Go backend разворачивается на отдельной платформе, а Actions автоматически доставляет туда образ из `ghcr.io`.

## Критерий первого релиза

Посетитель открывает GitHub Pages, управляет 3D-кластером, запускает четыре demo-сценария и видит изменение нод, зависимостей, latency и event timeline. Проект собирается и проверяется GitHub Actions, а README содержит архитектурную схему, ссылку на demo и объяснение инженерных решений.
