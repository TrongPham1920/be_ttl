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

// GetHolidays lấy tất cả kỳ nghỉ
func GetHolidays(c *gin.Context) {
	var holidays []models.Holiday

	if err := config.DB.Find(&holidays).Error; err != nil {
		response.ServerError(c)
		return
	}

	pageStr := c.Query("page")
	limitStr := c.Query("limit")
	nameFilter := c.Query("name")
	priceStr := c.Query("price")
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
	var holidayResponses []dto.HolidayResponse
	tx := config.DB.Model(&models.Holiday{})
	if nameFilter != "" {
		decodedNameFilter, err := url.QueryUnescape(nameFilter)
		if err != nil {
			response.ServerError(c)
			return
		}
		tx = tx.Where("name ILIKE ?", "%"+decodedNameFilter+"%")
	}
	if priceStr != "" {
		price, err := strconv.ParseFloat(priceStr, 64)
		if err == nil {
			tx = tx.Where("price = ?", int(price))
		}
	}
	if fromDateStr != "" {
		fromDateComparable, err := time.Parse("02/01/2006", fromDateStr)
		if err != nil {
			response.BadRequest(c, "Sai định dạng fromDate")
			return
		}

		if toDateStr != "" {
			toDateComparable, err := time.Parse("02/01/2006", toDateStr)
			if err != nil {
				response.BadRequest(c, "Sai định dạng toDate")
				return
			}
			tx = tx.Where("SUBSTRING(from_date, 7, 4) || SUBSTRING(from_date, 4, 2) || SUBSTRING(from_date, 1, 2) >= ? AND SUBSTRING(to_date, 7, 4) || SUBSTRING(to_date, 4, 2) || SUBSTRING(to_date, 1, 2) <= ?", fromDateComparable.Format("20060102"), toDateComparable.Format("20060102"))
		} else {
			tx = tx.Where("SUBSTRING(from_date, 7, 4) || SUBSTRING(from_date, 4, 2) || SUBSTRING(from_date, 1, 2) >= ?", fromDateComparable.Format("20060102"))
		}
	}
	var totalHolidays int64
	if err := tx.Count(&totalHolidays).Error; err != nil {
		response.ServerError(c)
		return
	}
	tx = tx.Order("updated_at desc")

	if err := tx.Offset(page * limit).Limit(limit).Find(&holidays).Error; err != nil {
		response.ServerError(c)
		return
	}

	for _, holiday := range holidays {
		holidayResponses = append(holidayResponses, dto.HolidayResponse{
			ID:        holiday.ID,
			Name:      holiday.Name,
			FromDate:  holiday.FromDate,
			ToDate:    holiday.ToDate,
			Price:     holiday.Price,
			CreatedAt: holiday.CreatedAt,
			UpdatedAt: holiday.UpdatedAt,
		})
	}

	response.SuccessWithPagination(c, holidayResponses, page, limit, int(totalHolidays))
}

// CreateHoliday tạo một kỳ nghỉ mới
func CreateHoliday(c *gin.Context) {
	var request dto.CreateHolidayRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "Dữ liệu không hợp lệ")
		return
	}
	fromDate, err := time.Parse("02/01/2006", request.FromDate)
	if err != nil {
		response.BadRequest(c, "Định dạng ngày bắt đầu không hợp lệ")
		return
	}
	toDate, err := time.Parse("02/01/2006", request.ToDate)
	if err != nil {
		response.BadRequest(c, "Định dạng ngày kết thúc không hợp lệ")
		return
	}

	if toDate.Before(fromDate) {
		response.BadRequest(c, "Ngày kết thúc phải sau ngày bắt đầu")
		return
	}
	holiday := models.Holiday{
		Name:      request.Name,
		FromDate:  request.FromDate,
		ToDate:    request.ToDate,
		Price:     request.Price,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := config.DB.Create(&holiday).Error; err != nil {
		response.ServerError(c)
		return
	}

	//Xóa redis
	rdb, redisErr := config.ConnectRedis()
	if redisErr == nil {
		cacheKey := "holidays:all"
		_ = services.DeleteFromRedis(config.Ctx, rdb, cacheKey)
	}

	response.Success(c, holiday)
}

func GetDetailHoliday(c *gin.Context) {
	var holiday models.Holiday
	if err := config.DB.Where("id = ?", c.Param("id")).First(&holiday).Error; err != nil {
		response.NotFound(c)
		return
	}

	response.Success(c, holiday)
}

// UpdateHoliday cập nhật một kỳ nghỉ
func UpdateHoliday(c *gin.Context) {
	var holiday models.Holiday
	var request dto.UpdateHolidayRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "Dữ liệu không hợp lệ")
		return
	}
	if err := config.DB.First(&holiday, c.Param("id")).Error; err != nil {
		response.NotFound(c)
		return
	}

	holiday.Name = request.Name
	holiday.FromDate = request.FromDate
	holiday.ToDate = request.ToDate
	holiday.Price = request.Price
	holiday.UpdatedAt = time.Now()

	if err := config.DB.Save(&holiday).Error; err != nil {
		response.ServerError(c)
		return
	}

	//Xóa redis
	rdb, redisErr := config.ConnectRedis()
	if redisErr == nil {
		cacheKey := "holidays:all"
		_ = services.DeleteFromRedis(config.Ctx, rdb, cacheKey)
	}

	response.Success(c, holiday)
}

func DeleteHoliday(c *gin.Context) {
	var request dto.DeleteHolidayRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "Dữ liệu không hợp lệ")
		return
	}
	if len(request.IDs) == 0 {
		response.BadRequest(c, "Không có ID nào được cung cấp")
		return
	}

	if err := config.DB.Delete(&models.Holiday{}, request.IDs).Error; err != nil {
		response.ServerError(c)
		return
	}

	//Xóa redis
	rdb, redisErr := config.ConnectRedis()
	if redisErr == nil {
		cacheKey := "holidays:all"
		_ = services.DeleteFromRedis(config.Ctx, rdb, cacheKey)
	}

	response.Success(c, nil)
}
