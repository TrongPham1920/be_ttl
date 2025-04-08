package controllers

import (
	"log"
	"net/http"
	"new/services"

	"github.com/gin-gonic/gin"
)

func ChatSearchHandler(c *gin.Context) {
	var request struct {
		Message string `json:"message"`
	}
	if err := c.ShouldBindJSON(&request); err != nil || request.Message == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Message không hợp lệ"})
		return
	}

	// Gọi GPT để trích xuất thông tin
	params, err := services.GetHotelSearchParamsFromUserMessage(request.Message)
	if err != nil {
		log.Println("Lỗi GPT:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không trích xuất được thông tin từ GPT"})
		return
	}

	// Tìm kiếm trong ElasticSearch
	accommodations, _, err := services.SearchAccommodationsWithFilters(params)
	if err != nil {
		log.Println("Lỗi tìm kiếm Elastic:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không tìm được kết quả phù hợp"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"results": accommodations,
	})
}
