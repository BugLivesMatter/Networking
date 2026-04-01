package cache

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// Service инкапсулирует работу с Redis-кешем.
type Service interface {
	Get(ctx context.Context, key string, dest interface{}) (bool, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Del(ctx context.Context, key string) error
	DelByPattern(ctx context.Context, pattern string) error
	Exists(ctx context.Context, key string) (bool, error)
}

type redisService struct {
	client  *redis.Client
	enabled bool
}

// NewService создаёт сервис кеша с безопасным fallback при отключенном Redis.
func NewService(client *redis.Client, enabled bool) Service {
	return &redisService{
		client:  client,
		enabled: enabled && client != nil,
	}
}

func (s *redisService) Get(ctx context.Context, key string, dest interface{}) (bool, error) {
	if !s.enabled {
		return false, nil
	}

	raw, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		log.Printf("кеш: ошибка чтения ключа %q: %v", key, err)
		return false, err
	}

	if err := json.Unmarshal([]byte(raw), dest); err != nil {
		log.Printf("кеш: ошибка десериализации ключа %q: %v", key, err)
		return false, err
	}

	return true, nil
}

func (s *redisService) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if !s.enabled {
		return nil
	}

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	if err := s.client.Set(ctx, key, data, ttl).Err(); err != nil {
		log.Printf("кеш: ошибка записи ключа %q: %v", key, err)
		return err
	}

	return nil
}

func (s *redisService) Del(ctx context.Context, key string) error {
	if !s.enabled {
		return nil
	}
	if err := s.client.Del(ctx, key).Err(); err != nil {
		log.Printf("кеш: ошибка удаления ключа %q: %v", key, err)
		return err
	}
	return nil
}

func (s *redisService) DelByPattern(ctx context.Context, pattern string) error {
	if !s.enabled {
		return nil
	}

	var cursor uint64
	for {
		keys, nextCursor, err := s.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			log.Printf("кеш: ошибка сканирования паттерна %q: %v", pattern, err)
			return err
		}
		if len(keys) > 0 {
			if err := s.client.Unlink(ctx, keys...).Err(); err != nil {
				log.Printf("кеш: ошибка удаления ключей по паттерну %q: %v", pattern, err)
				return err
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return nil
}

func (s *redisService) Exists(ctx context.Context, key string) (bool, error) {
	if !s.enabled {
		return false, nil
	}
	count, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		log.Printf("кеш: ошибка проверки существования ключа %q: %v", key, err)
		return false, err
	}
	return count > 0, nil
}
