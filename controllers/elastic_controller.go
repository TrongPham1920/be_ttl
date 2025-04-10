package controllers

import (
	"new/response"
	"new/services"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Hàm tìm kiếm trong bán kính
func SearchNear(c *gin.Context) {
	lat, _ := strconv.ParseFloat(c.Query("lat"), 64)
	lon, _ := strconv.ParseFloat(c.Query("lon"), 64)
	radius := c.DefaultQuery("radius", "5km")

	results, err := services.NearbyAccommodations(lat, lon, radius)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	response.Success(c, results)
}
