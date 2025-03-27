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

func LoadEnv() error {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	return err
}

func GetEnv(key string) string {
	return os.Getenv(key)
}
