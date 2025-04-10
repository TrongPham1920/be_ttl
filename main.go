package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"new/config"

	"new/jobs"
	"new/routes"
	"new/services"
	"new/services/logger"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/olahol/melody"
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

	router, m, c, err := config.InitApp()
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}

	userService := services.NewUserService(services.UserServiceOptions{
		DB:     config.DB,
		Logger: logger.NewDefaultLogger(logger.InfoLevel),
	}, m)
	userServiceAdapter := services.NewUserServiceAdapter(userService)
	jobs.SetUserAmountUpdater(userServiceAdapter)

	recreateUserTable()
	// Xử lý kết nối WebSocket với Observer Pattern
	m.HandleConnect(func(s *melody.Session) {
		userIDStr := s.Request.URL.Query().Get("userID")
		if userIDStr != "" {
			userID, _ := strconv.ParseUint(userIDStr, 10, 32)
			s.Set("userID", userIDStr)
			userService.RegisterObserver(s, uint(userID))
		}
	})

	m.HandleDisconnect(func(s *melody.Session) {
		userIDStr, exists := s.Get("userID")
		if exists {
			userID, _ := strconv.ParseUint(userIDStr.(string), 10, 32)
			userService.RemoveObserver(s, uint(userID))
		}
	})
	if err := config.InitCronJobs(c, m); err != nil {
		log.Fatalf("Failed to initialize cron jobs: %v", err)
	}

	config.InitWebSocket(router, m)

	routes.SetupRoutes(router, config.DB, config.RedisClient, config.Cloudinary, m, userService)

	router.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	go func() {
		pingURL := "https://backend.trothalo.click/ping"
		for {
			resp, err := http.Get(pingURL)
			if err != nil {
				log.Printf("Error pinging /ping endpoint: %v", err)
			} else {
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				log.Printf("Ping response: %s", string(body))
			}
			time.Sleep(5 * time.Minute)
		}
	}()
	//Elastic dùng để Index dữ liệu hoặc xóa index
	services.ConnectElastic()
	// services.IndexHotelsToES()
	// services.DeleteIndex("accommodations")
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
