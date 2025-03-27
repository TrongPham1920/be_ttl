package models

import "time"

type CheckInRecord struct {
	ID     uint      `gorm:"primaryKey"`
	UserID uint      `gorm:"index;not null"`
	Date   time.Time `gorm:"not null"`
}
