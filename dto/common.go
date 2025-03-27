package dto

import "new/response"

// PaginatedResponse là struct chung cho các response có phân trang
type PaginatedResponse[T any] struct {
	Data       T                   `json:"data"`
	Pagination response.Pagination `json:"pagination"`
}
