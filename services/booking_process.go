package services

import (
	"errors"
	"new/models"
)

// BookingProcess định nghĩa interface cho quy trình đặt phòng
type BookingProcess interface {
	ValidateBooking() error
	ProcessPayment() error
	SendConfirmation() error
}

// BaseBookingProcess định nghĩa cấu trúc cơ bản cho quy trình đặt phòng
type BaseBookingProcess struct {
	order *models.Order
}

// StandardBooking quy trình đặt phòng tiêu chuẩn
type StandardBooking struct {
	BaseBookingProcess
}

func NewStandardBooking(order *models.Order) *StandardBooking {
	return &StandardBooking{
		BaseBookingProcess: BaseBookingProcess{
			order: order,
		},
	}
}

func (b *StandardBooking) ValidateBooking() error {
	// Validate thông tin đặt phòng
	if b.order.UserID == nil {
		return errors.New("user ID is required")
	}
	if len(b.order.RoomID) == 0 {
		return errors.New("at least one room is required")
	}
	return nil
}

func (b *StandardBooking) ProcessPayment() error {
	// Xử lý thanh toán
	// TODO: Implement payment processing logic
	return nil
}

func (b *StandardBooking) SendConfirmation() error {
	// Gửi email xác nhận
	// TODO: Implement email sending logic
	return nil
}

// ExpressBooking quy trình đặt phòng nhanh
type ExpressBooking struct {
	BaseBookingProcess
}

func NewExpressBooking(order *models.Order) *ExpressBooking {
	return &ExpressBooking{
		BaseBookingProcess: BaseBookingProcess{
			order: order,
		},
	}
}

func (b *ExpressBooking) ValidateBooking() error {
	// Validate thông tin đặt phòng nhanh
	if b.order.UserID == nil {
		return errors.New("user ID is required")
	}
	if len(b.order.RoomID) == 0 {
		return errors.New("at least one room is required")
	}
	return nil
}

func (b *ExpressBooking) ProcessPayment() error {
	// Xử lý thanh toán nhanh
	// TODO: Implement express payment processing logic
	return nil
}

func (b *ExpressBooking) SendConfirmation() error {
	// Gửi SMS xác nhận
	// TODO: Implement SMS sending logic
	return nil
}
