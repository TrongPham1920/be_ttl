package controllers

import (
	"encoding/json"
	"fmt"
	"new/config"
	"new/dto"
	"new/models"
	"new/response"
	"strings"

	"github.com/gin-gonic/gin"
)

func CreateBank(c *gin.Context) {
	var bankRequest dto.BankRequest

	if err := c.ShouldBindJSON(&bankRequest); err != nil {
		response.BadRequest(c, "Lỗi khi ràng buộc dữ liệu")
		return
	}

	bankRequest.BankShortName = strings.ToUpper(bankRequest.BankShortName)

	var existingBankByName models.BankFake
	if err := config.DB.Where("bank_name = ?", bankRequest.BankName).First(&existingBankByName).Error; err == nil {
		response.BadRequest(c, "Ngân hàng đã tồn tại")
		return
	}

	var existingBankByShortName models.BankFake
	if err := config.DB.Where("bank_short_name = ?", bankRequest.BankShortName).First(&existingBankByShortName).Error; err == nil {
		response.BadRequest(c, "Tên viết tắt ngân hàng đã tồn tại")
		return
	}

	if err := bankRequest.Validate(); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var accountNumbers []string
	if err := json.Unmarshal(bankRequest.AccountNumbers, &accountNumbers); err != nil {
		response.BadRequest(c, "Lỗi khi giải mã danh sách số tài khoản")
		return
	}

	accountSet := make(map[string]struct{})
	for _, accountNumber := range accountNumbers {
		if _, exists := accountSet[accountNumber]; exists {
			response.BadRequest(c, "Danh sách số tài khoản chứa số tài khoản trùng lặp")
			return
		}
		accountSet[accountNumber] = struct{}{}
	}

	bank := models.BankFake{
		BankName:       bankRequest.BankName,
		BankShortName:  bankRequest.BankShortName,
		AccountNumbers: bankRequest.AccountNumbers,
		Icon:           bankRequest.Icon,
	}

	if err := config.DB.Create(&bank).Error; err != nil {
		response.ServerError(c)
		return
	}

	bankResponse := dto.BankResponse{
		ID:             bank.ID,
		BankName:       bank.BankName,
		BankShortName:  bank.BankShortName,
		AccountNumbers: bank.AccountNumbers,
		Icon:           bank.Icon,
	}

	response.Success(c, bankResponse)
}

func AddAccountNumbers(c *gin.Context) {
	var request dto.AddAccountNumbersRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "Lỗi khi ràng buộc dữ liệu")
		return
	}

	var bank models.BankFake
	if err := config.DB.First(&bank, request.BankID).Error; err != nil {
		response.NotFound(c)
		return
	}

	if request.AccountNumbers != nil {
		var accounts []string
		if err := json.Unmarshal(request.AccountNumbers, &accounts); err != nil {
			response.BadRequest(c, "Định dạng số tài khoản không hợp lệ")
			return
		}

		accountMap := make(map[string]int)
		duplicates := []string{}

		for _, account := range accounts {
			accountMap[account]++
			if accountMap[account] == 2 {
				duplicates = append(duplicates, account)
			}
		}

		if len(duplicates) > 0 {
			response.BadRequest(c, "Có số tài khoản trùng lặp")
			return
		}

		existingAccounts := make([]string, 0)
		for _, account := range accounts {
			var count int64
			err := config.DB.Model(&models.BankFake{}).Where("id = ? AND account_numbers::jsonb @> ?::jsonb", bank.ID, fmt.Sprintf(`["%s"]`, account)).Count(&count).Error
			if err != nil {
				response.ServerError(c)
				return
			}
			if count > 0 {
				existingAccounts = append(existingAccounts, account)
			}
		}

		if len(existingAccounts) > 0 {
			response.BadRequest(c, "Có số tài khoản trùng lặp trong cơ sở dữ liệu")
			return
		}

		var existingAccountNumbers []string
		if err := json.Unmarshal(bank.AccountNumbers, &existingAccountNumbers); err != nil {
			response.ServerError(c)
			return
		}
		existingAccountNumbers = append(existingAccountNumbers, accounts...)
		bank.AccountNumbers, _ = json.Marshal(existingAccountNumbers)

		if err := bank.Validate(); err != nil {
			response.BadRequest(c, "Số tài khoản không hợp lệ")
			return
		}
	}

	if err := config.DB.Save(&bank).Error; err != nil {
		response.ServerError(c)
		return
	}

	response.Success(c, nil)
}

func GetAllBanks(c *gin.Context) {
	var banks []models.BankFake

	if err := config.DB.Find(&banks).Error; err != nil {
		response.ServerError(c)
		return
	}

	var bankResponses []dto.BankResponse
	for _, bank := range banks {
		bankResponses = append(bankResponses, dto.BankResponse{
			ID:             bank.ID,
			BankName:       bank.BankName,
			BankShortName:  bank.BankShortName,
			AccountNumbers: bank.AccountNumbers,
			Icon:           bank.Icon,
		})
	}

	response.Success(c, bankResponses)
}

func DeleteAllBanks(c *gin.Context) {
	if err := config.DB.Exec("DELETE FROM bank_fakes").Error; err != nil {
		response.ServerError(c)
		return
	}

	response.Success(c, nil)
}
