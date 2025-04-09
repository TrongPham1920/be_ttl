package controllers

import (
	"net/http"
	"new/services"

	"github.com/gin-gonic/gin"
)

func ChatSearchHandler(c *gin.Context) {
	var req struct {
		Message string `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Message == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message không hợp lệ"})
		return
	}

	// Gọi GPT để phân tích câu hỏi
	filters, err := services.ExtractSearchFiltersFromGPT(req.Message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không phân tích được câu hỏi"})
		return
	}

	// Lọc các chỗ đã đặt nếu có from/to date
	excludeIDs := []uint{}
	if filters.FromDate != nil && filters.ToDate != nil {
		statuses, err := getAllAccommodationStatuses(c, *filters.FromDate, *filters.ToDate)
		if err == nil {
			for _, status := range statuses {
				excludeIDs = append(excludeIDs, status.AccommodationID)
			}
		}
	}

	// Build truy vấn & tìm kiếm Elastic
	query := services.BuildESQueryFromFilters(filters, excludeIDs)
	results, total, err := services.SearchElastic(services.Es, query, "accommodations")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tìm kiếm: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"total":   total,
	})
}
