package jobs

import (
	"log"
	"time"

	"github.com/olahol/melody"
	"github.com/robfig/cron/v3"
)

// UserAmountUpdater định nghĩa interface cho việc cập nhật số tiền của user
type UserAmountUpdater interface {
	UpdateUserAmounts(m *melody.Melody) error
}

var userAmountUpdater UserAmountUpdater

// SetUserAmountUpdater thiết lập implementation cho UserAmountUpdater
func SetUserAmountUpdater(updater UserAmountUpdater) {
	userAmountUpdater = updater
}

// InitCronJobs khởi tạo các cron jobs
func InitCronJobs(c *cron.Cron, m *melody.Melody) error {
	// Cron job chạy lúc 0h mỗi ngày
	_, err := c.AddFunc("0 0 * * *", func() {
		now := time.Now()
		log.Printf("Đang chạy cập nhật số tiền cho người dùng lúc: %v", now)
		// if userAmountUpdater == nil {
		// 	log.Printf("Lỗi: UserAmountUpdater chưa được thiết lập")
		// 	return
		// }
		// if err := userAmountUpdater.UpdateUserAmounts(m); err != nil {
		// 	log.Printf("Lỗi khi cập nhật số tiền cho người dùng: %v", err)
		// }
	})
	if err != nil {
		return err
	}

	c.Start()
	log.Println("Cron jobs initialized successfully")
	return nil
}
