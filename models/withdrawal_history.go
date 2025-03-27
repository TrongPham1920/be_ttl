package models

import "time"

type WithdrawalHistory struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    uint      `gorm:"not null"`
	Amount    int64     `gorm:"not null"`
	Status    string    `gorm:"type:varchar(20);not null;default:0"`
	Reason    string    `gorm:"type:varchar(255)" json:"reason"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`

	User User `gorm:"foreignKey:UserID" json:"user"`
}
