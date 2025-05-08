package models

import "time"

type ChatHistory struct {
	ID          int       `json:"id" gorm:"primaryKey"`
	UserID      int       `json:"user_id"`
	Sender      string    `json:"sender"` // "user" or "bot"
	MessageType string    `json:"message_type"`
	Content     string    `json:"content"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
}
																																																																																																																																																																																							