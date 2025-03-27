package commands

import (
	"new/models"

	"gorm.io/gorm"
)

// OrderCommand định nghĩa interface cho các command
type OrderCommand interface {
	Execute() error
}

// CreateOrderCommand command để tạo order mới
type CreateOrderCommand struct {
	order *models.Order
	db    *gorm.DB
}

func NewCreateOrderCommand(order *models.Order, db *gorm.DB) *CreateOrderCommand {
	return &CreateOrderCommand{
		order: order,
		db:    db,
	}
}

func (c *CreateOrderCommand) Execute() error {
	return c.db.Create(c.order).Error
}

// UpdateOrderCommand command để cập nhật order
type UpdateOrderCommand struct {
	order *models.Order
	db    *gorm.DB
}

func NewUpdateOrderCommand(order *models.Order, db *gorm.DB) *UpdateOrderCommand {
	return &UpdateOrderCommand{
		order: order,
		db:    db,
	}
}

func (c *UpdateOrderCommand) Execute() error {
	return c.db.Save(c.order).Error
}

// DeleteOrderCommand command để xóa order
type DeleteOrderCommand struct {
	orderID uint
	db      *gorm.DB
}

func NewDeleteOrderCommand(orderID uint, db *gorm.DB) *DeleteOrderCommand {
	return &DeleteOrderCommand{
		orderID: orderID,
		db:      db,
	}
}

func (c *DeleteOrderCommand) Execute() error {
	return c.db.Delete(&models.Order{}, c.orderID).Error
}
