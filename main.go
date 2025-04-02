package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"new/config"

	"new/jobs"
	"new/routes"
	"new/services"
	"new/services/logger"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func recreateUserTable() {
	// if err := config.DB.AutoMigrate(&models.Room{}, &models.Benefit{}, &models.User{}, models.Rate{}, models.Order{}, models.Invoice{}, models.Bank{}, models.Accommodation{}, models.AccommodationStatus{}, models.BankFake{}, models.UserDiscount{}, models.Discount{}, models.Holiday{}, models.RoomStatus{}, models.WithdrawalHistory{}); err != nil {
	// 	panic("Failed to migrate tables: " + err.Error())
	// }

}

func main() {

	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: không load được file .env, sử dụng biến môi trường có sẵn: %v", err)
	}

	// Khởi tạo ứng dụng
	router, m, c, err := config.InitApp()
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}

	configCors := cors.DefaultConfig()
	configCors.AddAllowHeaders("Authorization")
	configCors.AllowCredentials = true
	configCors.AllowAllOrigins = true
	router.Use(cors.New(configCors))
	router.SetTrustedProxies(nil)

	// Khởi tạo các services
	userService := services.NewUserService(services.UserServiceOptions{
		DB:     config.DB,
		Logger: logger.NewDefaultLogger(logger.InfoLevel),
	})
	userServiceAdapter := services.NewUserServiceAdapter(userService)
	jobs.SetUserAmountUpdater(userServiceAdapter)

	recreateUserTable()

	// Khởi tạo cron jobs
	if err := jobs.InitCronJobs(c, m); err != nil {
		log.Fatalf("Failed to initialize cron jobs: %v", err)
	}

	// Khởi tạo WebSocket
	config.InitWebSocket(router, m)

	// Setup các routes của ứng dụng
	routes.SetupRoutes(router, config.DB, config.RedisClient, config.Cloudinary, m)

	// Endpoint ping
	router.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	// Goroutine tự động gọi endpoint /ping mỗi 5 phút
	go func() {
		pingURL := "https://backend.trothalo.click/ping"
		for {
			resp, err := http.Get(pingURL)
			if err != nil {
				log.Printf("Error pinging /ping endpoint: %v", err)
			} else {
				body, _ := ioutil.ReadAll(resp.Body)
				resp.Body.Close()
				log.Printf("Ping response: %s", string(body))
			}
			time.Sleep(5 * time.Minute)
		}
	}()

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}

	log.Println("Server starting on port " + port + "...")
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
