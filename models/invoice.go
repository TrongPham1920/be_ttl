package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Invoice struct {
	ID              uint       `json:"id" gorm:"primaryKey"`              // Mã hóa đơn
	InvoiceCode     string     `json:"invoiceCode" gorm:"unique;size:20"` // Mã hóa đơn duy nhất
	OrderID         uint       `json:"orderId"`                           // Liên kết với Order
	Order           Order      `json:"order" gorm:"foreignKey:OrderID"`
	TotalAmount     float64    `json:"totalAmount"`           // Tổng số tiền từ Order
	PaidAmount      float64    `json:"paidAmount"`            // Số tiền đã thanh toán
	RemainingAmount float64    `json:"remainingAmount"`       // Số tiền còn phải thanh toán
	Status          int        `json:"status"`                // 0: Chưa thanh toán, 1: Đã thanh toán
	PaymentDate     *time.Time `json:"paymentDate,omitempty"` // Ngày thanh toán
	PaymentType     *int       `json:"paymentType"`           // 0: tiền mặt , 1: ck ngân hàng, 2:momo
	CreatedAt       time.Time  `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt       time.Time  `gorm:"autoUpdateTime" json:"updatedAt"`
	AdminID         uint       `json:"adminId" `
}

func (invoice *Invoice) BeforeCreate(tx *gorm.DB) (err error) {
	invoice.InvoiceCode = fmt.Sprintf("TTL%d", time.Now().Unix())

	var count int64
	if err := tx.Model(&Invoice{}).Where("invoice_code = ?", invoice.InvoiceCode).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("InvoiceCode đã tồn tại, hãy thử lại")
	}
	return nil
}
