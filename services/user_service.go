package services

import (
	"context"
	"fmt"
	"time"
	_ "time/tzdata"

	"new/models"
	"new/services/logger"
	"new/services/notification"

	"github.com/olahol/melody"
	"gorm.io/gorm"
)

const (
	// RevenueShareRate tỷ lệ chia sẻ doanh thu cho user
	RevenueShareRate = 0.7
	// Timezone mặc định
	DefaultTimezone = "Asia/Ho_Chi_Minh"
	// MinRevenue giá trị doanh thu tối thiểu
	MinRevenue = 0
)

// Error codes
const (
	ErrCodeInvalidTimezone = "INVALID_TIMEZONE"
	ErrCodeNoRevenue       = "NO_REVENUE"
	ErrCodeUpdateFailed    = "UPDATE_FAILED"
	ErrCodeInvalidRevenue  = "INVALID_REVENUE"
	ErrCodeInvalidUserID   = "INVALID_USER_ID"
)

// ServiceError định nghĩa lỗi của service
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

// UserServiceInterface định nghĩa các phương thức của UserService
type UserServiceInterface interface {
	GetTodayUserRevenue(ctx context.Context) ([]models.UserRevenue, error)
	UpdateUserAmounts(ctx context.Context, notificationService notification.Service) error
}

// UserService xử lý các logic liên quan đến user
type UserService struct {
	db     *gorm.DB
	logger logger.Logger
}

// UserServiceOptions định nghĩa các options cho UserService
type UserServiceOptions struct {
	DB     *gorm.DB
	Logger logger.Logger
}

// NewUserService tạo một instance mới của UserService
func NewUserService(opts UserServiceOptions) *UserService {
	return &UserService{
		db:     opts.DB,
		logger: opts.Logger,
	}
}

// validateRevenue kiểm tra tính hợp lệ của doanh thu
func validateRevenue(revenue float64) error {
	if revenue < MinRevenue {
		return &ServiceError{
			Code:    ErrCodeInvalidRevenue,
			Message: fmt.Sprintf("doanh thu phải lớn hơn hoặc bằng %f", MinRevenue),
		}
	}
	return nil
}

// validateUserID kiểm tra tính hợp lệ của user ID
func validateUserID(userID uint) error {
	if userID == 0 {
		return &ServiceError{
			Code:    ErrCodeInvalidUserID,
			Message: "user ID không hợp lệ",
		}
	}
	return nil
}

// GetTodayUserRevenue lấy danh sách doanh thu trong ngày hôm nay
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

// updateUserAmount cập nhật số tiền cho một user
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

// sendNotification gửi thông báo qua notification service
func (s *UserService) sendNotification(notificationService notification.Service, userID uint, revenue float64) error {
	message := notification.NewMessageBuilder(userID, revenue).Build()
	return notificationService.SendMessage(message)
}

// UpdateUserAmounts cập nhật amount của user dựa trên revenue hôm nay
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

	// Bắt đầu transaction
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

// UserServiceAdapter chuyển đổi UserService thành UserAmountUpdater
type UserServiceAdapter struct {
	service *UserService
}

// NewUserServiceAdapter tạo một instance mới của UserServiceAdapter
func NewUserServiceAdapter(service *UserService) *UserServiceAdapter {
	return &UserServiceAdapter{
		service: service,
	}
}

// UpdateUserAmounts implement UserAmountUpdater interface
func (a *UserServiceAdapter) UpdateUserAmounts(m *melody.Melody) error {
	notificationService := notification.NewMelodyService(m)
	return a.service.UpdateUserAmounts(context.Background(), notificationService)
}
