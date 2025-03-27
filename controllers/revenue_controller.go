package controllers

import (
	"database/sql"
	"fmt"
	"new/config"
	"new/dto"
	"new/models"
	"new/response"
	"new/services"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func GetTotalRevenue(c *gin.Context) {
	var totalRevenue, currentMonthRevenue, currentWeekRevenue float64
	var lastMonthRevenue sql.NullFloat64
	var monthlyRevenue []dto.MonthRevenue
	var vat, actualMonthlyRevenue float64

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

	tx := config.DB.Model(&models.Invoice{})
	if currentUserRole == 2 {
		tx = tx.Where("order_id IN (?)", config.DB.Table("orders").
			Select("orders.id").
			Joins("JOIN accommodations ON accommodations.id = orders.accommodation_id").
			Where("accommodations.user_id = ?", currentUserID))
	}

	var invoices []models.Invoice
	if err := tx.Find(&invoices).Error; err != nil {
		response.ServerError(c)
		return
	}

	for _, invoice := range invoices {
		totalRevenue += invoice.TotalAmount

		currentMonth := time.Now().Format("2006-01")
		if invoice.CreatedAt.Format("2006-01") == currentMonth {
			currentMonthRevenue += invoice.TotalAmount
		}

		lastMonth := time.Now().AddDate(0, -1, 0).Format("2006-01")
		if invoice.CreatedAt.Format("2006-01") == lastMonth {
			lastMonthRevenue.Float64 += invoice.TotalAmount
		}

		currentWeekStart := time.Now().AddDate(0, 0, -int(time.Now().Weekday()))
		currentWeekEnd := currentWeekStart.AddDate(0, 0, 6)
		if invoice.CreatedAt.After(currentWeekStart) && invoice.CreatedAt.Before(currentWeekEnd) {
			currentWeekRevenue += invoice.TotalAmount
		}
	}

	currentYear := time.Now().Year()
	for i := 1; i <= 12; i++ {
		month := fmt.Sprintf("%d-%02d", currentYear, i)
		var revenue, orderCount float64

		for _, invoice := range invoices {
			if invoice.CreatedAt.Format("2006-01") == month {
				revenue += invoice.TotalAmount
				orderCount++
			}
		}

		monthlyRevenue = append(monthlyRevenue, dto.MonthRevenue{
			Month:      fmt.Sprintf("Tháng %d", i),
			Revenue:    revenue,
			OrderCount: int(orderCount),
		})
	}

	if currentUserRole == 1 {
		totalRevenue *= 0.30
		currentMonthRevenue *= 0.30
		lastMonthRevenue.Float64 *= 0.30
		currentWeekRevenue *= 0.30
		for i := range monthlyRevenue {
			monthlyRevenue[i].Revenue *= 0.30
		}
	} else if currentUserRole == 2 {
		vat = currentMonthRevenue * 30 / 100
		actualMonthlyRevenue = currentMonthRevenue - vat
		totalRevenue -= (totalRevenue * 30 / 100)
	}

	responseData := dto.RevenueResponse{
		TotalRevenue:         totalRevenue,
		CurrentMonthRevenue:  currentMonthRevenue,
		LastMonthRevenue:     lastMonthRevenue.Float64,
		CurrentWeekRevenue:   currentWeekRevenue,
		MonthlyRevenue:       monthlyRevenue,
		VAT:                  vat,
		ActualMonthlyRevenue: actualMonthlyRevenue,
	}

	response.Success(c, responseData)
}

func GetTotal(c *gin.Context) {
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

	nameFilter := c.Query("name")

	var users []models.User

	query := config.DB.Where("role = ?", 2)

	if nameFilter != "" {
		query = query.Where("name ILIKE ? OR email ILIKE ? OR phone_number ILIKE ?", "%"+nameFilter+"%", "%"+nameFilter+"%", "%"+nameFilter+"%")
	}

	if err := query.Find(&users).Error; err != nil {
		response.ServerError(c)
		return
	}

	calculateRevenue := func(userID uint) (float64, float64, float64, float64, float64, float64, float64, error) {
		var totalAmount, currentMonthRevenue, lastMonthRevenue, currentWeekRevenue float64

		if err := config.DB.Model(&models.Invoice{}).
			Where("admin_id = ?", userID).
			Select("COALESCE(SUM(total_amount), 0)").
			Scan(&totalAmount).Error; err != nil {
			return 0, 0, 0, 0, 0, 0, 0, nil
		}

		if err := config.DB.Model(&models.Invoice{}).
			Where("admin_id = ? AND EXTRACT(MONTH FROM created_at) = EXTRACT(MONTH FROM CURRENT_DATE) AND EXTRACT(YEAR FROM created_at) = EXTRACT(YEAR FROM CURRENT_DATE)", userID).
			Select("COALESCE(SUM(total_amount), 0)").
			Scan(&currentMonthRevenue).Error; err != nil {
			return 0, 0, 0, 0, 0, 0, 0, nil
		}

		if err := config.DB.Model(&models.Invoice{}).
			Where("admin_id = ? AND EXTRACT(MONTH FROM created_at) = EXTRACT(MONTH FROM CURRENT_DATE - INTERVAL '1 MONTH') AND EXTRACT(YEAR FROM created_at) = EXTRACT(YEAR FROM CURRENT_DATE)", userID).
			Select("COALESCE(SUM(total_amount), 0)").
			Scan(&lastMonthRevenue).Error; err != nil {
			return 0, 0, 0, 0, 0, 0, 0, nil
		}

		if err := config.DB.Model(&models.Invoice{}).
			Where("admin_id = ? AND EXTRACT(WEEK FROM created_at) = EXTRACT(WEEK FROM CURRENT_DATE) AND EXTRACT(YEAR FROM created_at) = EXTRACT(YEAR FROM CURRENT_DATE)", userID).
			Select("COALESCE(SUM(total_amount), 0)").
			Scan(&currentWeekRevenue).Error; err != nil {
			return 0, 0, 0, 0, 0, 0, 0, nil
		}

		vat := currentMonthRevenue * 0.3
		vatLastMonth := lastMonthRevenue * 0.3
		actualMonthlyRevenue := currentMonthRevenue - vat

		return totalAmount, currentMonthRevenue, lastMonthRevenue, currentWeekRevenue, vat, vatLastMonth, actualMonthlyRevenue, nil
	}

	var totalResponses []dto.TotalResponse
	for _, user := range users {
		totalAmount, currentMonthRevenue, lastMonthRevenue, currentWeekRevenue, vat, vatLastMonth, actualMonthlyRevenue, err := calculateRevenue(user.ID)

		if err != nil {
			response.ServerError(c)
			return
		}

		totalResponses = append(totalResponses, dto.TotalResponse{
			User: dto.InvoiceUserResponse{
				ID:          user.ID,
				Email:       user.Email,
				PhoneNumber: user.PhoneNumber,
			},
			TotalAmount:          totalAmount,
			CurrentMonthRevenue:  currentMonthRevenue,
			LastMonthRevenue:     lastMonthRevenue,
			CurrentWeekRevenue:   currentWeekRevenue,
			VAT:                  vat,
			VatLastMonth:         vatLastMonth,
			ActualMonthlyRevenue: actualMonthlyRevenue,
		})
	}

	response.Success(c, totalResponses)
}

func GetToday(c *gin.Context) {
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

	now := time.Now()
	year, month, _ := now.Date()
	loc := now.Location()
	firstDay := time.Date(year, month, 1, 0, 0, 0, 0, loc)
	lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, loc)

	var revenues []models.UserRevenue
	if err := config.DB.
		Where("user_id = ? ", currentUserID).
		Find(&revenues).Error; err != nil {
		response.ServerError(c)
		return
	}

	revenueMap := make(map[string]models.UserRevenue)
	for _, rev := range revenues {
		dateStr := rev.Date.Format("2006-01-02")
		revenueMap[dateStr] = rev
	}

	var result []gin.H
	for d := firstDay; !d.After(lastDay); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		if rev, ok := revenueMap[dateStr]; ok {
			result = append(result, gin.H{
				"date":        dateStr,
				"order_count": rev.OrderCount,
				"revenue":     rev.Revenue,
				"user_id":     rev.UserID,
			})
		} else {
			result = append(result, gin.H{
				"date":        dateStr,
				"order_count": 0,
				"revenue":     0,
				"user_id":     currentUserID,
			})
		}
	}

	response.Success(c, result)
}

func GetTodayUser(c *gin.Context) {
	revenues, err := services.GetTodayUserRevenue()
	if err != nil {
		response.ServerError(c)
		return
	}

	response.Success(c, revenues)
}

func GetUserRevene(c *gin.Context) {
	fromDateStr := c.Query("fromDate")
	toDateStr := c.Query("toDate")
	nameFilter := c.Query("name")
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

	dbQuery := config.DB.Preload("User")

	if fromDateStr != "" {
		fromDate, err := time.Parse("02/01/2006", fromDateStr)
		if err != nil {
			response.BadRequest(c, "fromDate không hợp lệ, định dạng: dd/mm/yyyy")
			return
		}
		dbQuery = dbQuery.Where("date >= ?", fromDate)
	}

	if toDateStr != "" {
		toDate, err := time.Parse("02/01/2006", toDateStr)
		if err != nil {
			response.BadRequest(c, "toDate không hợp lệ, định dạng: dd/mm/yyyy")
			return
		}
		dbQuery = dbQuery.Where("date <= ?", toDate)
	}

	var revenues []models.UserRevenue
	if err := dbQuery.Find(&revenues).Error; err != nil {
		response.ServerError(c)
		return
	}

	var responses []dto.UserRevenueResponse
	for _, rev := range revenues {
		var resp dto.UserRevenueResponse
		resp.ID = rev.ID
		resp.Date = rev.Date.Format("2006-01-02")
		resp.OrderCount = rev.OrderCount
		resp.Revenue = rev.Revenue

		resp.User.ID = rev.User.ID
		resp.User.Name = rev.User.Name
		resp.User.Email = rev.User.Email
		resp.User.PhoneNumber = rev.User.PhoneNumber

		responses = append(responses, resp)
	}

	if nameFilter != "" {
		var filtered []dto.UserRevenueResponse
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

	sort.Slice(responses, func(i, j int) bool {
		t1, _ := time.Parse("2006-01-02", responses[i].Date)
		t2, _ := time.Parse("2006-01-02", responses[j].Date)
		return t1.After(t2)
	})

	total := len(responses)
	start := page * limit
	end := start + limit

	if start >= total {
		responses = []dto.UserRevenueResponse{}
	} else if end > total {
		responses = responses[start:]
	} else {
		responses = responses[start:end]
	}

	response.SuccessWithPagination(c, responses, page, limit, total)
}
