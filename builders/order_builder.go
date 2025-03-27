package builders

import (
	"new/models"
)

// OrderBuilder giúp tạo order theo từng bước
type OrderBuilder struct {
	order *models.Order
}

// NewOrderBuilder tạo instance mới của OrderBuilder
func NewOrderBuilder() *OrderBuilder {
	return &OrderBuilder{
		order: &models.Order{},
	}
}

// WithUser thêm thông tin user
func (b *OrderBuilder) WithUser(userID uint) *OrderBuilder {
	b.order.UserID = &userID
	return b
}

// WithRoom thêm thông tin phòng
func (b *OrderBuilder) WithRoom(roomIDs []uint) *OrderBuilder {
	b.order.RoomID = roomIDs
	return b
}

// WithStatus thêm trạng thái
func (b *OrderBuilder) WithStatus(status int) *OrderBuilder {
	b.order.Status = status
	return b
}

// WithGuestInfo thêm thông tin khách
func (b *OrderBuilder) WithGuestInfo(guestName, guestPhone, guestEmail string) *OrderBuilder {
	b.order.GuestName = guestName
	b.order.GuestPhone = guestPhone
	b.order.GuestEmail = guestEmail
	return b
}

// WithCheckIn thêm thời gian check-in
func (b *OrderBuilder) WithCheckIn(checkIn string) *OrderBuilder {
	b.order.CheckInDate = checkIn
	return b
}

// WithCheckOut thêm thời gian check-out
func (b *OrderBuilder) WithCheckOut(checkOut string) *OrderBuilder {
	b.order.CheckOutDate = checkOut
	return b
}

// WithTotalPrice thêm tổng giá
func (b *OrderBuilder) WithTotalPrice(totalPrice float64) *OrderBuilder {
	b.order.TotalPrice = totalPrice
	return b
}

// Build tạo order hoàn chỉnh
func (b *OrderBuilder) Build() *models.Order {
	return b.order
}
