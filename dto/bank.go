package dto

import (
	"encoding/json"
	"fmt"
)

// BankRequest định nghĩa request tạo ngân hàng
type BankRequest struct {
	BankName       string          `json:"bankName" binding:"required"`
	BankShortName  string          `json:"bankShortName" binding:"required"`
	AccountNumbers json.RawMessage `json:"accountNumbers" binding:"required"`
	Icon           string          `json:"icon" binding:"required"`
}

// Validate kiểm tra tính hợp lệ của request
func (b *BankRequest) Validate() error {
	if b.BankName == "" {
		return fmt.Errorf("tên ngân hàng không được để trống")
	}

	if b.BankShortName == "" {
		return fmt.Errorf("tên viết tắt ngân hàng không được để trống")
	}

	if len(b.AccountNumbers) == 0 {
		return fmt.Errorf("danh sách số tài khoản không được để trống")
	}

	var accountNumbers []string
	if err := json.Unmarshal(b.AccountNumbers, &accountNumbers); err != nil {
		return fmt.Errorf("định dạng danh sách số tài khoản không hợp lệ: %v", err)
	}

	validBanks := map[string]bool{
		"SACOMBANK": true, "VIETINBANK": true, "VCB": true, "AGRIBANK": true,
		"MB": true, "TCB": true, "BIDV": true, "ACB": true, "SCB": true, "VPBANK": true,
	}
	if _, exists := validBanks[b.BankShortName]; !exists {
		return fmt.Errorf("ngân hàng không hợp lệ: %s", b.BankShortName)
	}

	for _, account := range accountNumbers {
		if len(account) < 8 || len(account) > 17 {
			return fmt.Errorf("số tài khoản phải có từ 8 đến 17 chữ số")
		}
	}

	return nil
}

// AddAccountNumbersRequest định nghĩa request thêm số tài khoản
type AddAccountNumbersRequest struct {
	BankID         uint            `json:"bankId" binding:"required"`
	AccountNumbers json.RawMessage `json:"accountNumbers" binding:"required"`
}

// BankResponse định nghĩa response cho ngân hàng
type BankResponse struct {
	ID             uint            `json:"id"`
	BankName       string          `json:"bankName"`
	BankShortName  string          `json:"bankShortName"`
	AccountNumbers json.RawMessage `json:"accountNumbers"`
	Icon           string          `json:"icon"`
}
