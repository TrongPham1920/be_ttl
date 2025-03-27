package config

import (
	"fmt"
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/olahol/melody"
	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

var RedisClient *redis.Client // Thay đổi kiểu dữ liệu từ interface{} sang *redis.Client

// InitApp khởi tạo toàn bộ ứng dụng
func InitApp() (*gin.Engine, *melody.Melody, *cron.Cron, error) {
	// Khởi tạo router
	router := gin.Default()

	// Khởi tạo các thành phần
	if err := initComponents(); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to initialize components: %v", err)
	}

	// Khởi tạo WebSocket
	m := melody.New()

	// Khởi tạo cron
	c := cron.New()

	// Cấu hình CORS
	configureCORS(router)

	return router, m, c, nil
}

// initComponents khởi tạo các thành phần cơ bản
func initComponents() error {
	// Load biến môi trường
	if err := LoadEnv(); err != nil {
		return fmt.Errorf("failed to load .env file: %v", err)
	}

	// Kết nối database
	ConnectDB()

	// Kết nối Cloudinary
	ConnectCloudinary()

	// Kết nối Redis
	var err error
	RedisClient, err = ConnectRedis()
	if err != nil {
		return fmt.Errorf("failed to connect to Redis: %v", err)
	}

	log.Println("All components initialized successfully")
	return nil
}

// configureCORS cấu hình CORS cho router
func configureCORS(router *gin.Engine) {
	configCors := cors.DefaultConfig()
	configCors.AddAllowHeaders("Authorization")
	configCors.AllowCredentials = true
	configCors.AllowAllOrigins = false
	configCors.AllowOriginFunc = func(origin string) bool {
		return true
	}
	router.Use(cors.New(configCors))
}

// InitCronJobs khởi tạo các cron jobs
func InitCronJobs(c *cron.Cron, m *melody.Melody) error {
	// Cron job chạy lúc 0h mỗi ngày
	_, err := c.AddFunc("0 0 * * *", func() {
		now := time.Now()
		log.Printf("Running UpdateUserAmounts at: %v", now)
		// if err := services.UpdateUserAmounts(m); err != nil {
		// 	log.Printf("Error updating user amounts: %v", err)
		// }
	})
	if err != nil {
		return fmt.Errorf("failed to add cron job: %v", err)
	}

	c.Start()
	log.Println("Cron jobs initialized successfully")
	return nil
}

// InitWebSocket khởi tạo WebSocket
func InitWebSocket(router *gin.Engine, m *melody.Melody) {
	router.GET("/ws", func(c *gin.Context) {
		m.HandleRequest(c.Writer, c.Request)
	})
	log.Println("WebSocket initialized successfully")
}

// InitSwagger khởi tạo Swagger documentation
func InitSwagger(router *gin.Engine) {
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	log.Println("Swagger documentation initialized successfully")
}
