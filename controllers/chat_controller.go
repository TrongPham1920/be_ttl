package controllers

import (
	"log"
	"net/http"
	"new/config"
	"new/dto"
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

// func handleUserMessage(client *Client, message []byte, c *gin.Context) {
// 	userInput := string(message)

// 	if userInput == "reset" {
// 		if err := services.ClearLastFilters(config.Ctx, config.RedisClient, client.UserID); err != nil {
// 			log.Println("ClearLastFilters:", err)
// 		}
// 		client.Send <- []byte("Đã reset bộ lọc tìm kiếm.")
// 		return
// 	}

// 	// Gọi GPT để phân tích ý định người dùng
// 	filters, response, err := services.ExtractSearchFiltersFromGPTWS(userInput)
// 	if err != nil {
// 		client.Send <- []byte("Lỗi khi phân tích yêu cầu.")
// 		return
// 	}

// 	// Nếu response có nội dung (ví dụ: gợi ý, tư vấn), gửi trước
// 	if response != "" {
// 		client.Send <- []byte(response)
// 	}

// 	// Nếu có filters => nghĩa là người dùng đã cần tìm kiếm khách sạn
// 	if filters != nil && client.UserID > 0 {

// 		// Lọc các chỗ đã đặt nếu có from/to date
// 		excludeIDs := []uint{}

// 		prevFilters, _ := services.GetLastFilters(config.Ctx, config.RedisClient, client.UserID)
// 		if prevFilters != nil {
// 			filters = services.MergeFilters(prevFilters, filters)
// 		}

// 		if filters.FromDate != nil && filters.ToDate != nil {
// 			statuses, err := GetAllAccommodationStatuses(c, *filters.FromDate, *filters.ToDate)
// 			if err == nil {
// 				for _, status := range statuses {
// 					excludeIDs = append(excludeIDs, status.AccommodationID)
// 				}
// 			}
// 		}
// 		_ = services.SaveLastFilters(config.Ctx, config.RedisClient, client.UserID, filters)
// 		log.Printf("Filters hiện tại sau khi gộp: %+v", filters)
// 		// Build truy vấn & tìm kiếm Elastic
// 		query := services.BuildESQueryFromFilters(filters, excludeIDs)
// 		results, _, err := services.SearchElastic(services.Es, query, "accommodations")
// 		if err != nil {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tìm kiếm: " + err.Error()})
// 			return
// 		}

// 		// Giới hạn kết quả tối đa 3 khách sạn
// 		if len(results) > 3 {
// 			results = results[:3]
// 		}

// 		// // Gửi tin nhắn giới thiệu
// 		// client.Send <- []byte("🏨 Đây là danh sách các khách sạn phù hợp với yêu cầu của bạn:")

// 		// Trả kết quả dưới dạng JSON (để frontend render card)
// 		hotelJSON, err := json.Marshal(results)
// 		if err == nil {
// 			client.Send <- hotelJSON
// 		} else {
// 			client.Send <- []byte("⚠️ Có lỗi khi gửi kết quả khách sạn.")
// 		}

// 		// // Lưu lịch sử nếu có userId
// 		// if client.UserID != -1 {
// 		// 	SaveChatHistory(client.UserID, userInput, "Kết quả tìm khách sạn")
// 		// }
// 	}
// }

func handleUserMessage(client *Client, message []byte, c *gin.Context, sessionId string) {
	userInput := string(message)

	// Tạo key Redis dùng cho session hoặc user
	redisKey := services.GetCacheKey(client.UserID, sessionId)

	responseMessages := services.HandleUserMessageWS(config.Ctx, config.RedisClient, services.Es, redisKey, client.UserID, userInput, c)
	for _, msg := range responseMessages {
		client.Send <- msg
	}
}
