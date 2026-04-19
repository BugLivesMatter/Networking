# ЛР6: MongoDB vs PostgreSQL — Отличия, изменения в коде и ответы на вопросы

## 1. Особенности MongoDB по сравнению с PostgreSQL

| Характеристика | PostgreSQL | MongoDB |
|---|---|---|
| **Модель данных** | Реляционная (таблицы, строки, столбцы) | Документоориентированная (коллекции, документы BSON) |
| **Схема** | Фиксированная (DDL, миграции) | Гибкая (schema-less), валидация опциональна |
| **Связи** | Внешние ключи (FK), JOIN | Ссылки по полю + ручной lookup; либо вложенные документы |
| **Первичный ключ** | Авто-инкремент / UUID (DDL) | `_id` — ObjectId или любой тип, в т.ч. UUID |
| **Транзакции** | ACID-транзакции «из коробки» | Multi-document транзакции с v4.0 (WiredTiger) |
| **Индексы** | B-Tree, Hash, GIN, GiST и др. | Single-field, Compound, Text, TTL, Geospatial, Hashed |
| **Запросы** | SQL (стандарт ISO) | MQL (MongoDB Query Language), Aggregation Pipeline |
| **Масштабирование** | Вертикальное + репликация; шардинг сложнее | Горизонтальное шардинг «из коробки» |
| **Soft Delete** | `deleted_at IS NULL` в SQL | `"deleted_at": null` в BSON-фильтре |
| **ORM/ODM (Go)** | GORM (`gorm.io/gorm`) | Официальный драйвер `go.mongodb.org/mongo-driver/v2` |
| **Миграции** | SQL-файлы через `golang-migrate` | Не требуются (schema-less); индексы создаются программно |
| **Хранение данных** | Строки в таблицах (heap pages) | BSON-документы (Binary JSON) |

---

## 2. Изменения в коде

### 2.1 Зависимости (`go.mod`)

| До (ЛР5) | После (ЛР6) |
|---|---|
| `gorm.io/gorm` | `go.mongodb.org/mongo-driver/v2` |
| `gorm.io/driver/postgres` | — |
| `github.com/golang-migrate/migrate/v4` | — |

### 2.2 Конфигурация (`internal/config/config.go`)

Заменены поля `DBHost`, `DBPort`, `DBUser`, `DBPassword`, `DBName` на:
- `MongoURI` — строка подключения (`mongodb://user:pass@host:port/db?authSource=admin`)
- `MongoDBName` — имя базы данных

Удалены методы `DSN()` и `MigrationDSN()`.

### 2.3 Инфраструктура (`docker-compose.yml`)

- Сервис `postgres:16` → `mongo:7`
- Переменные окружения: `POSTGRES_*` → `MONGO_INITDB_*`
- Volume: `wp_labs_postgres` → `wp_labs_mongo`

### 2.4 Domain-модели

Из всех моделей:
- Убраны GORM-теги (`gorm:"..."`)
- Убраны методы `TableName()` и `BeforeCreate()` (GORM-хуки)
- Убран импорт `gorm.io/gorm`
- Добавлены BSON-теги (`bson:"..."`)
- `gorm.DeletedAt` заменён на `*time.Time` с тегом `bson:"deleted_at,omitempty"`
- UUID хранится как тип `[16]byte` в поле `_id` (MongoDB Binary)

**Пример:**
```go
// До (GORM)
type Category struct {
    ID        uuid.UUID      `gorm:"type:uuid;primaryKey"`
    DeletedAt gorm.DeletedAt `gorm:"index"`
}

// После (MongoDB)
type Category struct {
    ID        uuid.UUID  `bson:"_id"`
    DeletedAt *time.Time `bson:"deleted_at,omitempty" json:"-"`
}
```

### 2.5 Репозитории (`internal/*/repository/`)

| Аспект | GORM (PostgreSQL) | mongo-driver v2 |
|---|---|---|
| Конструктор | `NewRepo(db *gorm.DB)` | `NewRepo(col *mongo.Collection)` |
| Поиск одной записи | `db.Where(...).First(&obj)` | `col.FindOne(ctx, filter).Decode(&obj)` |
| «Не найдено» | `gorm.ErrRecordNotFound` | `mongo.ErrNoDocuments` |
| Список | `db.Offset().Limit().Find()` | `col.CountDocuments()` + `col.Find(..., opts)` |
| Создание | `db.Create(&obj)` | `col.InsertOne(ctx, obj)` |
| Обновление | `db.Save(&obj)` | `col.UpdateOne(ctx, filter, bson.M{"$set": obj})` |
| Soft Delete | `db.Delete(&obj, id)` | `col.UpdateOne(ctx, filter, bson.M{"$set": bson.M{"deleted_at": now}})` |
| Фильтр soft delete | `WHERE deleted_at IS NULL` | `bson.M{"deleted_at": nil}` |
| Preload (JOIN) | `db.Preload("Category")` | Отдельный `FindOne` по `category_id` в методе `fillCategory()` |

UUID генерируется в самом репозитории (не в GORM-хуке `BeforeCreate`).

### 2.6 Инициализация индексов (`internal/database/mongodb.go`)

Создан новый пакет `database`, заменяющий SQL-миграции. При старте сервера вызывается `EnsureIndexes(ctx, db)`, который создаёт:

- `categories`: sparse-индекс на `deleted_at`
- `products`: составной индекс `category_id + deleted_at`
- `users`: unique-индексы на `email`; sparse-unique на `phone`, `yandex_id`, `vk_id`
- `refresh_tokens`: unique на `token_hash`; sparse-unique на `access_token_hash`; индекс на `user_id`
- `password_reset_tokens`: unique на `token`; индекс на `user_id`

### 2.7 Health/Diagnosis (`internal/health/diagnosis.go`)

- Параметр `*gorm.DB` заменён на `*mongo.Database`
- `SELECT 1` заменён на `db.RunCommand(ctx, bson.D{{Key: "ping", Value: 1}})`
- Тип ответа `DiagnosisPostgresSection` → `DiagnosisMongoSection`
- JSON-ключ `"postgresql"` → `"mongodb"`

### 2.8 Dockerfile

Убраны строки:
```dockerfile
COPY internal/migrations /app/internal/migrations
COPY --from=builder /app/internal/migrations ./internal/migrations
```

---

## 3. Ответы на контрольные вопросы

### Вопрос 1. Чем отличаются реляционные и документоориентированные СУБД?

**Реляционные БД** (PostgreSQL, MySQL) хранят данные в нормализованных таблицах со строгой схемой. Связи между таблицами реализуются через внешние ключи, запросы пишутся на SQL. Обеспечиваются ACID-гарантии на уровне отдельной транзакции. Хорошо масштабируются вертикально.

**Документоориентированные БД** (MongoDB) хранят данные в виде JSON/BSON-документов произвольной структуры внутри коллекций. Нет фиксированной схемы — разные документы в одной коллекции могут иметь разные поля. Связи реализуются через ссылки (поле с ID) или встраивание (embedded document). Горизонтальное масштабирование (шардинг) встроено. Схема валидируется опционально на уровне приложения или через MongoDB Schema Validation.

### Вопрос 2. Что такое BSON? Чем он отличается от JSON?

**BSON (Binary JSON)** — бинарный формат сериализации документов, используемый MongoDB для хранения и передачи данных. Отличия от JSON:

| | JSON | BSON |
|---|---|---|
| Формат | Текстовый (UTF-8) | Бинарный |
| Типы данных | string, number, bool, null, array, object | JSON-типы + Date, Binary, ObjectId, Decimal128, Int32/Int64, Regex и др. |
| Размер | Компактнее при простых структурах | Больше из-за бинарных заголовков, но содержит типовую информацию |
| Скорость парсинга | Медленнее (текст → структура) | Быстрее (бинарное представление напрямую читается) |
| Порядок полей | Не гарантирован | Гарантирован (поля сохраняются в порядке записи) |

BSON позволяет MongoDB эффективно хранить данные с точными типами (например, различать `int32` и `int64`) и поддерживать специальные типы, которых нет в JSON (например, `Date` без приведения к строке, `ObjectId`).

### Вопрос 3. Встраивание (embedding) или ссылки (references) — когда что использовать?

**Встраивание** (embedded document) — данные вложены прямо в родительский документ:
```json
{
  "_id": "...",
  "name": "Product",
  "category": { "name": "Electronics", "status": "active" }
}
```
**Применять когда:**
- Данные всегда читаются вместе (нет смысла делать отдельный запрос)
- Связь «один к одному» или «один ко многим» с небольшим числом вложенных элементов
- Вложенные данные не имеют самостоятельного жизненного цикла
- Размер документа не превышает 16 МБ

**Ссылки** (references) — хранение ID связанной сущности:
```json
{ "_id": "...", "name": "Product", "category_id": "..." }
```
**Применять когда:**
- Одни данные используются во многих документах (нормализация)
- Размер вложенных данных может сделать документ слишком большим
- Вложенные сущности имеют самостоятельный жизненный цикл (обновляются независимо)
- Связь «многие ко многим»

В данном проекте выбраны **ссылки**: `Product.CategoryID` хранит UUID категории, а поле `Product.Category` заполняется отдельным запросом в `fillCategory()` (аналог JOIN).

### Вопрос 4. Как MongoDB обеспечивает целостность данных без внешних ключей?

MongoDB не имеет встроенных внешних ключей (FK) и каскадных операций. Целостность данных обеспечивается:

1. **На уровне приложения** — сервисный слой проверяет существование связанной записи перед созданием зависимой. Например, перед созданием продукта сервис проверяет, что категория с `categoryID` существует и не удалена.

2. **Через транзакции** (MongoDB 4.0+) — при необходимости атомарно обновлять связанные коллекции.

3. **Через дизайн данных** — использование встраивания вместо ссылок там, где нужна атомарность.

4. **Schema Validation** — MongoDB поддерживает JSON Schema для валидации документов при вставке/обновлении.

В данном проекте целостность обеспечена сервисным слоем (`category/service/service.go` проверяет при удалении категории, есть ли связанные продукты через `CountByCategoryID`).

### Вопрос 5. Что такое валидация схем в MongoDB?

MongoDB поддерживает **Schema Validation** через JSON Schema (с v3.6) и MongoDB-специфичные операторы. Валидация задаётся при создании коллекции или командой `collMod`:

```js
db.createCollection("products", {
  validator: {
    $jsonSchema: {
      bsonType: "object",
      required: ["name", "price", "category_id"],
      properties: {
        name: { bsonType: "string", minLength: 1 },
        price: { bsonType: "double", minimum: 0 },
        status: { enum: ["available", "unavailable"] }
      }
    }
  },
  validationAction: "error"  // или "warn"
})
```

`validationAction: "error"` — операции вставки/обновления с некорректными документами отклоняются.  
`validationAction: "warn"` — операции пропускаются, но предупреждение логируется.

В данном проекте валидация реализована на уровне Go-сервисов (через DTO с тегами `binding:"required"` Gin-валидатора), что достаточно для учебного проекта.

### Вопрос 6. Почему в MongoDB нет JOIN? Как работает `$lookup`?

MongoDB — документоориентированная СУБД, данные хранятся денормализовано: все нужные для одного запроса данные обычно находятся в одном документе. JOIN не нужен, если данные встроены (embedded). Для ссылочных данных используется `$lookup` в Aggregation Pipeline — он выполняет аналог LEFT JOIN:

```js
db.products.aggregate([
  {
    $lookup: {
      from: "categories",
      localField: "category_id",
      foreignField: "_id",
      as: "category"
    }
  },
  { $unwind: "$category" }
])
```

В данном проекте вместо `$lookup` используется два отдельных запроса в репозитории: сначала `Find` продуктов, затем для каждого продукта `FindOne` категории (метод `fillCategory`). Это простее в реализации, хотя и менее эффективно при большом количестве продуктов (N+1 проблема). При необходимости оптимизации можно перейти на `$lookup` в Aggregation Pipeline.

### Вопрос 7. Какие типы индексов существуют в MongoDB?

| Тип | Описание | Пример использования |
|---|---|---|
| **Single Field** | Индекс по одному полю | `{email: 1}` — поиск по email |
| **Compound** | Составной индекс по нескольким полям | `{category_id: 1, deleted_at: 1}` — фильтр по категории и мягкому удалению |
| **Multikey** | Автоматически при индексировании массива | Поиск по тегам `{tags: 1}` |
| **Text** | Полнотекстовый поиск | `{description: "text"}` — поиск по словам в описании |
| **Geospatial (2dsphere/2d)** | Геопространственные запросы | Поиск ближайших точек |
| **Hashed** | Равномерное распределение для шардинга | `{_id: "hashed"}` |
| **Sparse** | Индексирует только документы с данным полем | `{phone: 1}, sparse: true` — не индексирует документы без `phone` |
| **Unique** | Запрещает дублирование значений | `{email: 1}, unique: true` |
| **TTL** | Автоматическое удаление документов по истечении времени | `{created_at: 1}, expireAfterSeconds: 3600` |
| **Partial** | Индексирует только документы, удовлетворяющие условию | `{status: 1}, partialFilterExpression: {status: "active"}` |

В данном проекте используются: Single Field, Compound, Unique, Sparse (и их комбинации).

### Вопрос 8. Как работает уникальность в MongoDB?

Уникальность обеспечивается через **уникальный индекс** (`unique: true`). MongoDB гарантирует, что в коллекции не может быть двух документов с одинаковым значением индексируемого поля. При попытке вставки дубликата возвращается ошибка `DuplicateKey (E11000)`.

**Sparse + Unique**: если поле может отсутствовать в ряде документов (опциональное), используется комбинация `unique + sparse`. Без `sparse: true` MongoDB считает отсутствующее поле равным `null`, и несколько документов без этого поля нарушат уникальный индекс. С `sparse: true` документы без данного поля не попадают в индекс.

Пример из проекта:
```go
// email — обязателен, уникален
options.Index().SetUnique(true)

// phone — опционален, но уникален если присутствует
options.Index().SetUnique(true).SetSparse(true)
```

В отличие от PostgreSQL, где `UNIQUE CONSTRAINT` создаёт индекс автоматически, в MongoDB уникальность всегда обеспечивается явным индексом.
