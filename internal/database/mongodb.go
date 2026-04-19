package database

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Connect устанавливает соединение с MongoDB и проверяет его.
func Connect(ctx context.Context, uri string) (*mongo.Client, error) {
	opts := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, err
	}
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx, nil); err != nil {
		return nil, err
	}
	return client, nil
}

// EnsureIndexes создаёт все необходимые индексы (заменяет SQL-миграции).
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	type indexSpec struct {
		collection string
		model      mongo.IndexModel
	}

	specs := []indexSpec{
		// categories
		{
			collection: "categories",
			model: mongo.IndexModel{
				Keys:    bson.D{{Key: "deleted_at", Value: 1}},
				Options: options.Index().SetSparse(true),
			},
		},
		// products
		{
			collection: "products",
			model: mongo.IndexModel{
				Keys: bson.D{{Key: "category_id", Value: 1}, {Key: "deleted_at", Value: 1}},
			},
		},
		// users — уникальные sparse-индексы
		{
			collection: "users",
			model: mongo.IndexModel{
				Keys:    bson.D{{Key: "email", Value: 1}},
				Options: options.Index().SetUnique(true),
			},
		},
		{
			collection: "users",
			model: mongo.IndexModel{
				Keys:    bson.D{{Key: "phone", Value: 1}},
				Options: options.Index().SetUnique(true).SetSparse(true),
			},
		},
		{
			collection: "users",
			model: mongo.IndexModel{
				Keys:    bson.D{{Key: "yandex_id", Value: 1}},
				Options: options.Index().SetUnique(true).SetSparse(true),
			},
		},
		{
			collection: "users",
			model: mongo.IndexModel{
				Keys:    bson.D{{Key: "vk_id", Value: 1}},
				Options: options.Index().SetUnique(true).SetSparse(true),
			},
		},
		// refresh_tokens
		{
			collection: "refresh_tokens",
			model: mongo.IndexModel{
				Keys:    bson.D{{Key: "token_hash", Value: 1}},
				Options: options.Index().SetUnique(true),
			},
		},
		{
			collection: "refresh_tokens",
			model: mongo.IndexModel{
				Keys:    bson.D{{Key: "access_token_hash", Value: 1}},
				Options: options.Index().SetUnique(true).SetSparse(true),
			},
		},
		{
			collection: "refresh_tokens",
			model: mongo.IndexModel{
				Keys: bson.D{{Key: "user_id", Value: 1}},
			},
		},
		// password_reset_tokens
		{
			collection: "password_reset_tokens",
			model: mongo.IndexModel{
				Keys:    bson.D{{Key: "token", Value: 1}},
				Options: options.Index().SetUnique(true),
			},
		},
		{
			collection: "password_reset_tokens",
			model: mongo.IndexModel{
				Keys: bson.D{{Key: "user_id", Value: 1}},
			},
		},
	}

	for _, s := range specs {
		col := db.Collection(s.collection)
		if _, err := col.Indexes().CreateOne(ctx, s.model); err != nil {
			return err
		}
	}
	return nil
}
