package services

import (
	"new/models"
)

// SendConfirmation gửi email xác nhận
func (s *NotificationService) SendConfirmation(order *models.Order) error {
	// TODO: Implement email sending logic
	return nil
}

// SendCancellation gửi thông báo hủy
func (s *NotificationService) SendCancellation(order *models.Order) error {
	// TODO: Implement cancellation notification logic
	return nil
}

// SendCompletion gửi thông báo hoàn thành
func (s *NotificationService) SendCompletion(order *models.Order) error {
	// TODO: Implement completion notification logic
	return nil
}
