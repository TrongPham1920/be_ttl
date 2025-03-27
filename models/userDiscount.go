package models

import "time"

type UserDiscount struct {
	ID         uint      `gorm:"primaryKey"` // Khóa chính
	UserID     uint      `gorm:"not null"`   // ID người dùng
	DiscountID uint      `gorm:"not null"`   // ID mã giảm giá
	UsageCount int       `gorm:"default:0"`  // Số lần sử dụng mã giảm giá
	CreatedAt  time.Time `gorm:"createdAt"`  // Thời gian tạo bản ghi
	UpdatedAt  time.Time `gorm:"updatedAt"`  // Thời gian cập nhật bản ghi

	User     User     `gorm:"foreignKey:UserID;" json:"user"`
	Discount Discount `gorm:"foreignKey:DiscountID;" json:"discount"`
}
