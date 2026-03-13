-- +goose Up
-- +goose StatementBegin

-- Создаём таблицу пользователей
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    phone VARCHAR(20) UNIQUE,
    password_hash TEXT NOT NULL,
    salt TEXT NOT NULL,
    yandex_id VARCHAR(255) UNIQUE,
    vk_id VARCHAR(255) UNIQUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Индексы для ускорения поиска
CREATE INDEX idx_users_email_active ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_deleted_at ON users(deleted_at);
CREATE INDEX idx_users_yandex_id ON users(yandex_id) WHERE yandex_id IS NOT NULL;
CREATE INDEX idx_users_vk_id ON users(vk_id) WHERE vk_id IS NOT NULL;

-- +goose StatementEnd