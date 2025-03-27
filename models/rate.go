package models

import "time"

type Rate struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	UserID          uint      `json:"userId"`
	AccommodationID uint      `json:"accommodationId"`
	Comment         string    `json:"comment"` // Bình luận của người dùng
	Star            int       `json:"star"`    // Số sao (điểm đánh giá)
	CreateAt        time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdateAt        time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
	User            User      `json:"user" gorm:"foreignKey:UserID"`
}
