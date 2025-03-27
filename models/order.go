package models

import (
	"time"
)

// Order status constants
const (
	OrderStatusPending   = 0
	OrderStatusConfirmed = 1
	OrderStatusCompleted = 2
	OrderStatusCancelled = 3
)

type Order struct {
	ID               uint          `json:"id" gorm:"primaryKey"`
	UserID           *uint         `json:"userId"`
	User             *User         `json:"user" gorm:"foreignKey:UserID"`
	AccommodationID  uint          `json:"accommodationId"`
	Accommodation    Accommodation `json:"accommodation" gorm:"foreignKey:AccommodationID;"`
	RoomID           []uint        `json:"roomId" gorm:"-"`
	Room             []Room        `json:"rooms" gorm:"many2many:order_rooms;"`
	CheckInDate      string        `json:"checkInDate"`
	CheckOutDate     string        `json:"checkOutDate"`
	Status           int           `json:"status"`
	CreatedAt        time.Time     `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt        time.Time     `gorm:"autoUpdateTime" json:"updatedAt"`
	GuestName        string        `json:"guestName,omitempty"`
	GuestEmail       string        `json:"guestEmail,omitempty"`
	GuestPhone       string        `json:"guestPhone,omitempty"`
	Price            int           `json:"price"`            // Giá cơ bản cho mỗi phòng
	HolidayPrice     float64       `json:"holidayPrice"`     // Giá lễ 10
	CheckInRushPrice float64       `json:"checkInRushPrice"` // Giá check-in gấp 5
	SoldOutPrice     float64       `json:"soldOutPrice"`     // Giá sold out 5
	DiscountPrice    float64       `json:"discountPrice"`    // Giá discount 20
	TotalPrice       float64       `json:"totalPrice"`       // Tổng giá
}

type OrderRequest struct {
	UserID          uint   `json:"userId"`
	AccommodationID uint   `json:"accommodationId"`
	RoomID          []uint `json:"roomId"`
	CheckInDate     string `json:"checkInDate"`
	CheckOutDate    string `json:"checkOutDate"`
	GuestName       string `json:"guestName,omitempty"`
	GuestEmail      string `json:"guestEmail,omitempty"`
	GuestPhone      string `json:"guestPhone,omitempty"`
}
