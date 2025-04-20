package controllers

import (
	"errors"
	"fmt"
	"strconv"

	"new/models"
	"new/services/logger"
	"new/services/notification"

	"github.com/gin-gonic/gin"
	"github.com/olahol/melody"
	"gorm.io/gorm"
)

const (
	ErrCodeInvalidInput  = "INVALID_INPUT"
	ErrCodeNotifyFailed  = "NOTIFY_FAILED"
	ErrCodeInvalidUserID = "INVALID_USER_ID"
	ErrCodeUserNotFound  = "USER_NOT_FOUND"
	ErrCodeUpdateFailed  = "UPDATE_FAILED"
)

type ServiceError struct {
	Code    string
	Message string
	Err     error
}

func (e *ServiceError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

type NotificationObserver interface {
	Notify(message string) error
}

type MelodyObserver struct {
	session *melody.Session
	userID  uint
}

func NewMelodyObserver(session *melody.Session, userID uint) *MelodyObserver {
	return &MelodyObserver{
		session: session,
		userID:  userID,
	}
}

func (o *MelodyObserver) Notify(message string) error {
	return o.session.Write([]byte(message))
}

type NotificationController struct {
	db        *gorm.DB
	logger    logger.Logger
	melody    *melody.Melody
	observers map[uint][]NotificationObserver
}

type NotificationControllerOptions struct {
	DB     *gorm.DB
	Logger logger.Logger
}

func NewNotificationController(opts NotificationControllerOptions, m *melody.Melody) *NotificationController {
	return &NotificationController{
		db:        opts.DB,
		logger:    opts.Logger,
		melody:    m,
		observers: make(map[uint][]NotificationObserver),
	}
}

func (c *NotificationController) sendNotification(notificationService notification.Service, userID uint, revenue float64) error {
	message := notification.NewMessageBuilder(userID, revenue).Build()
	return notificationService.SendMessage(message)
}

func (c *NotificationController) NotifyAll(ctx *gin.Context) {
	var req struct {
		Message string `json:"message" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, &ServiceError{
			Code:    ErrCodeInvalidInput,
			Message: "Tin nhắn là bắt buộc",
			Err:     err,
		})
		return
	}

	notificationService := notification.NewMelodyService(c.melody)
	err := notificationService.SendMessage(req.Message)
	if err != nil {
		ctx.JSON(500, &ServiceError{
			Code:    ErrCodeNotifyFailed,
			Message: "Lỗi gửi thông báo tổng",
			Err:     err,
		})
		return
	}

	ctx.JSON(200, gin.H{
		"code":    1,
		"message": "Đã gửi thông báo tổng thành công",
		"data":    req.Message,
	})
}

func (c *NotificationController) NotifyUser(ctx *gin.Context) {
	userIDStr := ctx.Param("userID")

	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		ctx.JSON(400, &ServiceError{
			Code:    ErrCodeInvalidUserID,
			Message: "ID người dùng không hợp lệ",
			Err:     err,
		})
		return
	}

	var req struct {
		Message string `json:"message" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, &ServiceError{
			Code:    ErrCodeInvalidInput,
			Message: "Tin nhắn là bắt buộc",
			Err:     err,
		})
		return
	}

	message := notification.NewMessageBuilder(uint(userID), 0).Build() + " " + req.Message

	observers := c.observers[uint(userID)]
	var user models.User

	if err := c.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(404, &ServiceError{
				Code:    ErrCodeUserNotFound,
				Message: "Không tìm thấy người dùng",
			})
			return
		}
		ctx.JSON(500, &ServiceError{
			Code:    ErrCodeUpdateFailed,
			Message: "Không thể lấy được người dùng",
			Err:     err,
		})
		return
	}

	if len(observers) > 0 {
		for _, observer := range observers {
			if err := observer.Notify(message); err != nil {

			}
		}
	}

	// err = sendNews(user.Email, "Thông báo từ hệ thống", message)
	// if err != nil {

	// }

	ctx.JSON(200, gin.H{
		"code":    1,
		"message": "Thông báo đã được gửi đến người dùng",
		"data":    message,
	})
}

func (c *NotificationController) RegisterObserver(session *melody.Session, userID uint) {
	observer := NewMelodyObserver(session, userID)
	c.observers[userID] = append(c.observers[userID], observer)
	c.logger.Info("Người quan sát đã đăng ký cho userID: %d", userID)
}

func (c *NotificationController) RemoveObserver(session *melody.Session, userID uint) {
	observers := c.observers[userID]
	for i, obs := range observers {
		if obs.(*MelodyObserver).session == session {
			c.observers[userID] = append(observers[:i], observers[i+1:]...)
			break
		}
	}
	c.logger.Info("Đã xóa người quan sát cho userID: %d", userID)
}
