package controllers

import (
	"encoding/json"
	"log"
	"net/http"
	"new/config"
	"new/dto"
	"new/models"
	"new/response"
	"new/services"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Client struct {
	UserID      int
	Conn        *websocket.Conn
	Send        chan []byte
	LastFilters *dto.SearchFilters
	SessionID   string
}

var clients = make(map[*Client]bool)

func HandleWebSocket(c *gin.Context) {
	userIdstr := c.Query("userId")
	userId, _ := strconv.Atoi(userIdstr)
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("WebSocket error:", err)
		return
	}

	sessionId, err := c.Cookie("session_id")
	if err != nil {
		sessionId = uuid.New().String()
		// Gửi lại cookie về phía client nếu cần
		c.SetCookie("session_id", sessionId, 3600*24, "/", "", false, true)
	}

	client := &Client{
		UserID:    userId,
		Conn:      conn,
		Send:      make(chan []byte),
		SessionID: sessionId,
	}

	clients[client] = true

	go readMessages(client, c)
	go writeMessages(client)
}

func readMessages(client *Client, c *gin.Context) {
	defer client.Conn.Close()

	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			log.Println("read error:", err)
			delete(clients, client)
			break
		}

		go handleUserMessage(client, message, c, client.SessionID)
		log.Println("session", client.SessionID)
	}
}

func writeMessages(client *Client) {
	for msg := range client.Send {
		err := client.Conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			log.Println("write error:", err)
			client.Conn.Close()
			delete(clients, client)
			break
		}
	}
}

func handleUserMessage(client *Client, message []byte, c *gin.Context, sessionId string) {
	var inputMsg dto.IncomingMessage
	err := json.Unmarshal(message, &inputMsg)
	if err != nil {
		log.Println("invalid user message format:", err)
		return
	}

	_ = services.SaveChatHistoryToDB(client.UserID, "user", "text", inputMsg.Text)
	// Tạo key Redis dùng cho session hoặc user
	redisKey := services.GetCacheKey(client.UserID, sessionId)

	responseMessages := services.HandleUserMessageWS(config.Ctx, config.RedisClient, services.Es, redisKey, client.UserID, inputMsg.Text, c)
	for _, msg := range responseMessages {
		client.Send <- msg

		var msgType string
		if json.Valid(msg) {
			msgType = "json"
		} else {
			msgType = "text"
		}

		_ = services.SaveChatHistoryToDB(client.UserID, "bot", msgType, string(msg))
	}
}

func GetChatHistory(c *gin.Context) {
	userIdstr := c.Param("id")
	userID, _ := strconv.Atoi(userIdstr)
	log.Println("Parsed userID:", userID)
	// Lấy query `page` và `limit`
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "20")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 20
	}

	offset := (page - 1) * limit

	var messages []models.ChatHistory

	err = config.DB.
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch messages"})
		return
	}
	// 3. Đảo ngược để hiển thị từ cũ -> mới
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	total := len(messages)

	response.SuccessWithPagination(c, messages, page, limit, total)
}
