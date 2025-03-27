package models

import (
	"fmt"
	"time"
)

type Discount struct {
	ID          uint      `json:"id" gorm:"primaryKey"`            // ID cho giảm giá
	Name        string    `json:"name"`                            // Tên của chương trình giảm giá
	Description string    `json:"description"`                     // code mã giảm
	Quantity    int       `json:"quantity"`                        // Số lượng giảm giá
	FromDate    string    `json:"fromDate"`                        // Ngày bắt đầu chương trình giảm giá
	ToDate      string    `json:"toDate"`                          // Ngày kết thúc chương trình giảm giá
	Discount    int       `json:"discount"`                        // Mức giảm giá (từ 50 trở xuống)
	Status      int       `json:"status" gorm:"default:1"`         // Trạng thái của chương trình (ví dụ: Active, Inactive)
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"createdAt"` // Thời gian tạo
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updatedAt"` // Thời gian cập nhật
}

func (b *Discount) ValidateStatusDiscount() error {
	if b.Status < 0 || b.Status > 1 {
		return fmt.Errorf("invalid Status: %d, must be either 0 or 1", b.Status)
	}
	return nil
}
