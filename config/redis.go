package config

import (
	"context"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

var Ctx = context.Background()

// Hàm nạp biến môi trường từ tệp `.env`
func loadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Không thể nạp file .env, sử dụng biến môi trường hệ thống nếu có")
	}
}

// Hàm kết nối đến Redis
func ConnectRedis() (*redis.Client, error) {
	// Nạp biến môi trường từ tệp `.env`
	loadEnv()

	// Khởi tạo client Redis với các tùy chọn
	RDB := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDR"),
		Username: os.Getenv("REDIS_USER"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})

	// Kiểm tra kết nối
	res, err := RDB.Ping(Ctx).Result()
	if err != nil {
		return nil, err
	}

	log.Println("Kết nối Redis thành công:", res)
	return RDB, nil
}
