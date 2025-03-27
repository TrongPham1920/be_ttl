package models

type Bank struct {
	BankId        uint   `json:"bankId" gorm:"primaryKey"`
	UserId        uint   `json:"userId"`
	BankName      string `json:"bankName" gorm:"not null"`
	AccountNumber string `json:"accountNumber" gorm:"not null;unique"`
	BankShortName string `json:"bankShortName" gorm:"not null"`
}
