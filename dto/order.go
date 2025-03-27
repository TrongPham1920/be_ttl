package dto

import (
	"new/types"
	"time"
)

// OrderResponse là DTO cho response của order
type OrderResponse struct {
	ID              uint                      `json:"id"`
	OrderCode       string                    `json:"orderCode"`
	UserID          uint                      `json:"userId"`
	AccommodationID uint                      `json:"accommodationId"`
	TotalPrice      float64                   `json:"totalPrice"`
	PaidAmount      float64                   `json:"paidAmount"`
	Status          int                       `json:"status"`
	CreatedAt       string                    `json:"createdAt"`
	UpdatedAt       string                    `json:"updatedAt"`
	User            types.InvoiceUserResponse `json:"user"`
	Accommodation   AccommodationResponse     `json:"accommodation"`
}

// ActorResponse là DTO cho thông tin user/actor
type ActorResponse struct {
	Name        string `json:"name"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phoneNumber"`
}

// UpdateOrderStatusRequest là DTO cho request cập nhật trạng thái order
type UpdateOrderStatusRequest struct {
	Status int `json:"status" binding:"required"`
}

// StatusUpdateRequest là DTO cho request cập nhật trạng thái order với paid amount
type StatusUpdateRequest struct {
	ID         uint    `json:"id"`
	Status     int     `json:"status"`
	PaidAmount float64 `json:"paidAmount"`
}

type OrderUserResponse struct {
	ID               uint                       `json:"id"`
	User             ActorResponse              `json:"user"`
	Accommodation    OrderAccommodationResponse `json:"accommodation"`
	Room             []OrderRoomResponse        `json:"room"`
	CheckInDate      string                     `json:"checkInDate"`
	CheckOutDate     string                     `json:"checkOutDate"`
	Status           int                        `json:"status"`
	CreatedAt        time.Time                  `json:"createdAt"`
	UpdatedAt        time.Time                  `json:"updatedAt"`
	Price            int                        `json:"price"`            // Giá cơ bản cho mỗi phòng
	HolidayPrice     float64                    `json:"holidayPrice"`     // Giá lễ
	CheckInRushPrice float64                    `json:"checkInRushPrice"` // Giá check-in gấp
	SoldOutPrice     float64                    `json:"soldOutPrice"`     // Giá sold out
	DiscountPrice    float64                    `json:"discountPrice"`    // Giá discount
	TotalPrice       float64                    `json:"totalPrice"`
	InvoiceCode      string                     `json:"invoiceCode"`
}

type OrderAccommodationResponse struct {
	ID       uint   `json:"id"`
	Type     int    `json:"type"`
	Name     string `json:"name"`
	Address  string `json:"address"`
	Ward     string `json:"ward"`
	District string `json:"district"`
	Province string `json:"province"`
	Price    int    `json:"price"`
	Avatar   string `json:"avatar"`
}

type OrderRoomResponse struct {
	ID              uint   `json:"id"`
	AccommodationID uint   `json:"accommodationId"`
	RoomName        string `json:"roomName"`
	Price           int    `json:"price"`
}

type CreateOrderRequest struct {
	UserID          uint   `json:"userId"`
	AccommodationID uint   `json:"accommodationId"`
	RoomID          []uint `json:"roomId"`
	CheckInDate     string `json:"checkInDate"`
	CheckOutDate    string `json:"checkOutDate"`
	GuestName       string `json:"guestName,omitempty"`
	GuestEmail      string `json:"guestEmail,omitempty"`
	GuestPhone      string `json:"guestPhone,omitempty"`
}
