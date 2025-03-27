package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"new/config"
	_ "new/docs"
	"new/jobs"
	"new/routes"
	"new/services"
	"new/services/logger"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func recreateUserTable() {
	// if err := config.DB.AutoMigrate(&models.Room{}, &models.Benefit{}, &models.User{}, models.Rate{}, models.Order{}, models.Invoice{}, models.Bank{}, models.Accommodation{}, models.AccommodationStatus{}, models.BankFake{}, models.UserDiscount{}, models.Discount{}, models.Holiday{}, models.RoomStatus{}, models.WithdrawalHistory{}); err != nil {
	// 	panic("Failed to migrate tables: " + err.Error())
	// }

	//test	Db
	// testUserID := uint(2)

	// for i := 1; i <= 500; i++ {
	// 	// Tạo dữ liệu giả cho hình ảnh với danh sách URL mà bạn yêu cầu
	// 	imgData, err := json.Marshal([]string{
	// 		"https://res.cloudinary.com/dqipg0or3/image/upload/v1740413058/uploads/qie2oeiajk8j7wwg8seh.jpg",
	// 		"https://res.cloudinary.com/dqipg0or3/image/upload/v1740413059/uploads/domlvkwnaoklhjqtwqmu.jpg",
	// 		"https://res.cloudinary.com/dqipg0or3/image/upload/v1740413059/uploads/eskliphwt7yc9mhmczvm.jpg",
	// 		"https://res.cloudinary.com/dqipg0or3/image/upload/v1740413060/uploads/upck5rgvr7wowrx2bzaz.jpg",
	// 		"https://res.cloudinary.com/dqipg0or3/image/upload/v1740413060/uploads/htx5nzcm9i6i5y70ybgv.jpg",
	// 		"https://res.cloudinary.com/dqipg0or3/image/upload/v1740413061/uploads/xiqtah9exsn6jhybkwlo.jpg",
	// 		"https://res.cloudinary.com/dqipg0or3/image/upload/v1740413061/uploads/wvvnu5rpgndrl79n5exq.jpg",
	// 		"https://res.cloudinary.com/dqipg0or3/image/upload/v1740413063/uploads/jqufrmzvcp2adssedlz5.jpg",
	// 	})
	// 	if err != nil {
	// 		log.Fatalf("Lỗi khi mã hóa imgData: %v", err)
	// 	}

	// 	// Tạo dữ liệu giả cho nội thất
	// 	furnitureData, err := json.Marshal([]string{
	// 		"Chair",
	// 		"Table",
	// 	})
	// 	if err != nil {
	// 		log.Fatalf("Lỗi khi mã hóa furnitureData: %v", err)
	// 	}

	// 	accommodation := models.Accommodation{
	// 		Type:             2,
	// 		UserID:           testUserID,
	// 		Name:             fmt.Sprintf("Test Accommodation %d", i),
	// 		Address:          fmt.Sprintf("Address %d", i),
	// 		Avatar:           "https://res.cloudinary.com/dqipg0or3/image/upload/v1740413047/avatars/obtrpfkzvr5k83bur5w0.jpg",
	// 		Img:              imgData,
	// 		ShortDescription: "Đây là mô tả ngắn cho test data.",
	// 		Description:      "Đây là mô tả chi tiết cho test data.",
	// 		Status:           1,
	// 		Num:              10,
	// 		Furniture:        furnitureData,
	// 		People:           2,
	// 		Price:            100 + i,
	// 		NumBed:           2,
	// 		NumTolet:         1,
	// 		TimeCheckIn:      "14:00",
	// 		TimeCheckOut:     "12:00",
	// 		Province:         "Test Province",
	// 		District:         "Test District",
	// 		Ward:             "Test Ward",
	// 		Longitude:        106.0 + float64(i)/100,
	// 		Latitude:         10.0 + float64(i)/100,
	// 		CreateAt:         time.Now(),
	// 		UpdateAt:         time.Now(),
	// 		Benefits: []models.Benefit{
	// 			{Id: 1, Name: "Wifi miễn phí"},
	// 			{Id: 2, Name: "Hồ bơi"},
	// 		},
	// 	}

	// 	if err := config.DB.Create(&accommodation).Error; err != nil {
	// 		log.Fatalf("Lỗi khi tạo Accommodation %d: %v", i, err)
	// 	}
	// 	fmt.Printf("Đã tạo Accommodation ID: %d\n", accommodation.ID)
	// }

}

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	// Khởi tạo ứng dụng
	router, m, c, err := config.InitApp()
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}

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

	// Khởi tạo Swagger
	config.InitSwagger(router)

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
