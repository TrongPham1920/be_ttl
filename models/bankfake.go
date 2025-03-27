package models

import (
	"encoding/json"
	"fmt"

	"github.com/go-playground/validator/v10"
)

type BankFake struct {
	ID             uint            `gorm:"primaryKey" json:"id"`
	BankName       string          `json:"bankName" gorm:"not null"`
	BankShortName  string          `json:"bankShortName" gorm:"not null"`
	AccountNumbers json.RawMessage `json:"accountNumbers" gorm:"type:json" validate:"required"`
	Icon           string          `json:"icon" gorm:"not null"`
}

func validateAccountNumber(bankShortName string, accountNumbers json.RawMessage) error {
	var accounts []string
	if err := json.Unmarshal(accountNumbers, &accounts); err != nil {
		return fmt.Errorf("định dạng số tài khoản không hợp lệ: %v", err)
	}

	validBanks := map[string]bool{
		"SACOMBANK": true, "VIETINBANK": true, "VCB": true, "AGRIBANK": true,
		"MB": true, "TCB": true, "BIDV": true, "ACB": true, "SCB": true, "VPBANK": true,
	}
	if _, exits := validBanks[bankShortName]; !exits {
		return fmt.Errorf("ngân hàng không hợp lệ: %s", bankShortName)
	}

	for _, account := range accounts {
		length := len(account)
		if length < 8 || length > 17 {
			return fmt.Errorf("số tài khoản phải có từ 8 đến 17 chữ số")
		}
	}
	return nil
}

func (b *BankFake) Validate() error {
	validate := validator.New()

	if err := validate.Struct(b); err != nil {
		return err
	}

	return validateAccountNumber(b.BankShortName, b.AccountNumbers)
}
