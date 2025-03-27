package models

import "errors"

// OrderState định nghĩa interface cho các trạng thái order
type OrderState interface {
	Confirm(order *Order) error
	Cancel(order *Order) error
	Complete(order *Order) error
}

// PendingState trạng thái chờ xác nhận
type PendingState struct{}

func (s *PendingState) Confirm(order *Order) error {
	order.Status = OrderStatusConfirmed
	return nil
}

func (s *PendingState) Cancel(order *Order) error {
	order.Status = OrderStatusCancelled
	return nil
}

func (s *PendingState) Complete(order *Order) error {
	return errors.New("cannot complete pending order")
}

// ConfirmedState trạng thái đã xác nhận
type ConfirmedState struct{}

func (s *ConfirmedState) Confirm(order *Order) error {
	return errors.New("order already confirmed")
}

func (s *ConfirmedState) Cancel(order *Order) error {
	order.Status = OrderStatusCancelled
	return nil
}

func (s *ConfirmedState) Complete(order *Order) error {
	order.Status = OrderStatusCompleted
	return nil
}

// CompletedState trạng thái hoàn thành
type CompletedState struct{}

func (s *CompletedState) Confirm(order *Order) error {
	return errors.New("order already completed")
}

func (s *CompletedState) Cancel(order *Order) error {
	return errors.New("cannot cancel completed order")
}

func (s *CompletedState) Complete(order *Order) error {
	return errors.New("order already completed")
}

// CancelledState trạng thái đã hủy
type CancelledState struct{}

func (s *CancelledState) Confirm(order *Order) error {
	return errors.New("cannot confirm cancelled order")
}

func (s *CancelledState) Cancel(order *Order) error {
	return errors.New("order already cancelled")
}

func (s *CancelledState) Complete(order *Order) error {
	return errors.New("cannot complete cancelled order")
}

// GetOrderState trả về state tương ứng với trạng thái order
func GetOrderState(status int) OrderState {
	switch status {
	case OrderStatusPending:
		return &PendingState{}
	case OrderStatusConfirmed:
		return &ConfirmedState{}
	case OrderStatusCompleted:
		return &CompletedState{}
	case OrderStatusCancelled:
		return &CancelledState{}
	default:
		return &PendingState{}
	}
}
