package main

import (
	"log"
	"os"

	// "strconv"

	"new/config"
	"new/models"

	"new/routes"
	"new/services"
	"new/services/logger"

	"github.com/joho/godotenv"
	// "github.com/olahol/melody"
)

func recreateUserTable() {
	// if err := config.DB.AutoMigrate(&models.Room{}, &models.Benefit{}, &models.User{}, models.Rate{}, models.Order{}, models.Invoice{}, models.Bank{}, models.Accommodation{}, models.AccommodationStatus{}, models.BankFake{}, models.UserDiscount{}, models.Discount{}, models.Holiday{}, models.RoomStatus{}, models.WithdrawalHistory{}); err != nil {
	// 	panic("Failed to migrate tables: " + err.Error())
	// }

	if err := config.DB.AutoMigrate(&models.Notification{}); err != nil {
		panic("Failed to migrate tables: " + err.Error())
	}

}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Không load được file .env: %v", err)
	}

	router, m, err := config.InitApp()
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}

	userService := services.NewUserService(services.UserServiceOptions{
		DB:     config.DB,
		Logger: logger.NewDefaultLogger(logger.InfoLevel),
	}, m)

	recreateUserTable()

	// Xử lý kết nối WebSocket với Observer Pattern
	// m.HandleConnect(func(s *melody.Session) {
	// 	userIDStr := s.Request.URL.Query().Get("userID")
	// 	if userIDStr != "" {
	// 		userID, _ := strconv.ParseUint(userIDStr, 10, 32)
	// 		s.Set("userID", userIDStr)
	// 		notificationController.RegisterObserver(s, uint(userID))
	// 	}
	// })

	// m.HandleDisconnect(func(s *melody.Session) {
	// 	userIDStr, exists := s.Get("userID")
	// 	if exists {
	// 		userID, _ := strconv.ParseUint(userIDStr.(string), 10, 32)
	// 		notificationController.RemoveObserver(s, uint(userID))
	// 	}
	// })

	config.InitWebSocket(router, m)

	routes.SetupRoutes(router, config.DB, config.RedisClient, config.Cloudinary, m, userService)

	// router.GET("/ping", func(c *gin.Context) {
	// 	c.String(http.StatusOK, "pong")
	// })

	// go func() {
	// 	pingURL := "https://be.trothalo.click/ping"
	// 	for {
	// 		resp, err := http.Get(pingURL)
	// 		if err != nil {
	// 			log.Printf("Error pinging /ping endpoint: %v", err)
	// 		} else {
	// 			body, _ := io.ReadAll(resp.Body)
	// 			resp.Body.Close()
	// 			log.Printf("Ping response: %s", string(body))
	// 		}
	// 		time.Sleep(5 * time.Minute)
	// 	}
	// }()

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
