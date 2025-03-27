package models

import "time"

type Revenue struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	UserID      uint      `json:"userId"`
	OrderID     uint      `json:"orderId"`
	Amount      float64   `json:"amount"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updatedAt" gorm:"autoUpdateTime"`
	User        User      `json:"user" gorm:"foreignKey:UserID"`
	Order       Order     `json:"order" gorm:"foreignKey:OrderID"`
}
