package models

import (
	"time"

	"github.com/lib/pq"
)

type User struct {
	ID               uint            `gorm:"primaryKey" json:"id"`
	CreatedAt        time.Time       `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt        time.Time       `gorm:"autoUpdateTime" json:"updatedAt"`
	Name             string          `gorm:"default:New User" json:"name"`
	Email            string          `gorm:"unique" json:"email"`
	Password         string          `json:"password"`
	IsVerified       bool            `gorm:"default:false" json:"is_verified"`
	Code             string          `json:"code"`
	CodeCreatedAt    time.Time       `gorm:"autoCreateTime" json:"codeCreatedAt"`
	PhoneNumber      string          `gorm:"unique;type:varchar(11);not null" json:"phoneNumber"`
	Avatar           string          `gorm:"default:'https://res.cloudinary.com/dqipg0or3/image/upload/v1740564293/avatars/oil5t4os8o5x6dmmwusw.png'" json:"avatar"`
	Role             int             `gorm:"default:0" json:"role"`
	Status           int             `gorm:"default:0" json:"status"`
	Gender           int             `json:"gender"`
	DateOfBirth      string          `gorm:"default:'01/01/2000'" json:"dateOfBirth"`
	Banks            []Bank          `json:"banks" gorm:"foreignKey:UserId"`
	Children         []User          `gorm:"foreignKey:AdminId" json:"children,omitempty"`
	AdminId          *uint           `json:"adminId,omitempty"`
	Amount           int64           `gorm:"default:0" json:"amount"`
	AccommodationIDs pq.Int64Array   `json:"accommodation_ids" gorm:"type:integer[]"`
	CheckIns         []CheckInRecord `json:"checkins" gorm:"foreignKey:UserID"`
	DateCheck        time.Time       `json:"dateCheck"`
}
