package controllers

import (
	"context"
	"fmt"
	"log"
	"net/url"
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
	"github.com/redis/go-redis/v9"
)

func GetInvoices(c *gin.Context) {
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

	invoiceCodeFilter := c.Query("invoiceCode")
	statusFilter := c.Query("status")

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

	var cacheKey string
	if currentUserRole == 2 {
		cacheKey = fmt.Sprintf("invoices:admin:%d", currentUserID)
	} else if currentUserRole == 3 {
		cacheKey = fmt.Sprintf("invoices:receptionist:%d", currentUserID)
	} else {
		cacheKey = "invoices:all"
	}

	rdb, err := config.ConnectRedis()
	if err != nil {
		response.ServerError(c)
		return
	}

	var allInvoices []dto.InvoiceResponse

	// Lấy hóa đơn từ cache nếu có
	if err := services.GetFromRedis(config.Ctx, rdb, cacheKey, &allInvoices); err != nil || len(allInvoices) == 0 {
		tx := config.DB.Model(&models.Invoice{})

		if currentUserRole == 2 {
			tx = tx.Where("order_id IN (?)", config.DB.Table("orders").
				Select("orders.id").
				Joins("JOIN accommodations ON accommodations.id = orders.accommodation_id").
				Where("accommodations.user_id = ?", currentUserID))
		} else if currentUserRole == 3 {
			var adminID int
			if err := config.DB.Model(&models.User{}).Select("admin_id").Where("id = ?", currentUserID).Scan(&adminID).Error; err != nil {
				response.Forbidden(c)
				return
			}
			tx = tx.Where("order_id IN (?)", config.DB.Table("orders").
				Select("orders.id").
				Joins("JOIN accommodations ON accommodations.id = orders.accommodation_id").
				Where("accommodations.user_id = ?", adminID))
		}

		var invoices []models.Invoice
		if err := tx.Find(&invoices).Error; err != nil {
			response.ServerError(c)
			return
		}

		// Duyệt qua danh sách hóa đơn để chuyển đổi sang InvoiceResponse và thêm thông tin user/guest
		for _, invoice := range invoices {
			var order models.Order

			if err := config.DB.Where("id = ?", invoice.OrderID).First(&order).Error; err != nil {

				continue
			}

			invoiceResp := dto.InvoiceResponse{
				ID:              invoice.ID,
				InvoiceCode:     invoice.InvoiceCode,
				OrderID:         invoice.OrderID,
				TotalAmount:     invoice.TotalAmount,
				PaidAmount:      invoice.PaidAmount,
				RemainingAmount: invoice.RemainingAmount,
				Status:          invoice.Status,
				PaymentDate:     nil,
				CreatedAt:       invoice.CreatedAt.Format("2006-01-02 15:04:05"),
				UpdatedAt:       invoice.UpdatedAt.Format("2006-01-02 15:04:05"),
				AdminID:         invoice.AdminID,
			}

			if order.UserID != nil {
				var user models.User
				if err := config.DB.Where("id = ?", order.UserID).First(&user).Error; err == nil {
					invoiceResp.User = dto.InvoiceUserResponse{
						ID:          user.ID,
						Name:        user.Name,
						Email:       user.Email,
						PhoneNumber: user.PhoneNumber,
					}
				}
			} else {
				invoiceResp.User = dto.InvoiceUserResponse{
					ID:          0,
					Name:        order.GuestName,
					Email:       order.GuestEmail,
					PhoneNumber: order.GuestPhone,
				}

			}

			allInvoices = append(allInvoices, invoiceResp)
		}

		// Cập nhật cache nếu cần
		if err := services.SetToRedis(config.Ctx, rdb, cacheKey, allInvoices, 60*time.Minute); err != nil {
			log.Printf("Error caching invoices: %v", err)
		}
	}

	// Áp dụng filter theo invoiceCode và status
	filteredInvoices := make([]dto.InvoiceResponse, 0)
	for _, invoice := range allInvoices {
		if invoiceCodeFilter != "" {
			decodedNameFilter, _ := url.QueryUnescape(invoiceCodeFilter)
			if !strings.Contains(strings.ToLower(invoice.InvoiceCode), strings.ToLower(decodedNameFilter)) {
				continue
			}
		}
		if statusFilter != "" {
			status, _ := strconv.Atoi(statusFilter)
			if invoice.Status != status {
				continue
			}
		}
		filteredInvoices = append(filteredInvoices, invoice)
	}

	// Sắp xếp theo CreatedAt giảm dần
	sort.Slice(filteredInvoices, func(i, j int) bool {
		return filteredInvoices[i].CreatedAt > filteredInvoices[j].CreatedAt
	})
	total := len(filteredInvoices)

	// Phân trang
	start := page * limit
	end := start + limit
	if start >= total {
		filteredInvoices = []dto.InvoiceResponse{}
	} else if end > total {
		filteredInvoices = filteredInvoices[start:]
	} else {
		filteredInvoices = filteredInvoices[start:end]
	}

	response.SuccessWithPagination(c, filteredInvoices, page, limit, total)
}

func GetDetailInvoice(c *gin.Context) {
	var invoice models.Invoice
	if err := config.DB.Where("id = ?", c.Param("id")).First(&invoice).Error; err != nil {
		response.NotFound(c)
		return
	}
	var order models.Order
	if err := config.DB.Where("id = ?", invoice.OrderID).First(&order).Error; err != nil {
		response.NotFound(c)
		return
	}
	var user models.User
	if err := config.DB.Where("id = ?", order.UserID).First(&user).Error; err != nil {
		response.NotFound(c)
		return
	}
	invoiceResponse := dto.InvoiceResponse{
		ID:              invoice.ID,
		InvoiceCode:     invoice.InvoiceCode,
		OrderID:         invoice.OrderID,
		TotalAmount:     invoice.TotalAmount,
		PaidAmount:      invoice.PaidAmount,
		RemainingAmount: invoice.RemainingAmount,
		Status:          invoice.Status,
		PaymentDate:     nil,
		CreatedAt:       invoice.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:       invoice.UpdatedAt.Format("2006-01-02 15:04:05"),
		AdminID:         invoice.AdminID,
		User: dto.InvoiceUserResponse{
			ID:          user.ID,
			Email:       user.Email,
			PhoneNumber: user.PhoneNumber,
		},
	}

	response.Success(c, invoiceResponse)
}

func UpdatePaymentStatus(c *gin.Context) {
	var request dto.UpdatePaymentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "Dữ liệu không hợp lệ")
		return
	}

	var invoice models.Invoice
	if err := config.DB.Where("id = ?", request.ID).First(&invoice).Error; err != nil {
		response.NotFound(c)
		return
	}

	invoice.PaymentType = &request.PaymentType
	currentTime := time.Now()
	invoice.PaymentDate = &currentTime
	invoice.Status = 1
	invoice.RemainingAmount = 0
	invoice.PaidAmount = invoice.TotalAmount

	if err := config.DB.Save(&invoice).Error; err != nil {
		response.ServerError(c)
		return
	}

	redisClient, err := config.ConnectRedis()
	if err != nil {
		response.ServerError(c)
		return
	}

	cacheKeyPattern := "invoices:*"
	keys, err := redisClient.Keys(config.Ctx, cacheKeyPattern).Result()
	if err != nil {
		response.ServerError(c)
		return
	}

	if len(keys) > 0 {
		err := redisClient.Del(config.Ctx, keys...).Err()
		if err != nil {
			response.ServerError(c)
			return
		}
	}

	response.Success(c, nil)
}

func SendPay(c *gin.Context) {
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

	var request dto.SendPayRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "Dữ liệu không hợp lệ")
		return
	}

	totalVat := request.Vat + request.VatLastMonth

	qrCodeURL := fmt.Sprintf(
		"https://img.vietqr.io/image/SACOMBANK-060915374450-compact.jpg?amount=%d&addInfo=Chuyen%%20khoan%%20phi%%20",
		totalVat,
	)

	if err := services.SendPayEmail(request.Email, request.Vat, request.VatLastMonth, totalVat, qrCodeURL); err != nil {
		response.ServerError(c)
		return
	}

	response.Success(c, nil)
}

func DeleteKeysByPattern(ctx context.Context, rdb *redis.Client, pattern string) error {
	iter := rdb.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		if err := rdb.Del(ctx, iter.Val()).Err(); err != nil {
			return fmt.Errorf("lỗi khi xóa key %s: %v", iter.Val(), err)
		}
	}
	if err := iter.Err(); err != nil {
		return fmt.Errorf("lỗi khi duyệt các key với pattern %s: %v", pattern, err)
	}
	return nil
}
