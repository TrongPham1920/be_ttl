package services

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"new/config"
	"new/dto"
	"new/models"
	"strconv"
	"time"

	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gin-gonic/gin"
)

var Es *elasticsearch.Client

// Kết nối Elastic
func ConnectElastic() {
	cfg := elasticsearch.Config{
		Addresses: []string{
			"http://14.225.212.252:9200",
		},
		Username: "elastic",
		Password: "123456",
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	var err error
	Es, err = elasticsearch.NewClient(cfg)
	if err != nil {
		log.Fatal("❌ Không thể kết nối Elasticsearch:", err)
	}

	log.Println("🟢 Kết nối Elasticsearch thành công!")
}

// Hàm tạo data để index vào Elastic
func GetAllAccommodationsForIndexing() ([]map[string]interface{}, error) {
	var accommodations []models.Accommodation

	err := config.DB.Preload("Benefits").Preload("Rooms").Preload("User").Preload("AccommodationStatuses").Find(&accommodations).Error
	if err != nil {
		return nil, err
	}

	var formattedAccommodations []map[string]interface{}

	for _, acc := range accommodations {
		// Danh sách lợi ích theo format [{id, name}]
		var benefits []map[string]interface{}
		for _, b := range acc.Benefits {
			benefits = append(benefits, map[string]interface{}{
				"id":   b.Id,
				"name": b.Name,
			})
		}

		// User object
		user := map[string]interface{}{
			"id":          acc.User.ID,
			"name":        acc.User.Name,
			"email":       acc.User.Email,
			"phoneNumber": acc.User.PhoneNumber,
		}

		accData := map[string]interface{}{
			"id":               acc.ID,
			"type":             acc.Type,
			"province":         acc.Province,
			"name":             acc.Name,
			"address":          acc.Address,
			"avatar":           acc.Avatar,
			"shortDescription": acc.ShortDescription,
			"description":      acc.Description,
			"status":           acc.Status,
			"num":              acc.Num,
			"people":           acc.People,
			"price":            acc.Price,
			"numBed":           acc.NumBed,
			"numTolet":         acc.NumTolet,
			"district":         acc.District,
			"ward":             acc.Ward,
			"longitude":        acc.Longitude,
			"latitude":         acc.Latitude,
			"benefits":         benefits,
			"user":             user,
		}

		formattedAccommodations = append(formattedAccommodations, accData)
	}

	return formattedAccommodations, nil
}

// Hàm xử lý Index vào Elastic
func IndexHotelsToES() error {
	accommodations, err := GetAllAccommodationsForIndexing()
	if err != nil {
		return err
	}

	var buf strings.Builder
	for _, acc := range accommodations {
		// Ép kiểu id an toàn
		id := fmt.Sprintf("%v", acc["id"])

		// Ghi metadata Bulk
		meta := fmt.Sprintf(`{ "index" : { "_index" : "accommodations_test", "_id" : "%s" } }`, id)
		buf.WriteString(meta + "\n")

		// Chuyển acc thành JSON
		hotelJSON, err := json.Marshal(acc)
		if err != nil {
			log.Printf(" Lỗi khi convert accommodation thành JSON: %v\n", err)
			continue
		}
		buf.WriteString(string(hotelJSON) + "\n")
	}

	return sendBulkRequest(buf.String())
}

// Hàm hỗ trợ cho IndexHotelsToES gửi request theo bulk đến Elasticsearch (tăng tốc nếu data lớn)
func sendBulkRequest(data string) error {
	res, err := Es.Bulk(bytes.NewReader([]byte(data)), Es.Bulk.WithContext(context.Background()))
	if err != nil {
		return fmt.Errorf("❌ Lỗi khi gửi Bulk API: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println("❌ Không thể đọc body từ phản hồi:", err)
		return err
	}
	if len(body) == 0 {
		fmt.Println("⚠️ Body trả về rỗng!") // ✅ Check 3
	} else {
		fmt.Println("📨 Phản hồi từ Elasticsearch:", string(body))
	}

	// Log phản hồi dạng raw
	fmt.Println("📨 Phản hồi từ Elasticsearch:")
	fmt.Println(string(body))

	// Parse và log lỗi từng item nếu có
	var bulkRes map[string]interface{}
	if err := json.Unmarshal(body, &bulkRes); err != nil {
		return fmt.Errorf("❌ Lỗi khi parse phản hồi: %w", err)
	}

	if items, ok := bulkRes["items"].([]interface{}); ok {
		for _, item := range items {
			indexOp := item.(map[string]interface{})["index"].(map[string]interface{})
			if errorInfo, exists := indexOp["error"]; exists {
				fmt.Printf("❌ Lỗi khi index document ID %v: %+v\n", indexOp["_id"], errorInfo)
			}
		}
	}

	if res.IsError() {
		return fmt.Errorf("❌ Elasticsearch trả về lỗi tổng thể: %s", string(body))
	}

	log.Println("✅ Dữ liệu đã được index thành công vào Elasticsearch!")
	return nil
}

// Xóa index trong Elasticsearch
func DeleteIndex(indexName string) error {
	res, err := Es.Indices.Delete([]string{indexName}, Es.Indices.Delete.WithContext(context.Background()))
	if err != nil {
		return fmt.Errorf("❌ Lỗi khi xóa index %s: %w", indexName, err)
	}
	defer res.Body.Close()

	// Kiểm tra phản hồi từ Elasticsearch
	if res.IsError() {
		return fmt.Errorf("⚠️ Elasticsearch trả về lỗi khi xóa index %s: %s", indexName, res.Status())
	}

	log.Printf("✅ Index '%s' đã được xóa thành công!", indexName)
	return nil
}

// Hàm cho chức năng AutoComplete
func AutocompleteAccommodations(keyword string) ([]map[string]interface{}, error) {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"should": []map[string]interface{}{
					{"match_phrase_prefix": map[string]interface{}{"name": map[string]interface{}{"query": keyword}}},
					{"match_phrase_prefix": map[string]interface{}{"address": map[string]interface{}{"query": keyword}}},
				},
			},
		},
		"size": 5,
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, err
	}

	res, err := Es.Search(
		Es.Search.WithContext(context.Background()),
		Es.Search.WithIndex("accommodations"),
		Es.Search.WithBody(&buf),
		Es.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var r map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return nil, err
	}

	results := []map[string]interface{}{}
	for _, hit := range r["hits"].(map[string]interface{})["hits"].([]interface{}) {
		results = append(results, hit.(map[string]interface{})["_source"].(map[string]interface{}))
	}
	return results, nil
}

// Hàm tìm kiếm trong bán kính 5km
func NearbyAccommodations(lat, lon float64, radius string) ([]map[string]interface{}, error) {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": map[string]interface{}{
					"geo_distance": map[string]interface{}{
						"distance": radius,
						"location": map[string]float64{
							"lat": lat,
							"lon": lon,
						},
					},
				},
			},
		},
		"sort": []map[string]interface{}{
			{
				"_geo_distance": map[string]interface{}{
					"location": map[string]float64{"lat": lat, "lon": lon},
					"order":    "asc",
					"unit":     "km",
				},
			},
		},
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, err
	}

	res, err := Es.Search(
		Es.Search.WithContext(context.Background()),
		Es.Search.WithIndex("accommodations"),
		Es.Search.WithBody(&buf),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var r map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return nil, err
	}

	results := []map[string]interface{}{}
	for _, hit := range r["hits"].(map[string]interface{})["hits"].([]interface{}) {
		results = append(results, hit.(map[string]interface{})["_source"].(map[string]interface{}))
	}
	return results, nil
}

// Hàm Parse dữ liệu phục vụ tìm kiếm trong elastic
func ParseSearchFilters(c *gin.Context) (*dto.SearchFilters, error) {
	parseIntPtr := func(str string) *int {
		if str == "" {
			return nil
		}
		if val, err := strconv.Atoi(str); err == nil {
			return &val
		}
		return nil
	}

	fromDateStr := c.Query("fromDate")
	toDateStr := c.Query("toDate")
	var fromDate, toDate *time.Time
	if fromDateStr != "" && toDateStr != "" {
		layout := "02/01/2006"
		fd, err1 := time.Parse(layout, fromDateStr)
		td, err2 := time.Parse(layout, toDateStr)
		if err1 == nil && err2 == nil {
			fromDate, toDate = &fd, &td
		}
	}

	// Parse benefits
	benefitIDs := []int{}
	raw := c.Query("benefitId")
	raw = strings.Trim(raw, "[]")
	for _, s := range strings.Split(raw, ",") {
		if id, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
			benefitIDs = append(benefitIDs, id)
		}
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if page < 1 {
		page = 1
	}

	return &dto.SearchFilters{
		Name:       c.Query("name"),
		Province:   c.Query("province"),
		District:   c.Query("district"),
		Ward:       c.Query("ward"),
		Type:       parseIntPtr(c.Query("type")),
		Status:     parseIntPtr(c.Query("status")),
		People:     parseIntPtr(c.Query("people")),
		NumBed:     parseIntPtr(c.Query("numBed")),
		NumTolet:   parseIntPtr(c.Query("numTolet")),
		PriceMin:   parseIntPtr(c.Query("priceMin")),
		PriceMax:   parseIntPtr(c.Query("priceMax")),
		BenefitIDs: benefitIDs,
		FromDate:   fromDate,
		ToDate:     toDate,
		Page:       page,
		Limit:      limit,
	}, nil
}

// Hàm xây đựng Query để truy vấn vào Elastic Search
func BuildESQueryFromFilters(filters *dto.SearchFilters, excludeIDs []uint) map[string]interface{} {
	must := []map[string]interface{}{}

	if filters.Name != "" {
		must = append(must, map[string]interface{}{
			"match": map[string]interface{}{
				"name.raw": map[string]interface{}{
					"query":     filters.Name,
					"fuzziness": "AUTO",
				},
			},
		})
	}
	if filters.Province != "" {
		must = append(must, map[string]interface{}{
			"match": map[string]interface{}{
				"province": filters.Province,
			},
		})
	}
	if filters.District != "" {
		must = append(must, map[string]interface{}{
			"match": map[string]interface{}{
				"district": filters.District,
			},
		})
	}
	if filters.Ward != "" {
		must = append(must, map[string]interface{}{
			"match": map[string]interface{}{
				"ward": filters.Ward,
			},
		})
	}
	if filters.Type != nil {
		must = append(must, map[string]interface{}{
			"term": map[string]interface{}{
				"type": filters.Type,
			},
		})
	}
	if filters.Status != nil {
		must = append(must, map[string]interface{}{
			"term": map[string]interface{}{
				"status": *filters.Status,
			},
		})
	}
	if filters.People != nil {
		must = append(must, map[string]interface{}{
			"range": map[string]interface{}{
				"people": map[string]interface{}{"gte": *filters.People},
			},
		})
	}
	if filters.NumBed != nil {
		must = append(must, map[string]interface{}{
			"range": map[string]interface{}{
				"numBed": map[string]interface{}{"gte": *filters.NumBed},
			},
		})
	}
	if filters.NumTolet != nil {
		must = append(must, map[string]interface{}{
			"range": map[string]interface{}{
				"numTolet": map[string]interface{}{"gte": *filters.NumTolet},
			},
		})
	}
	if filters.PriceMin != nil || filters.PriceMax != nil {
		priceRange := map[string]interface{}{}
		if filters.PriceMin != nil {
			priceRange["gte"] = *filters.PriceMin
		}
		if filters.PriceMax != nil {
			priceRange["lte"] = *filters.PriceMax
		}
		must = append(must, map[string]interface{}{
			"range": map[string]interface{}{
				"price": priceRange,
			},
		})
	}
	if len(filters.BenefitIDs) > 0 {
		terms := make([]interface{}, len(filters.BenefitIDs))
		for i, id := range filters.BenefitIDs {
			terms[i] = id
		}
		must = append(must, map[string]interface{}{
			"terms": map[string]interface{}{
				"benefits.id": terms,
			},
		})
	}

	boolQuery := map[string]interface{}{"must": must}
	if len(excludeIDs) > 0 {
		boolQuery["must_not"] = []map[string]interface{}{
			{
				"terms": map[string]interface{}{
					"id": excludeIDs,
				},
			},
		}
	}

	return map[string]interface{}{
		"from": (filters.Page - 1) * filters.Limit,
		"size": filters.Limit,
		"query": map[string]interface{}{
			"bool": boolQuery,
		},
		"sort": []map[string]interface{}{
			{"id": map[string]string{"order": "desc"}},
		},
	}
}

// Hàm tìm kiếm trong Elastic
func SearchElastic(es *elasticsearch.Client, query map[string]interface{}, index string) ([]dto.AccommodationResponse, int, error) {
	var buf bytes.Buffer

	// Encode truy vấn JSON vào buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, 0, fmt.Errorf("lỗi encode query: %w", err)
	}

	// Gửi request đến ElasticSearch
	res, err := es.Search(
		es.Search.WithContext(context.Background()),
		es.Search.WithIndex(index),
		es.Search.WithBody(&buf),
		es.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("lỗi khi gọi elasticsearch: %w", err)
	}
	defer res.Body.Close()

	// Kiểm tra lỗi response
	if res.IsError() {
		return nil, 0, fmt.Errorf("lỗi response từ elasticsearch: %s", res.String())
	}

	// Decode response JSON
	var r map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return nil, 0, fmt.Errorf("lỗi decode response: %w", err)
	}

	// Lấy danh sách kết quả
	hitsRaw, ok := r["hits"].(map[string]interface{})["hits"].([]interface{})
	if !ok {
		return nil, 0, fmt.Errorf("lỗi parsing hits")
	}

	// Lấy tổng số kết quả
	total := int(r["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))

	results := make([]dto.AccommodationResponse, 0)

	// Convert từng item
	for _, hit := range hitsRaw {
		source, ok := hit.(map[string]interface{})["_source"].(map[string]interface{})
		if !ok {
			continue
		}

		// Dùng json để convert sang struct
		data, err := json.Marshal(source)
		if err != nil {
			continue
		}

		var item dto.AccommodationResponse
		if err := json.Unmarshal(data, &item); err != nil {
			continue
		}

		results = append(results, item)
	}

	return results, total, nil
}
