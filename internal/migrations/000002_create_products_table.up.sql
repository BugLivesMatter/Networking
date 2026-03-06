-- ============================================
-- Миграция: Создание таблицы продуктов
-- Описание: Создаёт таблицу products с внешним ключом на categories
-- ============================================

-- Создаём таблицу products
CREATE TABLE products (
    -- Уникальный идентификатор продукта (UUID)
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Название продукта (обязательное поле)
    name VARCHAR(255) NOT NULL,
    
    -- Описание продукта (текст, может быть пустым)
    description TEXT,
    
    -- Цена продукта (должна быть >= 0)
    price DECIMAL(10, 2) NOT NULL CHECK (price >= 0),
    
    -- Ссылка на категорию (внешний ключ)
    -- При удалении категории все её продукты тоже удаляются (CASCADE)
    category_id UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    
    -- Количество товара на складе (должно быть >= 0)
    stock_quantity INTEGER NOT NULL DEFAULT 0 CHECK (stock_quantity >= 0),
    
    -- Статус продукта (active, archived, draft и т.д.)
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    
    -- Дата и время создания записи
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Дата и время последнего обновления записи
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Дата мягкого удаления (NULL = не удалено)
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Индекс для фильтрации по мягкому удалению
CREATE INDEX idx_products_deleted_at ON products(deleted_at);

-- Индекс для быстрого соединения с таблицей categories
CREATE INDEX idx_products_category_id ON products(category_id);

-- Индекс для фильтрации по статусу
CREATE INDEX idx_products_status ON products(status);

-- Индекс для поиска по названию продукта
CREATE INDEX idx_products_name ON products(name);