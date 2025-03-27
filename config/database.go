package config

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func getDBConfigByEnv(env string) string {
	var user, password, host, port, name string

	switch env {
	case "dev":
		user = os.Getenv("DEV_DB_USER")
		password = os.Getenv("DEV_DB_PASSWORD")
		host = os.Getenv("DEV_DB_HOST")
		port = os.Getenv("DEV_DB_PORT")
		name = os.Getenv("DEV_DB_NAME")
	case "qc":
		user = os.Getenv("QC_DB_USER")
		password = os.Getenv("QC_DB_PASSWORD")
		host = os.Getenv("QC_DB_HOST")
		port = os.Getenv("QC_DB_PORT")
		name = os.Getenv("QC_DB_NAME")
	case "prod":
		user = os.Getenv("PROD_DB_USER")
		password = os.Getenv("PROD_DB_PASSWORD")
		host = os.Getenv("PROD_DB_HOST")
		port = os.Getenv("PROD_DB_PORT")
		name = os.Getenv("PROD_DB_NAME")
	default:
		log.Fatalf("Unknown environment: %s", env)
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=require TimeZone=Asia/Ho_Chi_Minh",
		host, user, password, name, port)
	println(dsn)
	return dsn
}

func ConnectDB() {
	var err error
	env := os.Getenv("ENV")
	dsn := getDBConfigByEnv(env)

	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Fail to connect to db : %v", err)
	}

	fmt.Println("Successfully connected to db")
}
