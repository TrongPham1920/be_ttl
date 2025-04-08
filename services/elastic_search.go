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
	"new/models"
	"strconv"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esutil"
)

var es *elasticsearch.Client

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
	es, err = elasticsearch.NewClient(cfg)
	if err != nil {
		log.Fatal("❌ Không thể kết nối Elasticsearch:", err)
	}

	log.Println("🟢 Kết nối Elasticsearch thành công!")
}

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
		meta := fmt.Sprintf(`{ "index" : { "_index" : "accommodations", "_id" : "%s" } }`, id)
		buf.WriteString(meta + "\n")

		// Chuyển acc thành JSON
		hotelJSON, err := json.Marshal(acc)
		if err != nil {
			log.Printf("❌ Lỗi khi convert accommodation thành JSON: %v\n", err)
			continue
		}
		buf.WriteString(string(hotelJSON) + "\n")
	}

	return sendBulkRequest(buf.String())
}

// Gửi request bulk đến Elasticsearch
func sendBulkRequest(data string) error {
	res, err := es.Bulk(bytes.NewReader([]byte(data)), es.Bulk.WithContext(context.Background()))
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
	res, err := es.Indices.Delete([]string{indexName}, es.Indices.Delete.WithContext(context.Background()))
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
func SearchAccommodations(query string) ([]models.Accommodation, error) {
	if es == nil {
		return nil, fmt.Errorf("ElasticSearch client chưa được khởi tạo")
	}
	// Tạo truy vấn Elasticsearch
	searchQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"should": []map[string]interface{}{
					{"multi_match": map[string]interface{}{
						"query":     query,
						"fields":    []string{"name^3", "address^2", "province", "district", "ward", "shortDescription", "description", "benefits_summary"},
						"fuzziness": "AUTO", // Cho phép tìm kiếm gần đúng
					}},
					{"match_phrase_prefix": map[string]interface{}{
						"name": query, // Hỗ trợ gợi ý tìm kiếm
					}},
				},
				"filter": []map[string]interface{}{
					{"terms": map[string]interface{}{
						"type": []int{0, 1, 2}, // 0: Hotel, 1: Homestay, 2: Resort (giả định)
					}},
				},
				"minimum_should_match": 1,
			},
		},
		"sort": []map[string]interface{}{
			{"_score": "desc"},
		},
	}

	// Chuyển thành JSON
	queryBody, _ := json.Marshal(searchQuery)

	// Gửi request đến Elasticsearch
	res, err := es.Search(
		es.Search.WithContext(context.Background()),
		es.Search.WithIndex("accommodations"),
		es.Search.WithBody(bytes.NewReader(queryBody)),
		es.Search.WithPretty(),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	// Xử lý kết quả trả về
	var result struct {
		Hits struct {
			Hits []struct {
				Source models.Accommodation `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, err
	}

	// Lưu danh sách kết quả
	var accommodations []models.Accommodation
	for _, hit := range result.Hits.Hits {
		accommodations = append(accommodations, hit.Source)
	}

	return accommodations, nil
}

func GetUnavailableAccommodationIDs(fromDate, toDate string) ([]uint, error) {
	var ids []uint
	err := config.DB.
		Table("accommodation_statuses").
		Select("accommodation_id").
		Where("from_date <= ? AND to_date >= ?", toDate, fromDate).
		Group("accommodation_id").
		Pluck("accommodation_id", &ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}

func SearchAccommodationsWithFilters(params map[string]string) ([]models.Accommodation, int, error) {
	filters := BuildFilters(params)

	// Check fromDate & toDate để loại bỏ các accommodation đã được đặt
	if fromDate, ok := params["fromDate"]; ok && fromDate != "" {
		if toDate, ok2 := params["toDate"]; ok2 && toDate != "" {
			unavailableIDs, err := GetUnavailableAccommodationIDs(fromDate, toDate)
			if err == nil && len(unavailableIDs) > 0 {
				filters = append(filters, map[string]interface{}{
					"bool": map[string]interface{}{
						"must_not": map[string]interface{}{
							"terms": map[string]interface{}{
								"id": unavailableIDs,
							},
						},
					},
				})
			}
		}
	}

	boolQuery := BuildBoolQuery(params["search"], filters)
	queryBody := BuildESQueryBody(boolQuery, params)
	return ExecuteESQuery(queryBody)
}

func BuildFilters(params map[string]string) []map[string]interface{} {
	filters := []map[string]interface{}{}

	if v := params["type"]; v != "" {
		filters = append(filters, term("type", v))
	}
	if v := params["province"]; v != "" {
		filters = append(filters, term("province", v))
	}
	if v := params["district"]; v != "" {
		filters = append(filters, term("district", v))
	}
	if v := params["status"]; v != "" {
		filters = append(filters, term("status", v))
	}
	if v := params["numBed"]; v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			filters = append(filters, rangeGTE("numBed", val))
		}
	}
	if v := params["numTolet"]; v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			filters = append(filters, rangeGTE("numTolet", val))
		}
	}
	if v := params["people"]; v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			filters = append(filters, rangeGTE("people", val))
		}
	}
	if v := params["benefitId"]; v != "" {
		benefitIDs := strings.Split(v, ",")
		filters = append(filters, map[string]interface{}{
			"terms": map[string]interface{}{"benefitIds": benefitIDs},
		})
	}

	return filters
}

// Build bool query with should + filter
func BuildBoolQuery(search string, filters []map[string]interface{}) map[string]interface{} {
	shouldQuery := []map[string]interface{}{}
	if search != "" {
		shouldQuery = append(shouldQuery,
			map[string]interface{}{
				"multi_match": map[string]interface{}{
					"query":     search,
					"fields":    []string{"name^3", "address^2", "province", "district", "ward", "shortDescription", "description", "benefits_summary"},
					"fuzziness": "AUTO",
				},
			},
			map[string]interface{}{
				"match_phrase_prefix": map[string]interface{}{
					"name": search,
				},
			},
		)
	}

	boolQuery := map[string]interface{}{
		"should":               shouldQuery,
		"filter":               filters,
		"minimum_should_match": 1,
	}

	return map[string]interface{}{"bool": boolQuery}
}

// Build full ES query body
func BuildESQueryBody(query map[string]interface{}, params map[string]string) map[string]interface{} {
	page, _ := strconv.Atoi(params["page"])
	limit, _ := strconv.Atoi(params["limit"])
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	offset := (page - 1) * limit

	return map[string]interface{}{
		"from":  offset,
		"size":  limit,
		"query": query,
		"sort": []map[string]interface{}{
			{"_score": "desc"},
		},
	}
}

// Execute ES search
func ExecuteESQuery(query map[string]interface{}) ([]models.Accommodation, int, error) {
	var results struct {
		Hits struct {
			Total struct {
				Value int `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source models.Accommodation `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	res, err := es.Search(
		es.Search.WithContext(context.Background()),
		es.Search.WithIndex("accommodations"),
		es.Search.WithBody(esutil.NewJSONReader(query)),
		es.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&results); err != nil {
		return nil, 0, err
	}

	accommodations := make([]models.Accommodation, 0)
	for _, hit := range results.Hits.Hits {
		accommodations = append(accommodations, hit.Source)
	}

	return accommodations, results.Hits.Total.Value, nil
}

// Helper: term
func term(field string, value interface{}) map[string]interface{} {
	return map[string]interface{}{
		"term": map[string]interface{}{field: value},
	}
}

// Helper: range gte
func rangeGTE(field string, value int) map[string]interface{} {
	return map[string]interface{}{
		"range": map[string]interface{}{
			field: map[string]interface{}{"gte": value},
		},
	}
}

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

	res, err := es.Search(
		es.Search.WithContext(context.Background()),
		es.Search.WithIndex("accommodations"),
		es.Search.WithBody(&buf),
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

	res, err := es.Search(
		es.Search.WithContext(context.Background()),
		es.Search.WithIndex("accommodations"),
		es.Search.WithBody(&buf),
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
