package controllers

import (
	"encoding/json"
	"fmt"
	"log"
	"new/config"
	"new/dto"
	"new/models"
	"new/response"
	"new/services"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func GetAllRates(c *gin.Context) {
	accommodationIdFilter := c.DefaultQuery("accommodationId", "")

	cacheKey := "rates:all"
	if accommodationIdFilter != "" {
		cacheKey = fmt.Sprintf("rates:accommodation:%s", accommodationIdFilter)
	}

	// Kết nối Redis
	rdb, err := config.ConnectRedis()
	if err != nil {
		response.ServerError(c)
		return
	}

	var rates []models.Rate

	// Lấy dữ liệu từ Redis
	err = services.GetFromRedis(config.Ctx, rdb, cacheKey, &rates)
	if err == nil && len(rates) > 0 {
		var rateResponses []dto.RateResponse
		for _, rate := range rates {
			rateResponse := dto.RateResponse{
				ID:              rate.ID,
				AccommodationID: rate.AccommodationID,
				Comment:         rate.Comment,
				Star:            rate.Star,
				CreatedAt:       rate.CreateAt,
				UpdatedAt:       rate.UpdateAt,
				User: dto.UserInfo{
					ID:     rate.User.ID,
					Name:   rate.User.Name,
					Avatar: rate.User.Avatar,
				},
			}
			rateResponses = append(rateResponses, rateResponse)
		}
		response.Success(c, rateResponses)
		return
	}

	// Lấy dữ liệu từ database
	tx := config.DB.Preload("User")
	if accommodationIdFilter != "" {
		if parsedAccommodationId, err := strconv.Atoi(accommodationIdFilter); err == nil {
			tx = tx.Where("accommodation_id = ?", parsedAccommodationId)
		}
	}

	if err := tx.Limit(20).Find(&rates).Error; err != nil {
		response.ServerError(c)
		return
	}

	var rateResponses []dto.RateResponse
	for _, rate := range rates {
		rateResponse := dto.RateResponse{
			ID:              rate.ID,
			AccommodationID: rate.AccommodationID,
			Comment:         rate.Comment,
			Star:            rate.Star,
			CreatedAt:       rate.CreateAt,
			UpdatedAt:       rate.UpdateAt,
			User: dto.UserInfo{
				ID:     rate.User.ID,
				Name:   rate.User.Name,
				Avatar: rate.User.Avatar,
			},
		}
		rateResponses = append(rateResponses, rateResponse)
	}

	rateResponsesJSON, err := json.Marshal(rateResponses)
	if err != nil {
		response.ServerError(c)
		return
	}

	if err := services.SetToRedis(config.Ctx, rdb, cacheKey, rateResponsesJSON, 10*time.Minute); err != nil {
		log.Printf("Lỗi khi lưu danh sách đánh giá vào Redis: %v", err)
	}

	response.Success(c, rateResponses)
}

func CreateRate(c *gin.Context) {
	var rate models.Rate
	if err := c.ShouldBindJSON(&rate); err != nil {
		response.BadRequest(c, "Dữ liệu không hợp lệ")
		return
	}

	var existingRate models.Rate
	if err := config.DB.Where("user_id = ? AND accommodation_id = ?", rate.UserID, rate.AccommodationID).First(&existingRate).Error; err == nil {
		response.Error(c, 0, "Bạn đã đánh giá lưu trú này trước đó")
		return
	}

	if err := config.DB.Create(&rate).Error; err != nil {
		response.ServerError(c)
		return
	}

	if err := services.UpdateAccommodationRating(rate.AccommodationID); err != nil {
		response.ServerError(c)
		return
	}

	//Xóa redis
	rdb, redisErr := config.ConnectRedis()
	if redisErr == nil {
		cacheKey := "rates:all"
		_ = services.DeleteFromRedis(config.Ctx, rdb, cacheKey)
		cacheKey2 := "accommodations:all"
		_ = services.DeleteFromRedis(config.Ctx, rdb, cacheKey2)
	}

	response.Success(c, rate)
}

func GetRateDetail(c *gin.Context) {
	id := c.Param("id")
	var rate models.Rate
	if err := config.DB.Preload("User").First(&rate, id).Error; err != nil {
		response.NotFound(c)
		return
	}

	rateResponse := dto.RateResponse{
		ID:              rate.ID,
		AccommodationID: rate.AccommodationID,
		Comment:         rate.Comment,
		Star:            rate.Star,
		CreatedAt:       rate.CreateAt,
		UpdatedAt:       rate.UpdateAt,
		User: dto.UserInfo{
			ID:     rate.User.ID,
			Name:   rate.User.Name,
			Avatar: rate.User.Avatar,
		},
	}

	response.Success(c, rateResponse)
}

func UpdateRate(c *gin.Context) {
	var rateInput struct {
		ID      uint   `json:"id"`
		Comment string `json:"comment"`
		Star    int    `json:"star"`
	}

	if err := c.ShouldBindJSON(&rateInput); err != nil {
		response.BadRequest(c, "Dữ liệu không hợp lệ")
		return
	}

	var rate models.Rate
	if err := config.DB.First(&rate, rateInput.ID).Error; err != nil {
		response.NotFound(c)
		return
	}

	rate.Comment = rateInput.Comment
	rate.Star = rateInput.Star

	if err := config.DB.Save(&rate).Error; err != nil {
		response.ServerError(c)
		return
	}

	if err := services.UpdateAccommodationRating(rate.AccommodationID); err != nil {
		response.ServerError(c)
		return
	}

	rateResponse := dto.RateUpdateResponse{
		ID:              rate.ID,
		AccommodationID: rate.AccommodationID,
		Comment:         rate.Comment,
		Star:            rate.Star,
		CreatedAt:       rate.CreateAt,
		UpdatedAt:       rate.UpdateAt,
	}

	//Xóa redis
	rdb, redisErr := config.ConnectRedis()
	if redisErr == nil {
		cacheKey := "rates:all"
		_ = services.DeleteFromRedis(config.Ctx, rdb, cacheKey)
		cacheKey2 := "accommodations:all"
		_ = services.DeleteFromRedis(config.Ctx, rdb, cacheKey2)
	}

	response.Success(c, rateResponse)
}
