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
)

var RedisClient *redis.Client

func InitApp() (*gin.Engine, *melody.Melody, *cron.Cron, error) {

	router := gin.Default()

	configCors := cors.DefaultConfig()
	configCors.AddAllowHeaders("Authorization")
	configCors.AllowCredentials = true
	configCors.AllowAllOrigins = false
	configCors.AllowOriginFunc = func(origin string) bool {
		return true
	}
	router.Use(cors.New(configCors))

	router.SetTrustedProxies(nil)

	if err := initComponents(); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to initialize components: %v", err)
	}

	m := melody.New()

	c := cron.New()

	return router, m, c, nil
}

func initComponents() error {

	if err := LoadEnv(); err != nil {
		return fmt.Errorf("failed to load .env file: %v", err)
	}

	ConnectDB()

	ConnectCloudinary()

	var err error
	RedisClient, err = ConnectRedis()
	if err != nil {
		return fmt.Errorf("failed to connect to Redis: %v", err)
	}

	log.Println("All components initialized successfully")
	return nil
}

func InitCronJobs(c *cron.Cron, m *melody.Melody) error {

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

func InitWebSocket(router *gin.Engine, m *melody.Melody) {
	router.GET("/ws", func(c *gin.Context) {
		m.HandleRequest(c.Writer, c.Request)
	})
	log.Println("WebSocket initialized successfully")
}
