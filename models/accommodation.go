package models

import (
	"encoding/json"
	"fmt"
	"time"
)

type Accommodation struct {
	ID               uint                  `json:"id" gorm:"primaryKey"` // ID cho hotel
	Type             int                   `json:"type"`                 // Loại chỗ ở (type)
	UserID           uint                  `json:"userId"`               // ID của người dùng
	Name             string                `json:"name"`                 // Tên khách sạn (name)
	Address          string                `json:"address"`              // Địa chỉ khách sạn
	CreateAt         time.Time             `gorm:"autoCreateTime"`
	UpdateAt         time.Time             `gorm:"autoUpdateTime"`
	Avatar           string                `json:"avatar"`               // Avatar khách sạn
	Img              json.RawMessage       `json:"img" gorm:"type:json"` // Hình ảnh khách sạn
	ShortDescription string                `json:"shortDescription"`     // Mô tả ngắn (shortDescription)
	Description      string                `json:"description"`          // Mô tả chi tiết
	Status           int                   `json:"status"`
	User             User                  `json:"user" gorm:"foreignKey:UserID"`           // Người dùng sở hữu
	Rooms            []Room                `json:"rooms" gorm:"foreignKey:AccommodationID"` // Danh sách các phòng
	Rates            []Rate                `json:"rates"`                                   // Danh sách các đánh giá
	Num              int                   `json:"num"`
	Furniture        json.RawMessage       `json:"furniture" gorm:"type:json"`
	People           int                   `json:"people"`
	Price            int                   `json:"price"`
	Benefits         []Benefit             `json:"benefits" gorm:"many2many:accommodation_benefits;"` // Mối quan hệ nhiều-nhiều
	NumBed           int                   `json:"numBed"`
	NumTolet         int                   `json:"numTolet"`
	TimeCheckOut     string                `json:"timeCheckOut"`
	TimeCheckIn      string                `json:"timeCheckIn"`
	Province         string                `json:"province"`
	District         string                `json:"district"`
	Ward             string                `json:"ward"`
	Longitude        float64               `json:"longitude"`
	Latitude         float64               `json:"latitude"`
	Statuses         []AccommodationStatus `json:"statuses" gorm:"foreignKey:AccommodationID"` // Danh sách trạng thái
}

func (r *Accommodation) ValidateType() error {
	if r.Type < 0 || r.Type > 4 {
		return fmt.Errorf("invalid Type: %d, must be between 0 and 4", r.Type)
	}
	return nil
}

func (r *Accommodation) ValidateStatus() error {
	if r.Status < 0 || r.Status > 4 {
		return fmt.Errorf("invalid Status: %d, must be between 0 and 4", r.Status)
	}
	return nil
}
