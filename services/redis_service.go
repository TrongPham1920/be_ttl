package services

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// Hàm lấy data từ Redis
func GetFromRedis(ctx context.Context, rdb *redis.Client, key string, target interface{}) error {
	cachedData, err := rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return err
	}

	// Parse JSON thành object
	if err := json.Unmarshal([]byte(cachedData), target); err != nil {
		return err
	}
	return nil
}

// Hàm lưu dữ liệu vào Redis
func SetToRedis(ctx context.Context, rdb *redis.Client, key string, value interface{}, ttl time.Duration) error {
	dataJSON, err := json.Marshal(value)
	if err != nil {
		return err
	}

	if err := rdb.Set(ctx, key, dataJSON, ttl).Err(); err != nil {
		return err
	}
	return nil
}

// Hàm xóa cache Redis
func DeleteFromRedis(ctx context.Context, rdb *redis.Client, key string) error {
	if err := rdb.Del(ctx, key).Err(); err != nil {
		return err
	}
	return nil
}
