package services

import (
	"new/models"

	"gorm.io/gorm"
)

// BookingFacade đơn giản hóa việc tương tác với các service
type BookingFacade struct {
	orderService        *OrderService
	paymentService      *PaymentService
	notificationService *NotificationService
}

// OrderService xử lý logic liên quan đến order
type OrderService struct {
	db *gorm.DB
}

// PaymentService xử lý logic thanh toán
type PaymentService struct {
	db *gorm.DB
}

// NotificationService xử lý logic gửi thông báo
type NotificationService struct {
	db *gorm.DB
}

// NewBookingFacade tạo instance mới của BookingFacade
func NewBookingFacade(db *gorm.DB) *BookingFacade {
	return &BookingFacade{
		orderService: &OrderService{
			db: db,
		},
		paymentService: &PaymentService{
			db: db,
		},
		notificationService: &NotificationService{
			db: db,
		},
	}
}

// CreateBooking tạo booking mới
func (f *BookingFacade) CreateBooking(order *models.Order) error {
	// Validate order
	if err := f.orderService.Validate(order); err != nil {
		return err
	}

	// Process payment
	if err := f.paymentService.Process(order); err != nil {
		return err
	}

	// Create order
	if err := f.orderService.Create(order); err != nil {
		return err
	}

	// Send notification
	if err := f.notificationService.SendConfirmation(order); err != nil {
		// Log error but don't fail the booking
		// TODO: Implement logging
	}

	return nil
}

// CancelBooking hủy booking
func (f *BookingFacade) CancelBooking(orderID uint) error {
	// Get order
	order, err := f.orderService.GetByID(orderID)
	if err != nil {
		return err
	}

	// Cancel order
	if err := f.orderService.Cancel(order); err != nil {
		return err
	}

	// Process refund if needed
	if err := f.paymentService.Refund(order); err != nil {
		// Log error but don't fail the cancellation
		// TODO: Implement logging
	}

	// Send cancellation notification
	if err := f.notificationService.SendCancellation(order); err != nil {
		// Log error but don't fail the cancellation
		// TODO: Implement logging
	}

	return nil
}

// CompleteBooking hoàn thành booking
func (f *BookingFacade) CompleteBooking(orderID uint) error {
	// Get order
	order, err := f.orderService.GetByID(orderID)
	if err != nil {
		return err
	}

	// Complete order
	if err := f.orderService.Complete(order); err != nil {
		return err
	}

	// Send completion notification
	if err := f.notificationService.SendCompletion(order); err != nil {
		// Log error but don't fail the completion
		// TODO: Implement logging
	}

	return nil
}
