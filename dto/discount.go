package dto

import "time"

// DiscountResponse là DTO cho response của discount
type DiscountResponse struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Quantity  int       `json:"quantity"`
	FromDate  string    `json:"fromDate"`
	ToDate    string    `json:"toDate"`
	Discount  int       `json:"discount"`
	Status    int       `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// CreateDiscountRequest là DTO cho yêu cầu tạo mới discount
type CreateDiscountRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description" binding:"required"`
	Quantity    int    `json:"quantity" binding:"required"`
	FromDate    string `json:"fromDate" binding:"required"`
	ToDate      string `json:"toDate" binding:"required"`
	Discount    int    `json:"discount" binding:"required"`
}

// UpdateDiscountRequest là DTO cho yêu cầu cập nhật discount
type UpdateDiscountRequest struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Quantity    int    `json:"quantity"`
	FromDate    string `json:"fromDate"`
	ToDate      string `json:"toDate"`
	Discount    int    `json:"discount"`
}

// ChangeDiscountStatusRequest là DTO cho yêu cầu thay đổi trạng thái discount
type ChangeDiscountStatusRequest struct {
	ID     uint `json:"id"`
	Status int  `json:"status"`
}
