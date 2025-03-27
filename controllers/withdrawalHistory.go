package controllers

import (
	"sort"
	"strconv"
	"strings"
	"unicode"

	"new/config"
	"new/dto"
	"new/models"
	"new/response"

	"github.com/gin-gonic/gin"
	"golang.org/x/text/unicode/norm"
)

// type WithdrawalHistoryInput struct {
// 	Amount int64 `json:"amount" binding:"required"`
// }

// type WithdrawalHistoryResponse struct {
// 	ID        uint      `json:"id"`
// 	Amount    int64     `json:"amount"`
// 	Status    string    `json:"status"`
// 	CreatedAt time.Time `json:"createdAt"`
// 	UpdatedAt time.Time `json:"updatedAt"`
// 	User      dto.Actor `json:"user"`
// 	Reason    string    `json:"reason"`
// }

// Bỏ dấu viết thường
func removeDiacritics(s string) string {
	// Chuẩn hóa chuỗi về dạng NFD (Normalization Form Decomposition)
	t := norm.NFD.String(s)
	var b strings.Builder
	for _, r := range t {
		// Loại bỏ các ký tự dấu (non-spacing marks)
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// CreateWithdrawalHistory tạo một lịch sử rút tiền mới
func CreateWithdrawalHistory(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		response.Unauthorized(c)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	currentUserID, currentUserRole, err := GetUserIDFromToken(tokenString)
	if err != nil {
		response.Unauthorized(c)
		return
	}

	if currentUserRole != 2 {
		response.Forbidden(c)
		return
	}

	var input dto.CreateWithdrawalRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var user models.User
	if err := config.DB.First(&user, currentUserID).Error; err != nil {
		response.ServerError(c)
		return
	}

	// Tính số tiền cho phép rút: nhỏ hơn 80% số dư hiện có của user
	allowedWithdrawal := user.Amount * 80 / 100
	if input.Amount >= allowedWithdrawal {
		response.BadRequest(c, "Số tiền rút phải nhỏ hơn 20% số dư của bạn")
		return
	}

	withdrawal := models.WithdrawalHistory{
		UserID: currentUserID,
		Amount: input.Amount,
	}

	if err := config.DB.Create(&withdrawal).Error; err != nil {
		response.ServerError(c)
		return
	}

	response.Success(c, withdrawal)
}

func GetWithdrawalHistory(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		response.Unauthorized(c)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	currentUserID, currentUserRole, err := GetUserIDFromToken(tokenString)
	if err != nil {
		response.Unauthorized(c)
		return
	}

	if currentUserRole != 1 && currentUserRole != 2 {
		response.Forbidden(c)
		return
	}

	var withdrawals []models.WithdrawalHistory
	dbQuery := config.DB.Preload("User").Preload("User.Banks")
	if currentUserRole == 2 {
		dbQuery = dbQuery.Where("user_id = ?", currentUserID)
	}

	if err := dbQuery.Find(&withdrawals).Error; err != nil {
		response.ServerError(c)
		return
	}

	// Chuyển đổi dữ liệu thành responses
	var responses []dto.WithdrawalHistoryResponse
	for _, w := range withdrawals {
		resp := dto.WithdrawalHistoryResponse{
			ID:        w.ID,
			Amount:    w.Amount,
			Status:    w.Status,
			CreatedAt: w.CreatedAt,
			UpdatedAt: w.UpdatedAt,
			Reason:    w.Reason,
			User: dto.Actor{
				Name:        w.User.Name,
				Email:       w.User.Email,
				PhoneNumber: w.User.PhoneNumber,
			},
		}

		if len(w.User.Banks) > 0 {
			resp.User.BankShortName = w.User.Banks[0].BankShortName
			resp.User.AccountNumber = w.User.Banks[0].AccountNumber
			resp.User.BankName = w.User.Banks[0].BankName
		}
		responses = append(responses, resp)
	}

	statusFilter := c.Query("status")
	if statusFilter != "" {
		var filtered []dto.WithdrawalHistoryResponse
		for _, resp := range responses {
			if resp.Status == statusFilter {
				filtered = append(filtered, resp)
			}
		}
		responses = filtered
	}

	nameFilter := c.Query("name")
	if nameFilter != "" {
		var filtered []dto.WithdrawalHistoryResponse
		normalizedFilter := removeDiacritics(strings.ToLower(strings.ReplaceAll(nameFilter, " ", "")))
		for _, resp := range responses {
			normalizedName := removeDiacritics(strings.ToLower(strings.ReplaceAll(resp.User.Name, " ", "")))
			normalizedPhone := removeDiacritics(strings.ToLower(strings.ReplaceAll(resp.User.PhoneNumber, " ", "")))
			if strings.Contains(normalizedName, normalizedFilter) || strings.Contains(normalizedPhone, normalizedFilter) {
				filtered = append(filtered, resp)
			}
		}
		responses = filtered
	}

	pageStr := c.Query("page")
	limitStr := c.Query("limit")

	page := 0
	limit := 10

	if pageStr != "" {
		if parsedPage, err := strconv.Atoi(pageStr); err == nil && parsedPage >= 0 {
			page = parsedPage
		}
	}

	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	sort.Slice(responses, func(i, j int) bool {
		return responses[i].CreatedAt.After(responses[j].CreatedAt)
	})

	total := len(responses)
	start := page * limit
	end := start + limit

	if start >= total {
		responses = []dto.WithdrawalHistoryResponse{}
	} else if end > total {
		responses = responses[start:]
	} else {
		responses = responses[start:end]
	}

	response.SuccessWithPagination(c, responses, page, limit, total)
}

func ConfirmWithdrawalHistory(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		response.Unauthorized(c)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	_, currentUserRole, err := GetUserIDFromToken(tokenString)
	if err != nil {
		response.Unauthorized(c)
		return
	}

	if currentUserRole != 1 {
		response.Forbidden(c)
		return
	}

	var input dto.UpdateWithdrawalStatusRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var withdrawal models.WithdrawalHistory
	if err := config.DB.First(&withdrawal, input.ID).Error; err != nil {
		response.NotFound(c)
		return
	}

	if input.Status == "1" {
		var user models.User
		if err := config.DB.First(&user, withdrawal.UserID).Error; err != nil {
			response.NotFound(c)
			return
		}

		user.Amount = user.Amount - withdrawal.Amount

		if err := config.DB.Save(&user).Error; err != nil {
			response.ServerError(c)
			return
		}
	}

	if input.Status == "2" && strings.TrimSpace(input.Reason) == "" {
		response.BadRequest(c, "Phải có lý do khi hủy giao dịch (Status = 2)")
		return
	}

	withdrawal.Status = input.Status
	if input.Status == "2" {
		withdrawal.Reason = input.Reason
	}

	if err := config.DB.Save(&withdrawal).Error; err != nil {
		response.ServerError(c)
		return
	}

	response.Success(c, withdrawal)
}
