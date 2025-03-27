package models

import "time"

type UserRevenue struct {
	ID         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     uint      `gorm:"not null;uniqueIndex:idx_user_date" json:"user_id"`
	User       User      `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"user"`
	Date       time.Time `gorm:"not null;uniqueIndex:idx_user_date" json:"date"`
	Revenue    float64   `gorm:"not null" json:"revenue"`
	OrderCount int       `gorm:"not null" json:"order_count"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
