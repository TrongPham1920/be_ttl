package controllers

import (
	"new/config"
	"new/models"
	"new/response"
	"new/services"
	"time"

	// "new/services/notification"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/olahol/melody"
)

type NotificationController struct {
	userService *services.UserService
	melody      *melody.Melody
}

func NewNotificationController(userService *services.UserService, melody *melody.Melody) *NotificationController {
	return &NotificationController{
		userService: userService,
		melody:      melody,
	}
}

// func (ctrl *NotificationController) NotifyAll(c *gin.Context) {
// 	var req struct {
// 		Message string `json:"message" binding:"required"`
// 	}
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		response.BadRequest(c, "Tin nhắn là bắt buộc")
// 		return
// 	}

// 	notificationService := notification.NewMelodyService(ctrl.melody)
// 	if err := notificationService.SendMessage(req.Message); err != nil {
// 		response.ServerError(c)
// 		return
// 	}

// 	response.Success(c, req.Message)
// }

// func (ctrl *NotificationController) NotifyUser(c *gin.Context) {
// 	userIDStr := c.Param("userID")
// 	userID, err := strconv.ParseUint(userIDStr, 10, 32)
// 	if err != nil {
// 		response.BadRequest(c, "ID người dùng không hợp lệ")
// 		return
// 	}

// 	var req struct {
// 		Message     string `json:"message" binding:"required"`
// 		Description string `json:"description"`
// 	}
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		response.BadRequest(c, "Tin nhắn là bắt buộc")
// 		return
// 	}

// 	message := req.Message
// 	observers := ctrl.userService.GetObservers(uint(userID))
// 	for _, observer := range observers {
// 		_ = observer.Notify(message)
// 	}
// 	notifyService := notification.NewNotifyService()
// 	if err := notifyService.CreateNotification(uint(userID), req.Message, req.Description); err != nil {
// 		response.ServerError(c)
// 		return
// 	}

//		response.Success(c, message)
//	}
func (ctrl *NotificationController) GetAllNotifications(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if token == "" {
		response.Unauthorized(c)
		return
	}
	token = strings.TrimPrefix(token, "Bearer ")

	_, role, err := GetUserIDFromToken(token)
	if err != nil {
		response.Unauthorized(c)
		return
	}

	if role != 1 {
		response.Forbidden(c)
		return
	}

	var notifications []models.Notification
	if err := config.DB.Order("created_at DESC").Find(&notifications).Error; err != nil {
		response.ServerError(c)
		return
	}

	response.Success(c, notifications)
}

func (ctrl *NotificationController) GetNotifyByUser(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if token == "" {
		response.Unauthorized(c)
		return
	}
	token = strings.TrimPrefix(token, "Bearer ")

	userID, err := GetIDFromToken(token)
	if err != nil {
		response.Unauthorized(c)
		return
	}

	var notifies []models.Notification
	if err := config.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&notifies).Error; err != nil {
		response.ServerError(c)
		return
	}

	response.Success(c, notifies)
}
func (ctrl *NotificationController) GetSystemNotifications(c *gin.Context) {
	var notifies []models.Notification
	if err := config.DB.Order("created_at DESC").Find(&notifies).Error; err != nil {
		response.ServerError(c)
		return
	}

	if len(notifies) == 0 {
		response.Success(c, "No system notifications found.")
		return
	}

	type NotificationResponse struct {
		Message     string    `json:"message"`
		Description string    `json:"description"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
	}

	var results []NotificationResponse
	for _, notify := range notifies {
		results = append(results, NotificationResponse{
			Message:     notify.Message,
			Description: notify.Description,
			CreatedAt:   notify.CreatedAt,
			UpdatedAt:   notify.UpdatedAt,
		})
	}
	response.Success(c, results)
}
