-- ============================================
-- Откат миграции: Удаление таблицы продуктов
-- Описание: Удаляет индексы и таблицу products
-- ============================================

-- Индекс для фильтрации по мягкому удалению
DROP INDEX IF EXISTS idx_products_deleted_at;

-- Индекс для быстрого соединения с таблицей categories
DROP INDEX IF EXISTS idx_products_category_id;

-- Индекс для фильтрации по статусу
DROP INDEX IF EXISTS idx_products_status;

-- Индекс для поиска по названию продукта
DROP INDEX IF EXISTS idx_products_name;

-- Удаляем таблицу products
-- внешний ключ на categories удаляется автоматически вместе с таблицей
DROP TABLE IF EXISTS products;