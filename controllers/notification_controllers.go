package controllers

import (
	"new/response"
	"strconv"

	"new/services"
	"new/services/notification"

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

func (ctrl *NotificationController) NotifyAll(c *gin.Context) {
	var req struct {
		Message string `json:"message" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Tin nhắn là bắt buộc")
		return
	}

	notificationService := notification.NewMelodyService(ctrl.melody)
	if err := notificationService.SendMessage(req.Message); err != nil {
		response.ServerError(c)
		return
	}

	response.Success(c, req.Message)
}

func (ctrl *NotificationController) NotifyUser(c *gin.Context) {
	userIDStr := c.Param("userID")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "ID người dùng không hợp lệ")
		return
	}

	var req struct {
		Message string `json:"message" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Tin nhắn là bắt buộc")
		return
	}

	message := notification.NewMessageBuilder(uint(userID), 0).Build() + " " + req.Message

	observers := ctrl.userService.GetObservers(uint(userID))
	if len(observers) > 0 {
		for _, observer := range observers {
			if err := observer.Notify(message); err != nil {
			}
		}
	}

	response.Success(c, message)
}
