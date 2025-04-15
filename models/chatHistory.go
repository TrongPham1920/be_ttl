package models

import "time"

type ChatHistory struct {
	ID        int       `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id"`
	Message   string    `json:"message"`
	Sender    string    `json:"sender"` // "user" or "bot"
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}
