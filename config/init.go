package config

import (
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/olahol/melody"
	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

var MelodyInstance *melody.Melody

func InitApp() (*gin.Engine, *melody.Melody, error) {
	router := gin.Default()

	configCors := cors.DefaultConfig()
	configCors.AddAllowHeaders("Authorization")
	configCors.AllowCredentials = true
	configCors.AllowAllOrigins = true
	// configCors.AllowOriginFunc = func(origin string) bool {
	// 	return true
	// }
	router.Use(cors.New(configCors))

	router.SetTrustedProxies(nil)

	ConnectDB()

	ConnectCloudinary()

	var err error
	RedisClient, err = ConnectRedis()
	if err != nil {
		log.Printf("Warning: Failed to connect to Redis: %v", err)
	}

	m := melody.New()
	MelodyInstance = m

	return router, m, nil
}

func InitWebSocket(router *gin.Engine, m *melody.Melody) {
	router.GET("/ws", func(c *gin.Context) {
		m.HandleRequest(c.Writer, c.Request)
	})
}
