package config

import (
	"log"
	"os"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/joho/godotenv"
)

var Cloudinary *cloudinary.Cloudinary

func ConnectCloudinary() {
	var err error
	Cloudinary, err = cloudinary.NewFromParams("dqipg0or3", "921786437263773", "cK1ylPWzyoC4bTWWtahq0QDVZUw")
	if err != nil {
		log.Fatalf("Lỗi khi khởi tạo Cloudinary: %v", err)
	}
}

func LoadEnv() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}
}

func GetEnv(key string) string {
	return os.Getenv(key)
}
