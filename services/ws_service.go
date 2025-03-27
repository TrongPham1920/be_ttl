package services

import (
	"fmt"
	"log"
	"time"
	_ "time/tzdata"

	"new/config"
	"new/models"

	"github.com/olahol/melody"
	"gorm.io/gorm"
)

// GetTodayUserRevenue lấy danh sách doanh thu trong ngày hôm nay
func GetTodayUserRevenue() ([]models.UserRevenue, error) {
	var revenues []models.UserRevenue

	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		return nil, fmt.Errorf("❌ Lỗi khi tải múi giờ: %w", err)
	}

	today := time.Now().In(loc).AddDate(0, 0, -1).Format("2006-01-02")

	err = config.DB.Where(`date::date = ?`, today).Find(&revenues).Error
	if err != nil {
		return nil, fmt.Errorf("❌ Lỗi khi truy vấn doanh thu ngày hiện tại: %w", err)
	}

	return revenues, nil
}

// UpdateUserAmounts cập nhật amount của user dựa trên revenue hôm nay
func UpdateUserAmounts(m *melody.Melody) error {
	db := config.DB

	revenues, err := GetTodayUserRevenue()
	if err != nil {
		log.Println("❌ Lỗi lấy doanh thu:", err)
		return err
	}

	if len(revenues) == 0 {
		log.Println("ℹ️ Không có doanh thu nào để cập nhật hôm nay.")
		return nil
	}

	// Bắt đầu transaction
	tx := db.Begin()

	for _, rev := range revenues {
		adjustedRevenue := rev.Revenue * 0.7

		if err := tx.Model(&models.User{}).
			Where("id = ?", rev.UserID).
			Update("amount", gorm.Expr("amount + ?", adjustedRevenue)).Error; err != nil {
			tx.Rollback() // Nếu có lỗi, rollback transaction
			log.Printf("❌ Lỗi cập nhật amount cho user %d: %v\n", rev.UserID, err)
			return err
		}
		log.Printf("✅ Cập nhật thành công user_id %d: +%.2f\n", rev.UserID, rev.Revenue)

		//thông báo
		message := fmt.Sprintf("🔔 User %d đã được cộng %.2f vào tài khoản.", rev.UserID, rev.Revenue)
		m.Broadcast([]byte(message))
	}

	tx.Commit()

	log.Println("✅ Hoàn tất cập nhật amount cho tất cả users.")
	return nil
}
