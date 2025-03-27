package dto

import "time"

type WithdrawalHistoryResponse struct {
	ID        uint      `json:"id"`
	Amount    int64     `json:"amount"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	User      Actor     `json:"user"`
	Reason    string    `json:"reason"`
}

type WithdrawalHistoryListResponse struct {
	Data  []WithdrawalHistoryResponse `json:"data"`
	Page  int                         `json:"page"`
	Limit int                         `json:"limit"`
	Total int64                       `json:"total"`
}

type CreateWithdrawalRequest struct {
	Amount int64 `json:"amount" binding:"required,min=1"`
}

type UpdateWithdrawalStatusRequest struct {
	ID     uint   `json:"id" binding:"required"`
	Status string `json:"status" binding:"required,min=0,max=2"`
	Reason string `json:"reason"`
}

type WithdrawalHistoryFilter struct {
	UserID   uint   `form:"userId,default=0"`
	Status   int    `form:"status,default=0"`
	FromDate string `form:"fromDate,default=''"`
	ToDate   string `form:"toDate,default=''"`
	Page     int    `form:"page,default=0"`
	Limit    int    `form:"limit,default=10"`
}
