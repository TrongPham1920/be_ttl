package services

import (
	"errors"
	"new/models"
)

// Validate kiểm tra tính hợp lệ của order
func (s *OrderService) Validate(order *models.Order) error {
	if order.UserID == nil {
		return errors.New("user ID is required")
	}
	if len(order.RoomID) == 0 {
		return errors.New("at least one room is required")
	}
	return nil
}

// Create tạo order mới
func (s *OrderService) Create(order *models.Order) error {
	return s.db.Create(order).Error
}

// GetByID lấy order theo ID
func (s *OrderService) GetByID(orderID uint) (*models.Order, error) {
	var order models.Order
	if err := s.db.First(&order, orderID).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

// Cancel hủy order
func (s *OrderService) Cancel(order *models.Order) error {
	order.Status = models.OrderStatusCancelled
	return s.db.Save(order).Error
}

// Complete hoàn thành order
func (s *OrderService) Complete(order *models.Order) error {
	order.Status = models.OrderStatusCompleted
	return s.db.Save(order).Error
}
