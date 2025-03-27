package services

import (
	"new/models"
)

// Process xử lý thanh toán
func (s *PaymentService) Process(order *models.Order) error {
	// TODO: Implement payment processing logic
	return nil
}

// Refund xử lý hoàn tiền
func (s *PaymentService) Refund(order *models.Order) error {
	// TODO: Implement refund logic
	return nil
}
