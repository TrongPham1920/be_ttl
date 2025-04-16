package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"
	_ "time/tzdata"

	"new/config"
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
	ErrCodeInvalidInput    = "INVALID_INPUT"
	ErrCodeUserNotFound    = "USER_NOT_FOUND"
	ErrCodeNotifyFailed    = "NOTIFY_FAILED"
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
		s.logger.Error("❌ Lỗi đầu vào: tin nhắn là bắt buộc: %v", err)
		c.JSON(400, &ServiceError{
			Code:    ErrCodeInvalidInput,
			Message: "Tin nhắn là bắt buộc",
			Err:     err,
		})
		return
	}

	notificationService := notification.NewMelodyService(s.melody)
	err := notificationService.SendMessage(req.Message)
	if err != nil {
		s.logger.Error("❌ Lỗi gửi thông báo tổng: %v", err)
		c.JSON(500, &ServiceError{
			Code:    ErrCodeNotifyFailed,
			Message: "Lỗi gửi thông báo tổng",
			Err:     err,
		})
		return
	}

	s.logger.Info("✅ Đã gửi thông báo tổng: %s", req.Message)
	c.JSON(200, gin.H{
		"code":    1,
		"message": "Đã gửi thông báo tổng thành công",
		"data":    req.Message,
	})
}

// NotifyUser với thông báo qua WebSocket và email đồng thời
func (s *UserService) NotifyUser(c *gin.Context) {
	userIDStr := c.Param("userID")
	s.logger.Info("Đã nhận userID từ yêu cầu: %s", userIDStr)

	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		s.logger.Error("❌ Không phân tích được userID: %s, lỗi: %v", userIDStr, err)
		c.JSON(400, &ServiceError{
			Code:    ErrCodeInvalidUserID,
			Message: "ID người dùng không hợp lệ",
			Err:     err,
		})
		return
	}
	s.logger.Info("Parsed userID: %d", userID)

	var req struct {
		Message string `json:"message" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		s.logger.Error("❌ Lỗi đầu vào cho userID %d: tin nhắn là bắt buộc: %v", userID, err)
		c.JSON(400, &ServiceError{
			Code:    ErrCodeInvalidInput,
			Message: "Tin nhắn là bắt buộc",
			Err:     err,
		})
		return
	}
	s.logger.Info("Đã nhận được tin nhắn cho userID %d: %s", userID, req.Message)

	message := notification.NewMessageBuilder(uint(userID), 0).Build() + " " + req.Message
	s.logger.Info("Tin nhắn được xây dựng cho userID %d: %s", userID, message)

	observers := s.observers[uint(userID)]
	var user models.User
	// Lấy thông tin user từ DB để lấy email
	if err := s.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.logger.Error("❌ Không tìm thấy người dùng cho userID: %d", userID)
			c.JSON(404, &ServiceError{
				Code:    ErrCodeUserNotFound,
				Message: "Không tìm thấy người dùng",
			})
			return
		}
		s.logger.Error("❌ Không thể tìm nạp người dùng cho userID %d: %v", userID, err)
		c.JSON(500, &ServiceError{
			Code:    ErrCodeUpdateFailed,
			Message: "Không thể lấy được người dùng",
			Err:     err,
		})
		return
	}

	// Gửi qua WebSocket nếu có observer
	if len(observers) > 0 {
		s.logger.Info("Tìm thấy %d người quan sát cho userID: %d", len(observers), userID)
		for _, observer := range observers {
			if err := observer.Notify(message); err != nil {
				s.logger.Error("❌ Không thông báo được qua WebSocket cho userID %d: %v", userID, err)
			}
		}
		s.logger.Info("✅ Đã gửi thành công thông báo WebSocket tới userID %d: %s", userID, req.Message)
	} else {
		s.logger.Info("Không tìm thấy người quan sát nào cho userID: %d", userID)
	}

	// Gửi qua email bất kể có observer hay không
	err = sendNews(user.Email, "Thông báo từ hệ thống", message)
	if err != nil {
		s.logger.Error("❌ Không gửi được thông báo qua email cho userID %d: %v", userID, err)
	} else {
		s.logger.Info("📧 Thông báo qua email đã được gửi đến %s cho userID: %d", user.Email, userID)
	}

	// Trả về response thành công
	c.JSON(200, gin.H{
		"code":    1,
		"message": "Thông báo đã được gửi đến người dùng",
		"data":    message,
	})
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

func UpdateDailyRevenue() error {
	query := `
		INSERT INTO user_revenues (user_id, date, revenue, order_count, created_at, updated_at)
		SELECT 
			invoices.admin_id AS user_id,
			DATE(orders.created_at) AS date,
			SUM(invoices.total_amount) AS revenue,
			COUNT(invoices.id) AS order_count,
			NOW() AS created_at,
			NOW() AS updated_at
		FROM invoices
		JOIN orders ON invoices.order_id = orders.id
		GROUP BY invoices.admin_id, DATE(orders.created_at)
		ON CONFLICT (user_id, date)
		DO UPDATE SET 
			revenue = EXCLUDED.revenue,
			order_count = EXCLUDED.order_count,
			updated_at = NOW();
	`

	if err := config.DB.Exec(query).Error; err != nil {
		log.Printf("Lỗi cập nhật doanh thu hàng ngày: %v", err)
		return fmt.Errorf("updateDailyRevenue error: %w", err)
	}

	log.Printf("Cập nhật doanh thu hàng ngày thành công lúc %v", time.Now())
	return nil
}

func UpdateUserTotalAmount() error {
	query := `
		UPDATE users
		SET amount = COALESCE(agg.total_revenue, 0)
		FROM (
			SELECT user_id, SUM(revenue) AS total_revenue
			FROM user_revenues
			GROUP BY user_id
		) AS agg
		WHERE users.id = agg.user_id;
	`

	if err := config.DB.Exec(query).Error; err != nil {
		log.Printf("Lỗi cập nhật tổng doanh thu cho user: %v", err)
		return fmt.Errorf("updateUserTotalAmount error: %w", err)
	}

	log.Println("Cập nhật tổng doanh thu thành công")
	return nil
}
