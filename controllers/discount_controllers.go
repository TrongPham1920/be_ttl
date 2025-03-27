package controllers

import (
	"net/url"
	"new/config"
	"new/dto"
	"new/models"
	"new/response"
	"new/services"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

var layout = "02/01/2006"

func ConvertDateToComparableFormat(dateStr string) (string, error) {
	parsedDate, err := time.Parse(layout, dateStr)
	if err != nil {
		return "", err
	}
	return parsedDate.Format("20060102"), nil
}

func GetDiscounts(c *gin.Context) {
	var discounts []models.Discount

	if err := config.DB.Find(&discounts).Error; err != nil {
		response.ServerError(c)
		return
	}

	pageStr := c.Query("page")
	limitStr := c.Query("limit")
	statusFilter := c.Query("status")
	nameFilter := c.Query("name")
	discountStr := c.Query("discount")
	quantityStr := c.Query("quantity")
	fromDateStr := c.Query("fromDate")
	toDateStr := c.Query("toDate")
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

	var discountResponses []dto.DiscountResponse

	tx := config.DB.Model(&models.Discount{})
	if nameFilter != "" {
		decodedNameFilter, err := url.QueryUnescape(nameFilter)
		if err != nil {
			response.ServerError(c)
			return
		}
		tx = tx.Where("name ILIKE ?", "%"+decodedNameFilter+"%")
	}
	if statusFilter != "" {
		tx = tx.Where("status = ?", statusFilter)
	}
	if discountStr != "" {
		discount, err := strconv.ParseFloat(discountStr, 64)
		if err == nil {
			tx = tx.Where("discount = ?", discount)
		}
	}
	if quantityStr != "" {
		quantity, err := strconv.ParseFloat(quantityStr, 64)
		if err == nil {
			tx = tx.Where("quantity = ?", quantity)
		}
	}
	if fromDateStr != "" {
		fromDateComparable, err := ConvertDateToComparableFormat(fromDateStr)
		if err != nil {
			response.BadRequest(c, "Sai định dạng fromDate")
			return
		}

		if toDateStr != "" {
			toDateComparable, err := ConvertDateToComparableFormat(toDateStr)
			if err != nil {
				response.BadRequest(c, "Sai định dạng toDate")
				return
			}
			tx = tx.Where("SUBSTRING(from_date, 7, 4) || SUBSTRING(from_date, 4, 2) || SUBSTRING(from_date, 1, 2) >= ? AND SUBSTRING(to_date, 7, 4) || SUBSTRING(to_date, 4, 2) || SUBSTRING(to_date, 1, 2) <= ?", fromDateComparable, toDateComparable)
		} else {
			tx = tx.Where("SUBSTRING(from_date, 7, 4) || SUBSTRING(from_date, 4, 2) || SUBSTRING(from_date, 1, 2) >= ?", fromDateComparable)
		}
	}

	var totalDiscounts int64
	if err := tx.Count(&totalDiscounts).Error; err != nil {
		response.ServerError(c)
		return
	}
	tx = tx.Order("updated_at desc")

	if err := tx.Offset(page * limit).Limit(limit).Find(&discounts).Error; err != nil {
		response.ServerError(c)
		return
	}
	for _, discount := range discounts {
		discountResponses = append(discountResponses, dto.DiscountResponse{
			ID:        discount.ID,
			Name:      discount.Name,
			Quantity:  discount.Quantity,
			FromDate:  discount.FromDate,
			ToDate:    discount.ToDate,
			Discount:  discount.Discount,
			Status:    discount.Status,
			CreatedAt: discount.CreatedAt,
			UpdatedAt: discount.UpdatedAt,
		})
	}

	response.SuccessWithPagination(c, discountResponses, page, limit, int(totalDiscounts))
}

func GetDiscountDetail(c *gin.Context) {
	var discount models.Discount
	discountId := c.Param("id")
	if err := config.DB.Where("id = ?", discountId).First(&discount).Error; err != nil {
		response.NotFound(c)
		return
	}
	response.Success(c, discount)
}

func CreateDiscount(c *gin.Context) {
	var request dto.CreateDiscountRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "Dữ liệu không hợp lệ")
		return
	}

	if request.Discount < 0 || request.Discount > 100 {
		response.BadRequest(c, "Mức giảm giá phải nằm trong khoảng từ 0 đến 100")
		return
	}

	fromDate, err := time.Parse(layout, request.FromDate)
	if err != nil {
		response.BadRequest(c, "Định dạng ngày bắt đầu không hợp lệ")
		return
	}
	toDate, err := time.Parse(layout, request.ToDate)
	if err != nil {
		response.BadRequest(c, "Định dạng ngày kết thúc không hợp lệ")
		return
	}

	if !toDate.After(fromDate) {
		response.BadRequest(c, "Ngày kết thúc phải sau ngày bắt đầu")
		return
	}
	discount := models.Discount{
		Name:        request.Name,
		Description: request.Description,
		Quantity:    request.Quantity,
		FromDate:    request.FromDate,
		ToDate:      request.ToDate,
		Discount:    request.Discount,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := config.DB.Create(&discount).Error; err != nil {
		response.ServerError(c)
		return
	}

	response.Success(c, discount)
}

func UpdateDiscount(c *gin.Context) {
	var request dto.UpdateDiscountRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "Dữ liệu không hợp lệ")
		return
	}

	var discount models.Discount
	if err := config.DB.First(&discount, request.ID).Error; err != nil {
		response.NotFound(c)
		return
	}

	if request.Name != "" {
		discount.Name = request.Name
	}
	if request.Description != "" {
		discount.Description = request.Description
	}
	if request.Quantity > 0 {
		discount.Quantity = request.Quantity
	}
	if request.FromDate != "" {
		discount.FromDate = request.FromDate
	}
	if request.ToDate != "" {
		discount.ToDate = request.ToDate
	}
	if request.Discount > 0 {
		discount.Discount = request.Discount
	}
	discount.UpdatedAt = time.Now()

	if err := config.DB.Save(&discount).Error; err != nil {
		response.ServerError(c)
		return
	}

	//Xóa redis
	rdb, redisErr := config.ConnectRedis()
	if redisErr == nil {
		cacheKey := "benefits:all"
		_ = services.DeleteFromRedis(config.Ctx, rdb, cacheKey)
	}

	response.Success(c, discount)
}

func DeleteDiscount(c *gin.Context) {
	id := c.Param("id")
	if err := config.DB.Delete(&models.Discount{}, id).Error; err != nil {
		response.ServerError(c)
		return
	}

	//Xóa redis
	rdb, redisErr := config.ConnectRedis()
	if redisErr == nil {
		cacheKey := "benefits:all"
		_ = services.DeleteFromRedis(config.Ctx, rdb, cacheKey)
	}

	response.Success(c, nil)
}

func ChangeDiscountStatus(c *gin.Context) {
	var request dto.ChangeDiscountStatusRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "Dữ liệu không hợp lệ")
		return
	}

	var discount models.Discount
	if err := config.DB.First(&discount, request.ID).Error; err != nil {
		response.NotFound(c)
		return
	}

	discount.Status = request.Status

	if err := discount.ValidateStatusDiscount(); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := config.DB.Model(&discount).Update("status", request.Status).Error; err != nil {
		response.ServerError(c)
		return
	}

	discount.Status = request.Status

	//Xóa redis
	rdb, redisErr := config.ConnectRedis()
	if redisErr == nil {
		cacheKey := "benefits:all"
		_ = services.DeleteFromRedis(config.Ctx, rdb, cacheKey)
	}

	response.Success(c, discount)
}
