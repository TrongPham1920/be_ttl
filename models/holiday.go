package models

import (
	"time"
)

type Holiday struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name"`
	FromDate  string    `json:"fromDate"`                        // Ngày bắt đầu kỳ nghỉ
	ToDate    string    `json:"toDate"`                          // Ngày kết thúc kỳ nghỉ
	Price     int       `json:"price"`                           // Giá cho kỳ nghỉ
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"` // Thời gian tạo
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"` // Thời gian cập nhật
}
