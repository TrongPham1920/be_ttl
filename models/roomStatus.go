package models

import "time"

type RoomStatus struct {
	ID        uint      `gorm:"primaryKey"`
	RoomID    uint      `gorm:"index"`
	FromDate  time.Time `gorm:"index"`
	ToDate    time.Time `gorm:"index"`
	Status    int
	CreatedAt time.Time
	UpdatedAt time.Time
}
