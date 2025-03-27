package constants

// User status
const (
	UserStatusActive   = 1
	UserStatusInactive = 0
)

// Order status
const (
	OrderStatusPending   = 0
	OrderStatusConfirmed = 1
	OrderStatusCompleted = 2
	OrderStatusCancelled = 3
)

// Room status
const (
	RoomStatusAvailable   = 1
	RoomStatusOccupied    = 2
	RoomStatusMaintenance = 3
)

// Payment status
const (
	PaymentStatusPending  = 0
	PaymentStatusSuccess  = 1
	PaymentStatusFailed   = 2
	PaymentStatusRefunded = 3
)
