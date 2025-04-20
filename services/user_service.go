package services

import (
	"context"
	"encoding/json"

	"fmt"
	"log"

	"time"
	_ "time/tzdata"

	"new/config"
	"new/models"
	"new/services/logger"
	"new/services/notification"

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
	s.logger.Info("Cập nhật thành công user_id %d: +%.2f", userID, revenue)
	return nil
}

func (s *UserService) UpdateUserAmounts(ctx context.Context, notificationService notification.Service) error {
	revenues, err := s.GetTodayUserRevenue(ctx)
	if err != nil {
		s.logger.Error("Lỗi lấy doanh thu: %v", err)
		return err
	}
	if len(revenues) == 0 {
		s.logger.Info("Không có doanh thu nào để cập nhật hôm nay.")
		return &ServiceError{
			Code:    ErrCodeNoRevenue,
			Message: "không có doanh thu để cập nhật",
		}
	}
	tx := s.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return &ServiceError{
			Code:    ErrCodeUpdateFailed,
			Message: "Lỗi khi bắt đầu transaction",
			Err:     tx.Error,
		}
	}
	for _, rev := range revenues {
		if err := s.updateUserAmount(ctx, tx, rev.UserID, rev.Revenue); err != nil {
			tx.Rollback()
			return err
		}

	}
	if err := tx.Commit().Error; err != nil {
		return &ServiceError{
			Code:    ErrCodeUpdateFailed,
			Message: "Lỗi khi commit transaction",
			Err:     err,
		}
	}
	s.logger.Info("Hoàn tất cập nhật amount cho tất cả users.")
	return nil
}

func (s *UserService) RegisterObserver(session *melody.Session, userID uint) {
	observer := NewMelodyObserver(session, userID)
	s.observers[userID] = append(s.observers[userID], observer)
	s.logger.Info("Người quan sát đã đăng ký cho userID: %d", userID)
}

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

func (s *UserService) GetObservers(userID uint) []NotificationObserver {
	return s.observers[userID]
}

type UserServiceAdapter struct {
	service *UserService
}

func NewUserServiceAdapter(service *UserService) *UserServiceAdapter {
	return &UserServiceAdapter{service: service}
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

func SeedTestAccommodations(count int, userID uint) {
	for i := 1; i <= count; i++ {
		// Tạo dữ liệu giả cho hình ảnh
		imgData, err := json.Marshal([]string{
			"https://res.cloudinary.com/dqipg0or3/image/upload/v1740413058/uploads/qie2oeiajk8j7wwg8seh.jpg",
			"https://res.cloudinary.com/dqipg0or3/image/upload/v1740413059/uploads/domlvkwnaoklhjqtwqmu.jpg",
			"https://res.cloudinary.com/dqipg0or3/image/upload/v1740413059/uploads/eskliphwt7yc9mhmczvm.jpg",
			"https://res.cloudinary.com/dqipg0or3/image/upload/v1740413060/uploads/upck5rgvr7wowrx2bzaz.jpg",
			"https://res.cloudinary.com/dqipg0or3/image/upload/v1740413060/uploads/htx5nzcm9i6i5y70ybgv.jpg",
			"https://res.cloudinary.com/dqipg0or3/image/upload/v1740413061/uploads/xiqtah9exsn6jhybkwlo.jpg",
			"https://res.cloudinary.com/dqipg0or3/image/upload/v1740413061/uploads/wvvnu5rpgndrl79n5exq.jpg",
			"https://res.cloudinary.com/dqipg0or3/image/upload/v1740413063/uploads/jqufrmzvcp2adssedlz5.jpg",
		})
		if err != nil {
			log.Fatalf("Lỗi khi mã hóa imgData: %v", err)
		}

		// Dữ liệu nội thất
		furnitureData, err := json.Marshal([]string{
			"Chair",
			"Table",
		})
		if err != nil {
			log.Fatalf("Lỗi khi mã hóa furnitureData: %v", err)
		}

		accommodation := models.Accommodation{
			Type:             2,
			UserID:           userID,
			Name:             fmt.Sprintf("Test Accommodation %d", i),
			Address:          fmt.Sprintf("Address %d", i),
			Avatar:           "https://res.cloudinary.com/dqipg0or3/image/upload/v1740413047/avatars/obtrpfkzvr5k83bur5w0.jpg",
			Img:              imgData,
			ShortDescription: "Đây là mô tả ngắn cho test data.",
			Description:      "Đây là mô tả chi tiết cho test data.",
			Status:           1,
			Num:              10,
			Furniture:        furnitureData,
			People:           2,
			Price:            100 + i,
			NumBed:           2,
			NumTolet:         1,
			TimeCheckIn:      "14:00",
			TimeCheckOut:     "12:00",
			Province:         "Test Province",
			District:         "Test District",
			Ward:             "Test Ward",
			Longitude:        106.0 + float64(i)/100,
			Latitude:         10.0 + float64(i)/100,
			CreateAt:         time.Now(),
			UpdateAt:         time.Now(),
			Benefits: []models.Benefit{
				{Id: 1, Name: "Wifi miễn phí"},
				{Id: 2, Name: "Hồ bơi"},
			},
		}

		if err := config.DB.Create(&accommodation).Error; err != nil {
			log.Fatalf("Lỗi khi tạo Accommodation %d: %v", i, err)
		}

		fmt.Printf("Đã tạo Accommodation ID: %d\n", accommodation.ID)
	}
}
