package models

import "time"

type AccommodationStatus struct {
	ID              uint      `gorm:"primaryKey"`
	AccommodationID uint      `gorm:"index"` // Liên kết với phòng
	FromDate        time.Time `gorm:"index"` // Ngày bắt đầu trạng thái
	ToDate          time.Time `gorm:"index"` // Ngày kết thúc trạng thái
	Status          int       // 0: có sẵn, 1: đã đặt, 2: đang bảo trì, v.v.
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
