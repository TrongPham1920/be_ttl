package services

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"
	_ "time/tzdata"

	"new/models"
	"new/services/logger"
	"new/services/notification"

	"github.com/gin-gonic/gin"
	"github.com/olahol/melody"
	"gorm.io/gorm"
)

const (
	RevenueShareRate = 0.7
	DefaultTimezone  = "Asia/Ho_Chi_Minh"
	MinRevenue       = 0
)

const (
	ErrCodeInvalidTimezone = "INVALID_TIMEZONE"
	ErrCodeNoRevenue       = "NO_REVENUE"
	ErrCodeUpdateFailed    = "UPDATE_FAILED"
	ErrCodeInvalidRevenue  = "INVALID_REVENUE"
	ErrCodeInvalidUserID   = "INVALID_USER_ID"
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

type UserServiceInterface interface {
	GetTodayUserRevenue(ctx context.Context) ([]models.UserRevenue, error)
	UpdateUserAmounts(ctx context.Context, notificationService notification.Service) error
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

type UserService struct {
	db        *gorm.DB
	logger    logger.Logger
	melody    *melody.Melody
	observers map[uint][]NotificationObserver
}

type UserServiceOptions struct {
	DB     *gorm.DB
	Logger logger.Logger
}

func NewUserService(opts UserServiceOptions, m *melody.Melody) *UserService {
	return &UserService{
		db:        opts.DB,
		logger:    opts.Logger,
		melody:    m,
		observers: make(map[uint][]NotificationObserver),
	}
}

func validateRevenue(revenue float64) error {
	if revenue < MinRevenue {
		return &ServiceError{
			Code:    ErrCodeInvalidRevenue,
			Message: fmt.Sprintf("doanh thu phải lớn hơn hoặc bằng %f", MinRevenue),
		}
	}
	return nil
}

func validateUserID(userID uint) error {
	if userID == 0 {
		return &ServiceError{
			Code:    ErrCodeInvalidUserID,
			Message: "user ID không hợp lệ",
		}
	}
	return nil
}

func (s *UserService) GetTodayUserRevenue(ctx context.Context) ([]models.UserRevenue, error) {
	var revenues []models.UserRevenue
	loc, err := time.LoadLocation(DefaultTimezone)
	if err != nil {
		return nil, &ServiceError{
			Code:    ErrCodeInvalidTimezone,
			Message: "timezone không hợp lệ",
			Err:     err,
		}
	}
	today := time.Now().In(loc).AddDate(0, 0, -1).Format("2006-01-02")
	err = s.db.WithContext(ctx).Where(`date::date = ?`, today).Find(&revenues).Error
	if err != nil {
		return nil, &ServiceError{
			Code:    ErrCodeUpdateFailed,
			Message: "lỗi khi truy vấn doanh thu ngày hiện tại",
			Err:     err,
		}
	}
	return revenues, nil
}

func (s *UserService) updateUserAmount(ctx context.Context, tx *gorm.DB, userID uint, revenue float64) error {
	if err := validateUserID(userID); err != nil {
		return err
	}
	if err := validateRevenue(revenue); err != nil {
		return err
	}
	adjustedRevenue := revenue * RevenueShareRate
	if err := tx.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", userID).
		Update("amount", gorm.Expr("amount + ?", adjustedRevenue)).Error; err != nil {
		return &ServiceError{
			Code:    ErrCodeUpdateFailed,
			Message: fmt.Sprintf("lỗi cập nhật amount cho user %d", userID),
			Err:     err,
		}
	}
	s.logger.Info("✅ Cập nhật thành công user_id %d: +%.2f", userID, revenue)
	return nil
}

func (s *UserService) sendNotification(notificationService notification.Service, userID uint, revenue float64) error {
	message := notification.NewMessageBuilder(userID, revenue).Build()
	return notificationService.SendMessage(message)
}

func (s *UserService) UpdateUserAmounts(ctx context.Context, notificationService notification.Service) error {
	revenues, err := s.GetTodayUserRevenue(ctx)
	if err != nil {
		s.logger.Error("❌ Lỗi lấy doanh thu: %v", err)
		return err
	}
	if len(revenues) == 0 {
		s.logger.Info("ℹ️ Không có doanh thu nào để cập nhật hôm nay.")
		return &ServiceError{
			Code:    ErrCodeNoRevenue,
			Message: "không có doanh thu để cập nhật",
		}
	}
	tx := s.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return &ServiceError{
			Code:    ErrCodeUpdateFailed,
			Message: "lỗi khi bắt đầu transaction",
			Err:     tx.Error,
		}
	}
	for _, rev := range revenues {
		if err := s.updateUserAmount(ctx, tx, rev.UserID, rev.Revenue); err != nil {
			tx.Rollback()
			return err
		}
		if err := s.sendNotification(notificationService, rev.UserID, rev.Revenue); err != nil {
			s.logger.Error("❌ Lỗi gửi thông báo: %v", err)
		}
	}
	if err := tx.Commit().Error; err != nil {
		return &ServiceError{
			Code:    ErrCodeUpdateFailed,
			Message: "lỗi khi commit transaction",
			Err:     err,
		}
	}
	s.logger.Info("✅ Hoàn tất cập nhật amount cho tất cả users.")
	return nil
}

// đăng ký observer cho user
func (s *UserService) RegisterObserver(session *melody.Session, userID uint) {
	observer := NewMelodyObserver(session, userID)
	s.observers[userID] = append(s.observers[userID], observer)
	s.logger.Info("Người quan sát đã đăng ký cho userID: %d", userID)
}

// xóa observer cho user
func (s *UserService) RemoveObserver(session *melody.Session, userID uint) {
	observers := s.observers[userID]
	for i, obs := range observers {
		if obs.(*MelodyObserver).session == session {
			s.observers[userID] = append(observers[:i], observers[i+1:]...)
			break
		}
	}
	s.logger.Info("Đã xóa người quan sát cho userID: %d", userID)
}

func (s *UserService) NotifyAll(c *gin.Context) {
	var req struct {
		Message string `json:"message" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "tin nhắn là bắt buộc"})
		return
	}
	notificationService := notification.NewMelodyService(s.melody)
	err := notificationService.SendMessage(req.Message)
	if err != nil {
		s.logger.Error("❌ Lỗi gửi thông báo tổng: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	s.logger.Info("✅ Đã gửi thông báo tổng: %s", req.Message)
	c.JSON(200, gin.H{"message": "Broadcast sent"})
}

// NotifyUser với thông báo qua WebSocket và email đồng thời
func (s *UserService) NotifyUser(c *gin.Context) {
	userIDStr := c.Param("userID")
	fmt.Println("Đã nhận userID từ yêu cầu:", userIDStr)

	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		fmt.Println("Không phân tích được userID:", userIDStr, "error:", err)
		c.JSON(400, gin.H{"error": "invalid userID"})
		return
	}
	fmt.Println("Parsed userID:", userID)

	var req struct {
		Message string `json:"message" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Println("Failed to bind JSON for userID", userID, "error:", err)
		c.JSON(400, gin.H{"error": "message is required"})
		return
	}
	fmt.Println("Đã nhận được tin nhắn cho userID", userID, ":", req.Message)

	message := notification.NewMessageBuilder(uint(userID), 0).Build() + " " + req.Message
	fmt.Println("Tin nhắn được xây dựng cho userID", userID, ":", message)

	observers := s.observers[uint(userID)]
	var user models.User
	// Lấy thông tin user từ DB để lấy email
	if err := s.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fmt.Println("Không tìm thấy người dùng cho userID:", userID)
			c.JSON(404, gin.H{"error": "không tìm thấy người dùng"})
			return
		}
		fmt.Println("Không thể tìm nạp người dùng cho userID", userID, ":", err)
		c.JSON(500, gin.H{"error": "không thể lấy được người dùng"})
		return
	}

	// Gửi qua WebSocket nếu có observer
	if len(observers) > 0 {
		fmt.Println("Found", len(observers), "người quan sát cho userID:", userID)
		for _, observer := range observers {
			if err := observer.Notify(message); err != nil {
				fmt.Println("❌ Không thông báo được userID", userID, ":", err)
			}
		}
		fmt.Println("✅ Đã gửi thành công thông báo WebSocket tới userID", userID, ":", req.Message)
	} else {
		fmt.Println("Không tìm thấy người quan sát nào cho userID:", userID)
	}

	// Gửi qua email bất kể có observer hay không
	err = sendNews(user.Email, "Thông báo từ hệ thống", message)
	if err != nil {
		fmt.Println("❌ Không gửi được thông báo qua email cho userID", userID, ":", err)
		// Không trả lỗi ngay, chỉ log vì WebSocket có thể đã thành công
	} else {
		fmt.Println("📧 Thông báo qua email đã được gửi đến", user.Email, "for userID:", userID)
	}

	// Trả về response thành công nếu ít nhất một trong hai phương thức (WebSocket hoặc email) hoạt động
	c.JSON(200, gin.H{"message": "Thông báo được gửi đến người dùng"})
}

type UserServiceAdapter struct {
	service *UserService
}

func NewUserServiceAdapter(service *UserService) *UserServiceAdapter {
	return &UserServiceAdapter{service: service}
}

func (a *UserServiceAdapter) UpdateUserAmounts(m *melody.Melody) error {
	notificationService := notification.NewMelodyService(m)
	return a.service.UpdateUserAmounts(context.Background(), notificationService)
}
