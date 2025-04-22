package services

import (
	"fmt"
	"log"
	"new/config"
	"new/models"
	"new/response"
	"time"

	"github.com/gin-gonic/gin"
)

func GetAllAccommodationStatuses(c *gin.Context, fromDate, toDate time.Time) ([]models.AccommodationStatus, error) {
	var statuses []models.AccommodationStatus

	// Tạo cache key
	cacheKey := "accommodations:statuses"

	// Kết nối Redis
	rdb, err := config.ConnectRedis()
	if err != nil {
		response.ServerError(c)
		return nil, fmt.Errorf("không thể kết nối Redis: %v", err)
	}

	// Thử lấy dữ liệu từ Redis
	if err := GetFromRedis(config.Ctx, rdb, cacheKey, &statuses); err == nil && len(statuses) > 0 {
		return filterAccommodationStatusesByDate(statuses, fromDate, toDate), nil
	}

	// Nếu không có trong Redis, truy vấn từ cơ sở dữ liệu
	today := time.Now().Truncate(24 * time.Hour)
	err = config.DB.Where("status != 0 AND to_date >= ?", today).Find(&statuses).Error
	if err != nil {
		return nil, fmt.Errorf("không thể lấy dữ liệu từ cơ sở dữ liệu: %v", err)
	}

	// Lưu dữ liệu vào Redis
	if err := SetToRedis(config.Ctx, rdb, cacheKey, statuses, 60*time.Minute); err != nil {
		log.Printf("Lỗi khi lưu dữ liệu vào Redis: %v", err)
	}

	return filterAccommodationStatusesByDate(statuses, fromDate, toDate), nil
}

// Hàm lọc danh sách phòng theo khoảng thời gian
func filterAccommodationStatusesByDate(statuses []models.AccommodationStatus, fromDate, toDate time.Time) []models.AccommodationStatus {
	var filteredStatuses []models.AccommodationStatus
	fromDate = fromDate.Truncate(24 * time.Hour)
	toDate = toDate.Truncate(24 * time.Hour)

	for _, status := range statuses {
		// Chuẩn hóa thời gian để tránh sai lệch múi giờ
		statusFromDate := status.FromDate.Truncate(24 * time.Hour)
		statusToDate := status.ToDate.Truncate(24 * time.Hour)

		// Nếu có giao nhau với khoảng tìm kiếm thì loại bỏ
		if !(toDate.Before(statusFromDate) || fromDate.After(statusToDate)) {
			filteredStatuses = append(filteredStatuses, status)
		}

	}
	log.Println("filtered Status", filteredStatuses)
	return filteredStatuses
}
