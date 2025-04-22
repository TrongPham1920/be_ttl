package services

import (
	"context"
	"encoding/json"
	"log"
	"strconv"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func GetCacheKey(userID int, sessionID string) string {
	if userID > 0 {
		return strconv.Itoa(userID)
	}
	return sessionID
}

func HandleUserMessageWS(
	ctx context.Context,
	rdb *redis.Client,
	es *elasticsearch.Client,
	redisKey string,
	userID int,
	userInput string,
	c *gin.Context,
) [][]byte {
	var responses [][]byte

	if userInput == "reset" {
		if err := ClearLastFilters(ctx, rdb, redisKey); err != nil {
			log.Println("ClearLastFilters:", err)
		}
		responses = append(responses, []byte("Đã reset bộ lọc tìm kiếm."))
		return responses
	}

	// Gọi GPT
	filters, gptResponse, err := ExtractSearchFiltersFromGPTWS(userInput)
	if err != nil {
		responses = append(responses, []byte("Lỗi khi phân tích yêu cầu."))
		return responses
	}

	if gptResponse != "" {
		responses = append(responses, []byte(gptResponse))
	}

	if filters == nil {
		return responses
	}

	// Gộp bộ lọc cũ
	prevFilters, _ := GetLastFilters(ctx, rdb, redisKey)
	if prevFilters != nil {
		filters = MergeFilters(prevFilters, filters)
	}

	// Lọc chỗ ở đã được đặt (nếu có ngày)
	var excludeIDs []uint
	if filters.FromDate != nil && filters.ToDate != nil {
		statuses, err := GetAllAccommodationStatuses(c, *filters.FromDate, *filters.ToDate)
		if err == nil {
			for _, status := range statuses {
				excludeIDs = append(excludeIDs, status.AccommodationID)
			}
		}
	}

	// Lưu bộ lọc mới
	_ = SaveLastFilters(ctx, rdb, redisKey, filters)

	// ElasticSearch
	query := BuildESQueryFromFilters(filters, excludeIDs)
	results, _, err := SearchElastic(es, query, "accommodations")
	if err != nil {
		responses = append(responses, []byte("Lỗi tìm kiếm: "+err.Error()))
		return responses
	}

	if len(results) > 3 {
		results = results[:3]
	}

	hotelJSON, err := json.Marshal(results)
	if err != nil {
		responses = append(responses, []byte("⚠️ Có lỗi khi gửi kết quả khách sạn."))
	} else {
		responses = append(responses, hotelJSON)
	}

	return responses
}
