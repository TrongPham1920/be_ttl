package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/url"
	"new/config"
	"new/dto"
	"new/models"
	"new/response"
	"new/services"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fiam/gounidecode/unidecode"
	"github.com/schollz/closestmatch"
	"github.com/texttheater/golang-levenshtein/levenshtein"
	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
)

func getAllAccommodationStatuses(c *gin.Context, fromDate, toDate time.Time) ([]models.AccommodationStatus, error) {
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
	if err := services.GetFromRedis(config.Ctx, rdb, cacheKey, &statuses); err == nil && len(statuses) > 0 {
		return filterAccommodationStatusesByDate(statuses, fromDate, toDate), nil
	}

	// Nếu không có trong Redis, truy vấn từ cơ sở dữ liệu
	today := time.Now().Truncate(24 * time.Hour)
	err = config.DB.Where("status != 0 AND to_date >= ?", today).Find(&statuses).Error
	if err != nil {
		return nil, fmt.Errorf("không thể lấy dữ liệu từ cơ sở dữ liệu: %v", err)
	}

	// Lưu dữ liệu vào Redis
	if err := services.SetToRedis(config.Ctx, rdb, cacheKey, statuses, 60*time.Minute); err != nil {
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
	return filteredStatuses
}

func GetAccBookingDates(c *gin.Context) {
	accommodationID := c.DefaultQuery("id", "")
	date := c.DefaultQuery("date", "")

	if accommodationID == "" || date == "" {
		response.BadRequest(c, "id và date là bắt buộc")
		return
	}

	layout := "01/2006"
	parsedDate, err := time.Parse(layout, date)
	if err != nil {
		response.BadRequest(c, "Ngày không hợp lệ, vui lòng sử dụng định dạng mm/yyyy")
		return
	}

	firstDay := time.Date(parsedDate.Year(), parsedDate.Month(), 1, 0, 0, 0, 0, time.Local)
	lastDay := firstDay.AddDate(0, 1, -1)

	var allDates []time.Time
	for day := firstDay; day.Before(lastDay.AddDate(0, 0, 1)); day = day.AddDate(0, 0, 1) {
		allDates = append(allDates, day)
	}

	var statuses []models.AccommodationStatus
	db := config.DB

	err = db.Where("accommodation_id = ?", accommodationID).Find(&statuses).Error
	if err != nil {
		log.Printf("Error retrieving accommodation statuses: %v", err)
		response.ServerError(c)
		return
	}

	// Lấy thông tin khách đặt phòng (chỉ lấy guest_name và guest_phone)
	orderMap, err := getGuestBookings(accommodationID)
	if err != nil {
		log.Printf("Error retrieving guest bookings: %v", err)
		response.ServerError(c)
		return
	}

	dateFormat := "02/01/2006"
	var roomResponses []map[string]interface{}

	for _, currentDate := range allDates {
		dateStr := currentDate.Format(dateFormat)
		var statusFound bool
		var status int

		for _, accStatus := range statuses {
			if currentDate.After(accStatus.FromDate.AddDate(0, 0, -1)) && currentDate.Before(accStatus.ToDate) {
				status = accStatus.Status
				statusFound = true
				break
			}
		}

		if !statusFound {
			status = 0
		}

		roomResponse := map[string]interface{}{
			"date":   dateStr,
			"status": status,
		}

		// Nếu có khách đặt phòng, gán vào response (dạng object)
		if guest, exists := orderMap[dateStr]; exists {
			roomResponse["guest"] = guest
		}

		roomResponses = append(roomResponses, roomResponse)
	}

	response.Success(c, roomResponses)
}

// getGuestBookings lấy thông tin khách đặt phòng theo accommodation_id cho API GetAccBookingDates
func getGuestBookings(accommodationID string) (map[string]map[string]string, error) {
	var orders []models.Order
	db := config.DB
	err := db.Where("accommodation_id = ?", accommodationID).Find(&orders).Error
	if err != nil {
		return nil, fmt.Errorf("lỗi khi lấy danh sách đặt phòng: %v", err)
	}

	dateFormat := "02/01/2006"
	orderMap := make(map[string]map[string]string)

	for _, order := range orders {
		checkIn, err := time.Parse(dateFormat, order.CheckInDate)
		if err != nil {
			log.Printf("Error parsing CheckIn date: %v", err)
			continue
		}

		checkOut, err := time.Parse(dateFormat, order.CheckOutDate)
		if err != nil {
			log.Printf("Error parsing CheckOut date: %v", err)
			continue
		}

		guestName := order.GuestName
		guestPhone := order.GuestPhone

		// Nếu guestName và guestPhone rỗng, truy vấn vào bảng Users
		if guestName == "" || guestPhone == "" {
			var user models.User
			err := db.Where("id = ?", order.UserID).First(&user).Error
			if err != nil {
				log.Printf("Không tìm thấy thông tin user ID %d: %v", order.UserID, err)
				continue
			}

			// Gán thông tin từ bảng users nếu thiếu
			if guestName == "" {
				guestName = user.Name
			}
			if guestPhone == "" {
				guestPhone = user.PhoneNumber
			}
		}

		for day := checkIn; !day.After(checkOut); day = day.AddDate(0, 0, 1) {
			dateKey := day.Format(dateFormat)
			// Chỉ lưu khách đầu tiên của ngày đó (nếu chưa có)
			if _, exists := orderMap[dateKey]; !exists {
				orderMap[dateKey] = map[string]string{
					"guest_name":  guestName,
					"guest_phone": guestPhone,
				}
			}
		}
	}

	return orderMap, nil
}

func GetAllAccommodations(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 0, "mess": "Authorization header is missing"})
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	currentUserID, currentUserRole, err := GetUserIDFromToken(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 0, "mess": "Invalid token"})
		return
	}

	// Tạo cache key dựa trên vai trò và user_id
	var cacheKey string
	if currentUserRole == 2 {
		cacheKey = fmt.Sprintf("accommodations:admin:%d", currentUserID)
	} else if currentUserRole == 3 {
		cacheKey = fmt.Sprintf("accommodations:receptionist:%d", currentUserID)
	} else {
		cacheKey = "accommodations:all"
	}

	// Kết nối Redis
	rdb, err := config.ConnectRedis()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 0, "mess": "Không thể kết nối Redis"})
		return
	}

	var allAccommodations []models.Accommodation

	// Lấy dữ liệu từ Redis
	if err := services.GetFromRedis(config.Ctx, rdb, cacheKey, &allAccommodations); err != nil || len(allAccommodations) == 0 {
		tx := config.DB.Model(&models.Accommodation{}).
			Preload("Rooms").
			Preload("Rates").
			Preload("Benefits").
			Preload("User").
			Preload("User.Banks")
		if currentUserRole == 2 {
			//Lấy data theo vai trò Admin (Role = 2)
			tx = tx.Where("user_id = ?", currentUserID)
		} else if currentUserRole == 3 {
			//Lấy data theo vai trò Receptionist (Role = 3)
			var adminID int
			if err := config.DB.Model(&models.User{}).Select("admin_id").Where("id = ?", currentUserID).Scan(&adminID).Error; err != nil {
				c.JSON(http.StatusForbidden, gin.H{"code": 0, "mess": "Không có quyền truy cập"})
				return
			}
			tx = tx.Where("user_id = ?", adminID)
		}

		// Lấy dữ liệu từ DB
		if err := tx.Find(&allAccommodations).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 0, "mess": "Không thể lấy danh sách chỗ ở"})
			return
		}

		accommodationsResponse := make([]dto.AccommodationResponse, 0)
		for _, acc := range allAccommodations {

			accommodationsResponse = append(accommodationsResponse, dto.AccommodationResponse{
				ID:               acc.ID,
				Type:             acc.Type,
				Name:             acc.Name,
				Address:          acc.Address,
				CreateAt:         acc.CreateAt,
				UpdateAt:         acc.UpdateAt,
				Avatar:           acc.Avatar,
				ShortDescription: acc.ShortDescription,
				Status:           acc.Status,
				Num:              acc.Num,
				People:           acc.People,
				Price:            acc.Price,
				NumBed:           acc.NumBed,
				NumTolet:         acc.NumTolet,
				Province:         acc.Province,
				District:         acc.District,
				Ward:             acc.Ward,
				Benefits:         acc.Benefits,
				Longitude:        acc.Longitude,
				Latitude:         acc.Latitude,
			})
		}

		// Lưu dữ liệu đã ép kiểu vào Redis
		if err := services.SetToRedis(config.Ctx, rdb, cacheKey, accommodationsResponse, 60*time.Minute); err != nil {
			log.Printf("Lỗi khi lưu danh sách chỗ ở vào Redis: %v", err)
		}

	}

	// Áp dụng filter từ dữ liệu cache
	typeFilter := c.Query("type")
	statusFilter := c.Query("status")
	nameFilter := c.Query("name")
	numBedFilter := c.Query("numBed")
	numToletFilter := c.Query("numTolet")
	peopleFilter := c.Query("people")
	provinceFilter := c.Query("province")

	filteredAccommodations := make([]models.Accommodation, 0)
	for _, acc := range allAccommodations {
		if typeFilter != "" {
			parsedTypeFilter, err := strconv.Atoi(typeFilter)
			if err == nil && acc.Type != parsedTypeFilter {
				continue
			}
		}
		if statusFilter != "" {
			parsedStatusFilter, err := strconv.Atoi(statusFilter)
			if err == nil && acc.Status != parsedStatusFilter {
				continue
			}
		}
		if provinceFilter != "" {
			decodedProvinceFilter, _ := url.QueryUnescape(provinceFilter)
			if !strings.Contains(strings.ToLower(acc.Province), strings.ToLower(decodedProvinceFilter)) {
				continue
			}
		}
		if nameFilter != "" {
			decodedNameFilter, _ := url.QueryUnescape(nameFilter)
			if !strings.Contains(strings.ToLower(acc.Name), strings.ToLower(decodedNameFilter)) {
				continue
			}
		}
		if numBedFilter != "" {
			numBed, _ := strconv.Atoi(numBedFilter)
			if acc.NumBed != numBed {
				continue
			}
		}
		if numToletFilter != "" {
			numTolet, _ := strconv.Atoi(numToletFilter)
			if acc.NumTolet != numTolet {
				continue
			}
		}
		if peopleFilter != "" {
			people, _ := strconv.Atoi(peopleFilter)
			if acc.People != people {
				continue
			}
		}
		filteredAccommodations = append(filteredAccommodations, acc)
	}
	total := len(filteredAccommodations)

	//Xếp theo update mới nhất
	sort.Slice(filteredAccommodations, func(i, j int) bool {
		return filteredAccommodations[i].UpdateAt.After(filteredAccommodations[j].UpdateAt)
	})
	// Pagination
	page := 0
	limit := 10
	if pageStr := c.Query("page"); pageStr != "" {
		if parsedPage, err := strconv.Atoi(pageStr); err == nil && parsedPage >= 0 {
			page = parsedPage
		}
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	start := page * limit
	end := start + limit
	if start >= len(filteredAccommodations) {
		filteredAccommodations = []models.Accommodation{}
	} else if end > len(filteredAccommodations) {
		filteredAccommodations = filteredAccommodations[start:]
	} else {
		filteredAccommodations = filteredAccommodations[start:end]
	}

	// Chuẩn bị response
	accommodationsResponse := make([]dto.AccommodationResponse, 0)
	for _, acc := range filteredAccommodations {
		accommodationsResponse = append(accommodationsResponse, dto.AccommodationResponse{
			ID:               acc.ID,
			Type:             acc.Type,
			Name:             acc.Name,
			Address:          acc.Address,
			CreateAt:         acc.CreateAt,
			UpdateAt:         acc.UpdateAt,
			Avatar:           acc.Avatar,
			ShortDescription: acc.ShortDescription,
			Status:           acc.Status,
			Num:              acc.Num,
			People:           acc.People,
			Price:            acc.Price,
			NumBed:           acc.NumBed,
			NumTolet:         acc.NumTolet,
			Province:         acc.Province,
			District:         acc.District,
			Ward:             acc.Ward,
			Benefits:         acc.Benefits,
			Longitude:        acc.Longitude,
			Latitude:         acc.Latitude,
		})
	}

	response.SuccessWithPagination(c, accommodationsResponse, page, limit, total)
}

// Hàm chuẩn hóa chuỗi
func normalizeInput(input string) string {
	input = strings.TrimSpace(input)
	input = strings.ToLower(unidecode.Unidecode(input))
	return input
}

// Tạo đối tượng closestmatch cho danh sách từ khóa
func createMatcher(keywords []string) *closestmatch.ClosestMatch {
	return closestmatch.New(keywords, []int{2, 3})
}

// Tính độ tương đồng giữa hai chuỗi
func calculateSimilarity(a, b string) float64 {
	distance := levenshtein.DistanceForStrings([]rune(a), []rune(b), levenshtein.DefaultOptions)
	maxLen := float64(len(a))
	if float64(len(b)) > maxLen {
		maxLen = float64(len(b))
	}

	if maxLen == 0 {
		return 1.0 // Nếu cả hai chuỗi đều rỗng, tương đồng là 100%
	}

	similarity := 1.0 - float64(distance)/maxLen
	return similarity
}

func extractRatingFromQuery(query string) int {
	// Sử dụng regex để tìm các số nguyên đi kèm từ "sao"
	re := regexp.MustCompile(`(\d+)\s*sao`) // Bắt số trước từ "sao"
	match := re.FindStringSubmatch(query)
	if len(match) < 2 {
		return -1 // Không tìm thấy xếp hạng sao
	}

	// Chuyển đổi xếp hạng sao từ chuỗi thành số nguyên
	ratingInt, err := strconv.Atoi(match[1])
	if err != nil {
		return -1
	}
	return ratingInt
}

// Tách thông tin từ query và ánh xạ type cùng xếp hạng sao
func parseAccommodationType(query string) (int, int) {
	// Danh sách từ khóa cho từng loại accommodation
	hotelKeywords := []string{"khách sạn", "hotel", "khach san", "ks"}
	homestayKeywords := []string{"homestay", "căn hộ", "nhà", "nhà nguyên căn", "can ho"}
	villaKeywords := []string{"villa", "biệt thự", "nhà nguyên căn"}

	// Chuẩn hóa query
	normalizedQuery := normalizeInput(query)
	rating := extractRatingFromQuery(normalizedQuery)

	// Tạo matcher cho từng nhóm từ khóa
	hotelMatcher := createMatcher(hotelKeywords)
	homestayMatcher := createMatcher(homestayKeywords)
	villaMatcher := createMatcher(villaKeywords)

	// Tìm từ khóa gần đúng cho từng nhóm
	hotelMatch := hotelMatcher.Closest(normalizedQuery)
	homestayMatch := homestayMatcher.Closest(normalizedQuery)
	villaMatch := villaMatcher.Closest(normalizedQuery)

	// Kiểm tra độ khớp rõ ràng nhất (ưu tiên kết quả đầu tiên khớp)
	if hotelMatch != "" && strings.Contains(normalizedQuery, hotelMatch) {
		// Kiểm tra xếp hạng sao

		return 0, rating // Hotel
	}
	if homestayMatch != "" && strings.Contains(normalizedQuery, homestayMatch) {
		rating := extractRatingFromQuery(normalizedQuery)
		return 1, rating // Homestay
	}
	if villaMatch != "" && strings.Contains(normalizedQuery, villaMatch) {
		rating := extractRatingFromQuery(normalizedQuery)
		return 2, rating // Villa
	}

	// Trả về -1 nếu không khớp
	return -1, -1
}

// Tạo danh sách các giá trị duy nhất từ cơ sở dữ liệu cho closestmatch
func prepareUniqueList(accommodations []models.Accommodation, field string) []string {
	uniqueValues := make(map[string]bool)

	for _, acc := range accommodations {
		var value string
		switch field {
		case "province":
			value = acc.Province
		case "ward":
			value = acc.Ward
		}
		if value != "" {
			uniqueValues[normalizeInput(value)] = true
		}
	}

	uniqueList := make([]string, 0, len(uniqueValues))
	for val := range uniqueValues {
		uniqueList = append(uniqueList, val)
	}
	return uniqueList
}

// Tính điểm phù hợp cho accommodation
func calculateScore(query string, acc models.Accommodation, cmProvince, cmWard *closestmatch.ClosestMatch) int {
	normalizedQuery := normalizeInput(query)
	accType, rating := parseAccommodationType(normalizedQuery)
	score := 0

	if accType != -1 && accType == acc.Type {
		score += 20
	}
	if rating != -1 && acc.Num == rating {
		score += 15
	}
	score += calculateLocationScore(normalizedQuery, acc, cmProvince, cmWard)
	score += calculateBenefitScore(normalizedQuery, acc.Benefits)

	return score
}

func calculateLocationScore(query string, acc models.Accommodation, cmProvince, cmWard *closestmatch.ClosestMatch) int {
	score := 0
	if cmProvince.Closest(query) == normalizeInput(acc.Province) {
		score += 13
	}
	if cmWard.Closest(query) == normalizeInput(acc.Ward) {
		score += 1
	}
	return score
}

func calculateBenefitScore(query string, benefits []models.Benefit) int {
	maxBenefitScore := 12
	benefitScore := 0

	for _, benefit := range benefits {
		normalizedBenefit := normalizeInput(benefit.Name)
		similarity := calculateSimilarity(query, normalizedBenefit)
		if similarity > 0.7 || strings.Contains(query, normalizedBenefit) {
			benefitScore += 4
			if benefitScore >= maxBenefitScore {
				break
			}
		}
	}
	return benefitScore
}

func filterAndScoreAccommodations(
	query string,
	accommodations []models.Accommodation,
	cmProvince, cmWard *closestmatch.ClosestMatch,
) []dto.ScoredAccommodation {
	var filteredAccommodations []dto.ScoredAccommodation
	scoreCh := make(chan dto.ScoredAccommodation, len(accommodations))
	var wg sync.WaitGroup

	for _, acc := range accommodations {
		wg.Add(1)
		go func(acc models.Accommodation) {
			defer wg.Done()
			score := calculateScore(query, acc, cmProvince, cmWard)
			if score > 0 {
				scoreCh <- dto.ScoredAccommodation{
					Accommodation: acc,
					Score:         score,
				}
			}
		}(acc)
	}

	go func() {
		wg.Wait()
		close(scoreCh)
	}()

	for scoredAcc := range scoreCh {
		filteredAccommodations = append(filteredAccommodations, scoredAcc)
	}

	sort.SliceStable(filteredAccommodations, func(i, j int) bool {
		return filteredAccommodations[i].Score > filteredAccommodations[j].Score
	})
	return filteredAccommodations
}

// Load dữ liệu từ DB
func loadAccommodationsFromDB(allAccommodations *[]models.Accommodation) error {
	return config.DB.Model(&models.Accommodation{}).
		Preload("Rooms").
		Preload("Rates").
		Preload("Benefits").
		Preload("User").
		Preload("User.Banks").
		Find(allAccommodations).Error
}

func isMatch(acc models.Accommodation, typeFilter, nameFilter, statusFilter, provinceFilter, districtFilter, numBedFilter, numToletFilter, peopleFilter, numFilter string, benefitIDs []int, statusMap map[uint]bool) bool {
	// Kiểm tra trạng thái có trong statusMap
	if _, exists := statusMap[uint(acc.ID)]; exists {
		return false
	}

	// Kiểm tra lọc theo typeFilter
	if typeFilter != "" {
		parsedTypeFilter, err := strconv.Atoi(typeFilter)
		if err == nil && acc.Type != parsedTypeFilter {
			return false
		}
	}

	if nameFilter != "" {
		decodedNameFilter, _ := url.QueryUnescape(nameFilter)
		if !strings.Contains(strings.ToLower(acc.Name), strings.ToLower(decodedNameFilter)) {
			return false
		}
	}

	// Kiểm tra lọc theo statusFilter
	if statusFilter != "" {
		parsedStatusFilter, err := strconv.Atoi(statusFilter)
		if err == nil && acc.Status != parsedStatusFilter {
			return false
		}
	}

	// Kiểm tra lọc theo provinceFilter
	if provinceFilter != "" {
		decodedProvinceFilter, _ := url.QueryUnescape(provinceFilter)
		if !strings.Contains(strings.ToLower(acc.Province), strings.ToLower(decodedProvinceFilter)) {
			return false
		}
	}

	// Kiểm tra lọc theo districtFilter
	if districtFilter != "" {
		decodedDistrictFilter, _ := url.QueryUnescape(districtFilter)
		if !strings.Contains(strings.ToLower(acc.District), strings.ToLower(decodedDistrictFilter)) {
			return false
		}
	}

	// Kiểm tra lọc theo nameFilter (chuỗi gần đúng)

	// Kiểm tra lọc theo numBedFilter
	if numBedFilter != "" {
		numBed, _ := strconv.Atoi(numBedFilter)
		if acc.NumBed != numBed {
			return false
		}
	}

	// Kiểm tra lọc theo numToletFilter
	if numToletFilter != "" {
		numTolet, _ := strconv.Atoi(numToletFilter)
		if acc.NumTolet != numTolet {
			return false
		}
	}

	// Kiểm tra lọc theo peopleFilter
	if peopleFilter != "" {
		people, _ := strconv.Atoi(peopleFilter)
		if acc.People != people {
			return false
		}
	}

	// Kiểm tra lọc theo numFilter
	if numFilter != "" {
		num, _ := strconv.Atoi(numFilter)
		if acc.Num != num {
			return false
		}
	}

	// Kiểm tra lọc theo benefitIDs
	if len(benefitIDs) > 0 {
		match := false
		for _, benefit := range acc.Benefits {
			for _, id := range benefitIDs {
				if benefit.Id == id {
					match = true
					break
				}
			}
			if match {
				break
			}
		}
		if !match {
			return false
		}
	}

	return true
}

func GetAllAccommodationsForUser(c *gin.Context) {
	// Các tham số filter
	typeFilter := c.Query("type")
	provinceFilter := c.Query("province")
	districtFilter := c.Query("district")
	benefitFilterRaw := c.Query("benefitId")
	numFilter := c.Query("num")
	statusFilter := c.Query("status")
	nameFilter := c.Query("name")
	numBedFilter := c.Query("numBed")
	numToletFilter := c.Query("numTolet")
	peopleFilter := c.Query("people")
	searchQuery := c.Query("search")

	fromDateStr := c.Query("fromDate")
	toDateStr := c.Query("toDate")

	pageStr := c.Query("page")
	limitStr := c.Query("limit")

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

	var fromDate, toDate time.Time
	var err error

	if fromDateStr != "" {
		fromDate, err = time.Parse("02/01/2006", fromDateStr)

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 0, "mess": "Dữ liệu fromDate không hợp lệ"})
			return
		}
	}

	if toDateStr != "" {
		toDate, err = time.Parse("02/01/2006", toDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 0, "mess": "Dữ liệu toDate không hợp lệ"})
			return
		}
	}

	statuses, err := getAllAccommodationStatuses(c, fromDate, toDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 0, "mess": "Không thể lấy trạng thái của accommodation"})
		return
	}

	statusMap := make(map[uint]bool)
	for _, status := range statuses {
		statusFromDate := status.FromDate.Truncate(24 * time.Hour)
		statusToDate := status.ToDate.Truncate(24 * time.Hour)

		if !(toDate.Before(statusFromDate) || fromDate.After(statusToDate)) {
			statusMap[status.AccommodationID] = true
		}
	}

	// Redis cache key
	cacheKey := "accommodations:all"
	rdb, err := config.ConnectRedis()
	if err != nil {
		c.JSON(http.StatusMovedPermanently, gin.H{"code": 0, "mess": "Không thể kết nối Redis"})
	}

	var allAccommodations []models.Accommodation

	// Lấy dữ liệu từ Redis
	if err := services.GetFromRedis(config.Ctx, rdb, cacheKey, &allAccommodations); err != nil || len(allAccommodations) == 0 {
		// Nếu không có dữ liệu trong Redis, lấy từ Database
		if err := loadAccommodationsFromDB(&allAccommodations); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 0, "mess": "Không thể lấy danh sách chỗ ở"})
			return
		}

		// Ép kiểu sang AccommodationResponse
		accommodationsResponse := make([]dto.AccommodationDetailResponse, 0)
		for _, acc := range allAccommodations {
			// Lấy thông tin User
			user := acc.User
			accommodationsResponse = append(accommodationsResponse, dto.AccommodationDetailResponse{
				ID:               acc.ID,
				Type:             acc.Type,
				Name:             acc.Name,
				Address:          acc.Address,
				CreateAt:         acc.CreateAt,
				UpdateAt:         acc.UpdateAt,
				Avatar:           acc.Avatar,
				ShortDescription: acc.ShortDescription,
				Status:           acc.Status,
				Num:              acc.Num,
				Furniture:        acc.Furniture,
				People:           acc.People,
				Price:            acc.Price,
				NumBed:           acc.NumBed,
				NumTolet:         acc.NumTolet,
				Benefits:         acc.Benefits,
				TimeCheckIn:      acc.TimeCheckIn,
				TimeCheckOut:     acc.TimeCheckOut,
				Province:         acc.Province,
				District:         acc.District,
				Ward:             acc.Ward,
				Longitude:        acc.Longitude,
				Latitude:         acc.Latitude,
				User: dto.Actor{
					Name:          user.Name,
					Email:         user.Email,
					PhoneNumber:   user.PhoneNumber,
					BankShortName: user.Banks[0].BankShortName,
					AccountNumber: user.Banks[0].AccountNumber,
					BankName:      user.Banks[0].BankName,
				},
			})
		}

		// Lưu dữ liệu đã ép kiểu vào Redis
		if err := services.SetToRedis(config.Ctx, rdb, cacheKey, accommodationsResponse, 60*time.Minute); err != nil {
			log.Printf("Lỗi khi lưu danh sách chỗ ở vào Redis: %v", err)
		}
	}

	benefitIDs := make([]int, 0)
	//chuyển đổi thành slice int (query mặc đinh string)
	if benefitFilterRaw != "" {
		// Loại bỏ các ký tự "[" và "]"
		benefitFilterRaw = strings.Trim(benefitFilterRaw, "[]")

		// Tách các phần tử bằng dấu phẩy
		benefitStrs := strings.Split(benefitFilterRaw, ",")

		// Chuyển đổi từng phần tử thành int
		for _, benefitStr := range benefitStrs {
			if benefitID, err := strconv.Atoi(strings.TrimSpace(benefitStr)); err == nil {
				benefitIDs = append(benefitIDs, benefitID)
			}
		}
	}

	cmProvince := createMatcher(prepareUniqueList(allAccommodations, "province"))
	cmWard := createMatcher(prepareUniqueList(allAccommodations, "ward"))

	// Áp dụng filter trên dữ liệu từ Redis

	filteredAccommodations := make([]models.Accommodation, 0)
	for _, acc := range allAccommodations {
		// Loại bỏ các accommodation đã bị đặt chồng lấn
		if _, exists := statusMap[acc.ID]; exists {
			continue
		}

		if !isMatch(acc, typeFilter, nameFilter, statusFilter, provinceFilter, districtFilter, numBedFilter, numToletFilter, peopleFilter, numFilter, []int{}, statusMap) {
			continue
		}
		filteredAccommodations = append(filteredAccommodations, acc)
	}

	// Xử lý search query
	if searchQuery != "" {

		scoredAccommodations := filterAndScoreAccommodations(searchQuery, filteredAccommodations, cmProvince, cmWard)
		filteredAccommodations = []models.Accommodation{}
		for _, scoredAcc := range scoredAccommodations {
			filteredAccommodations = append(filteredAccommodations, scoredAcc.Accommodation)
		}
	}

	//Xếp theo update mới nhất
	sort.Slice(filteredAccommodations, func(i, j int) bool {
		return filteredAccommodations[i].Num > (filteredAccommodations[j].Num)
	})

	// Pagination
	// Lấy total sau khi lọc
	total := len(filteredAccommodations)

	// Áp dụng phân trang
	start := page * limit
	end := start + limit
	if start >= total {
		filteredAccommodations = []models.Accommodation{}
	} else if end > total {
		filteredAccommodations = filteredAccommodations[start:]
	} else {
		filteredAccommodations = filteredAccommodations[start:end]
	}

	// Chuẩn bị response
	accommodationsResponse := make([]dto.AccommodationResponse, 0)
	for _, acc := range filteredAccommodations {
		accommodationsResponse = append(accommodationsResponse, dto.AccommodationResponse{
			ID:               acc.ID,
			Type:             acc.Type,
			Name:             acc.Name,
			Address:          acc.Address,
			CreateAt:         acc.CreateAt,
			UpdateAt:         acc.UpdateAt,
			Avatar:           acc.Avatar,
			ShortDescription: acc.ShortDescription,
			Status:           acc.Status,
			Num:              acc.Num,
			People:           acc.People,
			Price:            acc.Price,
			NumBed:           acc.NumBed,
			NumTolet:         acc.NumTolet,
			Province:         acc.Province,
			District:         acc.District,
			Ward:             acc.Ward,
			Benefits:         acc.Benefits,
			Longitude:        acc.Longitude,
			Latitude:         acc.Latitude,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 1,
		"mess": "Lấy danh sách chỗ ở thành công",
		"data": accommodationsResponse,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	})
}

// Hàm kiểm tra benefit
func normalizeBenefitName(name string) string {
	name = strings.ToLower(name)
	fields := strings.Fields(name)
	normalized := strings.Join(fields, " ")
	return normalized
}

func CreateAccommodation(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 0, "mess": "Authorization header is missing"})
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	currentUserID, currentUserRole, err := GetUserIDFromToken(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 0, "mess": "Invalid token"})
		return
	}
	var newAccommodation models.Accommodation
	var user models.User
	if err := config.DB.First(&user, currentUserID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusBadRequest, gin.H{"code": 0, "mess": "Người dùng không tồn tại"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 0, "mess": "Lỗi khi kiểm tra người dùng", "details": err.Error()})
		return
	}
	newAccommodation.UserID = currentUserID
	newAccommodation.User = user
	if err := c.ShouldBindJSON(&newAccommodation); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 0, "mess": "Dữ liệu đầu vào không hợp lệ", "details": err.Error()})
		return
	}

	if err := newAccommodation.ValidateType(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 0, "mess": err.Error()})
		return
	}

	if err := newAccommodation.ValidateStatus(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 0, "mess": err.Error()})
		return
	}

	imgJSON, err := json.Marshal(newAccommodation.Img)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 0, "mess": "Không thể mã hóa hình ảnh", "details": err.Error()})
		return
	}

	newAccommodation.Img = imgJSON

	var benefits []models.Benefit

	for _, benefit := range newAccommodation.Benefits {
		if benefit.Id != 0 {

			benefits = append(benefits, benefit)
		} else {

			normalizedBenefitName := normalizeBenefitName(benefit.Name)

			newBenefit := models.Benefit{Name: normalizedBenefitName}
			if err := config.DB.Where("LOWER(TRIM(name)) = ?", normalizedBenefitName).FirstOrCreate(&newBenefit).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"code": 0, "message": "Không thể tạo mới tiện ích", "details": err.Error()})
				return
			}

			benefits = append(benefits, newBenefit)
		}
	}

	newAccommodation.Benefits = benefits

	latitude, longitude, err := services.GetCoordinatesFromAddress(
		newAccommodation.Address,
		newAccommodation.District,
		newAccommodation.Province,
		newAccommodation.Ward,
		os.Getenv("MAPBOX_KEY"),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 0, "message": "Không thể mã hóa địa chỉ", "details": err.Error()})
	}
	// Kiểm tra xem tọa độ có bị trùng không
	var existingAccommodation models.Accommodation
	if err := config.DB.Where("longitude = ? AND latitude = ?", longitude, latitude).First(&existingAccommodation).Error; err == nil {
		c.JSON(http.StatusOK, gin.H{"code": 1,
			"mess":              "Địa chỉ này đã được sử dụng, vui lòng nhập địa chỉ khác.",
			"needChangeAddress": true})
		return
	}
	newAccommodation.Longitude = longitude
	newAccommodation.Latitude = latitude

	if err := config.DB.Create(&newAccommodation).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 0, "mess": "Không thể tạo chỗ ở", "details": err.Error()})
		return
	}
	// Xử lý Redis cache
	rdb, redisErr := config.ConnectRedis()
	if redisErr == nil {
		switch currentUserRole {
		case 1: // Super Admin
			_ = services.DeleteFromRedis(config.Ctx, rdb, "accommodations:all")
		case 2: // Admin
			// Xóa cache của admin
			adminCacheKey := fmt.Sprintf("accommodations:admin:%d", currentUserID)
			_ = services.DeleteFromRedis(config.Ctx, rdb, "accommodations:all")
			_ = services.DeleteFromRedis(config.Ctx, rdb, adminCacheKey)
			var receptionistIDs []int
			if err := config.DB.Model(&models.User{}).Where("admin_id = ?", currentUserID).Pluck("id", &receptionistIDs).Error; err == nil {
				for _, receptionistID := range receptionistIDs {
					receptionistCacheKey := fmt.Sprintf("accommodations:receptionist:%d", receptionistID)
					_ = services.DeleteFromRedis(config.Ctx, rdb, receptionistCacheKey)
				}
			}
		case 3: // Receptionist
			var adminID int
			_ = services.DeleteFromRedis(config.Ctx, rdb, "accommodations:all")
			if err := config.DB.Model(&models.User{}).Select("admin_id").Where("id = ?", currentUserID).Scan(&adminID).Error; err == nil {
				receptionistCacheKey := fmt.Sprintf("accommodations:receptionist:%d", currentUserID)
				adminCacheKey := fmt.Sprintf("accommodations:admin:%d", adminID)
				_ = services.DeleteFromRedis(config.Ctx, rdb, receptionistCacheKey)
				_ = services.DeleteFromRedis(config.Ctx, rdb, adminCacheKey)
			}
		}
	}
	response := dto.AccommodationDetailResponse{
		ID:               newAccommodation.ID,
		Type:             newAccommodation.Type,
		Name:             newAccommodation.Name,
		Address:          newAccommodation.Address,
		CreateAt:         newAccommodation.CreateAt,
		UpdateAt:         newAccommodation.UpdateAt,
		Avatar:           newAccommodation.Avatar,
		ShortDescription: newAccommodation.ShortDescription,
		Status:           newAccommodation.Status,
		Num:              newAccommodation.Num,
		Furniture:        newAccommodation.Furniture,
		People:           newAccommodation.People,
		Price:            newAccommodation.Price,
		NumBed:           newAccommodation.NumBed,
		NumTolet:         newAccommodation.NumTolet,
		Benefits:         newAccommodation.Benefits,
		TimeCheckIn:      newAccommodation.TimeCheckIn,
		TimeCheckOut:     newAccommodation.TimeCheckOut,
		Province:         newAccommodation.Province,
		District:         newAccommodation.District,
		Ward:             newAccommodation.Ward,
		Longitude:        newAccommodation.Longitude,
		Latitude:         newAccommodation.Latitude,
		User: dto.Actor{
			Name:        user.Name,
			Email:       user.Email,
			PhoneNumber: user.PhoneNumber,
		},
	}

	c.JSON(http.StatusOK, gin.H{"code": 1, "mess": "Tạo chỗ ở thành công", "data": response})
}

func GetAccommodationDetail(c *gin.Context) {
	accommodationId := c.Param("id")

	// Kết nối Redis
	rdb, redisErr := config.ConnectRedis()
	if redisErr != nil {
		response.ServerError(c)
		return
	}

	// Key cache cho tất cả accommodations
	cacheKey := "allaccommodations:all"

	// Lấy danh sách accommodations từ cache
	var cachedAccommodations []models.Accommodation
	if err := services.GetFromRedis(config.Ctx, rdb, cacheKey, &cachedAccommodations); err == nil {
		for _, acc := range cachedAccommodations {
			if fmt.Sprintf("%d", acc.ID) == accommodationId {
				var price int
				if acc.Type == 0 {
					price = getLowestPriceFromRooms(acc.Rooms)
				} else {
					price = acc.Price
				}
				// Tạo response từ cache
				resp := dto.AccommodationDetailResponse{
					ID:               acc.ID,
					Type:             acc.Type,
					Name:             acc.Name,
					Address:          acc.Address,
					CreateAt:         acc.CreateAt,
					UpdateAt:         acc.UpdateAt,
					Avatar:           acc.Avatar,
					Img:              acc.Img,
					ShortDescription: acc.ShortDescription,
					Description:      acc.Description,
					Status:           acc.Status,
					Num:              acc.Num,
					People:           acc.People,
					Price:            price,
					NumBed:           acc.NumBed,
					NumTolet:         acc.NumTolet,
					Furniture:        acc.Furniture,
					Benefits:         acc.Benefits,
					TimeCheckIn:      acc.TimeCheckIn,
					TimeCheckOut:     acc.TimeCheckOut,
					Province:         acc.Province,
					District:         acc.District,
					Ward:             acc.Ward,
					Longitude:        acc.Longitude,
					Latitude:         acc.Latitude,
					User: dto.Actor{
						Name:        acc.User.Name,
						Email:       acc.User.Email,
						PhoneNumber: acc.User.PhoneNumber,
					},
				}
				response.Success(c, resp)
				return
			}
		}
	}

	// Nếu không tìm thấy trong cache, truy vấn từ database
	var accommodation models.Accommodation
	if err := config.DB.Preload("Rooms").
		Preload("Rates").
		Preload("Benefits").
		Preload("User").Preload("User.Banks").First(&accommodation, accommodationId).Error; err != nil {
		response.NotFound(c)
		return
	}

	var price int
	if accommodation.Type == 0 {
		err := config.DB.Table("rooms").
			Where("accommodation_id = ?", accommodationId).
			Select("MIN(price)").
			Scan(&price).Error
		if err != nil {
			price = 0
		}
	} else {

		price = accommodation.Price
	}
	resp := dto.AccommodationDetailResponse{
		ID:               accommodation.ID,
		Type:             accommodation.Type,
		Name:             accommodation.Name,
		Address:          accommodation.Address,
		CreateAt:         accommodation.CreateAt,
		UpdateAt:         accommodation.UpdateAt,
		Avatar:           accommodation.Avatar,
		Img:              accommodation.Img,
		ShortDescription: accommodation.ShortDescription,
		Description:      accommodation.Description,
		Status:           accommodation.Status,
		Num:              accommodation.Num,
		People:           accommodation.People,
		Price:            price,
		NumBed:           accommodation.NumBed,
		NumTolet:         accommodation.NumTolet,
		Furniture:        accommodation.Furniture,
		Benefits:         accommodation.Benefits,
		TimeCheckIn:      accommodation.TimeCheckIn,
		TimeCheckOut:     accommodation.TimeCheckOut,
		Province:         accommodation.Province,
		District:         accommodation.District,
		Ward:             accommodation.Ward,
		Longitude:        accommodation.Longitude,
		Latitude:         accommodation.Latitude,
		User: dto.Actor{
			Name:        accommodation.User.Name,
			Email:       accommodation.User.Email,
			PhoneNumber: accommodation.User.PhoneNumber,
		},
	}

	response.Success(c, resp)
}

func UpdateAccommodation(c *gin.Context) {
	var request dto.AccommodationRequest
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		response.Unauthorized(c)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	currentUserID, currentUserRole, err := GetUserIDFromToken(tokenString)
	if err != nil {
		response.Unauthorized(c)
		return
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "Dữ liệu đầu vào không hợp lệ")
		return
	}

	var accommodation models.Accommodation

	if err := config.DB.Preload("User").Preload("Rooms").Preload("Rates").First(&accommodation, request.ID).Error; err != nil {
		response.NotFound(c)
		return
	}

	// Xử lý trường Img
	imgJSON, err := json.Marshal(request.Img)
	if err != nil {
		response.ServerError(c)
		return
	}

	// Xử lý trường Furniture
	furnitureJson, err := json.Marshal(request.Furniture)
	if err != nil {
		response.ServerError(c)
		return
	}
	latitude, longitude, err := services.GetCoordinatesFromAddress(
		request.Address,
		request.District,
		request.Province,
		request.Ward,
		os.Getenv("MAPBOX_KEY"),
	)
	if err != nil {
		response.ServerError(c)
	}
	if request.Type != -1 {
		accommodation.Type = request.Type
	}

	if request.Name != "" {
		accommodation.Name = request.Name
	}

	if request.Address != "" {
		accommodation.Address = request.Address
	}
	if request.Price != -1 {
		accommodation.Price = request.Price
	}
	if request.Avatar != "" {
		accommodation.Avatar = request.Avatar
	}

	if request.ShortDescription != "" {
		accommodation.ShortDescription = request.ShortDescription
	}

	if request.Description != "" {
		accommodation.Description = request.Description
	}

	if request.Status != 0 {
		accommodation.Status = request.Status
	}

	if len(request.Img) > 0 {
		accommodation.Img = imgJSON
	}

	if len(request.Furniture) > 0 {
		accommodation.Furniture = furnitureJson
	}

	if request.People != 0 {
		accommodation.People = request.People
	}

	if request.TimeCheckIn != "" {
		accommodation.TimeCheckIn = request.TimeCheckIn
	}

	if request.TimeCheckOut != "" {
		accommodation.TimeCheckOut = request.TimeCheckOut
	}

	if request.Province != "" {
		accommodation.Province = request.Province
	}

	if request.District != "" {
		accommodation.District = request.District
	}

	if request.Ward != "" {
		accommodation.Ward = request.Ward
	}

	if longitude != 0 && latitude != 0 {
		accommodation.Longitude = longitude
		accommodation.Latitude = latitude
	}
	var benefits []models.Benefit
	for _, benefit := range request.Benefits {
		if benefit.Id != 0 {
			benefits = append(benefits, benefit)
		} else {

			newBenefit := models.Benefit{Name: benefit.Name}
			if err := config.DB.Where("name = ?", benefit.Name).FirstOrCreate(&newBenefit).Error; err != nil {
				response.ServerError(c)
				return
			}
			benefits = append(benefits, newBenefit)
		}
	}

	if err := config.DB.Model(&accommodation).Association("Benefits").Replace(benefits); err != nil {
		response.ServerError(c)
		return
	}

	if err := config.DB.Save(&accommodation).Error; err != nil {
		response.ServerError(c)
		return
	}
	// Xử lý Redis cache
	rdb, redisErr := config.ConnectRedis()
	if redisErr == nil {
		switch currentUserRole {
		case 1: // Super Admin
			_ = services.DeleteFromRedis(config.Ctx, rdb, "accommodations:all")
			_ = services.DeleteFromRedis(config.Ctx, rdb, "benefits:all")
		case 2: // Admin
			// Xóa cache của admin
			adminCacheKey := fmt.Sprintf("accommodations:admin:%d", currentUserID)
			_ = services.DeleteFromRedis(config.Ctx, rdb, adminCacheKey)
			_ = services.DeleteFromRedis(config.Ctx, rdb, "accommodations:all")
			_ = services.DeleteFromRedis(config.Ctx, rdb, "benefits:all")
			var receptionistIDs []int
			if err := config.DB.Model(&models.User{}).Where("admin_id = ?", currentUserID).Pluck("id", &receptionistIDs).Error; err == nil {
				for _, receptionistID := range receptionistIDs {
					receptionistCacheKey := fmt.Sprintf("accommodations:receptionist:%d", receptionistID)
					_ = services.DeleteFromRedis(config.Ctx, rdb, receptionistCacheKey)
				}
			}
		case 3: // Receptionist
			var adminID int
			_ = services.DeleteFromRedis(config.Ctx, rdb, "benefits:all")
			_ = services.DeleteFromRedis(config.Ctx, rdb, "accommodations:all")
			if err := config.DB.Model(&models.User{}).Select("admin_id").Where("id = ?", currentUserID).Scan(&adminID).Error; err == nil {
				receptionistCacheKey := fmt.Sprintf("accommodations:receptionist:%d", currentUserID)
				adminCacheKey := fmt.Sprintf("accommodations:admin:%d", adminID)
				_ = services.DeleteFromRedis(config.Ctx, rdb, receptionistCacheKey)
				_ = services.DeleteFromRedis(config.Ctx, rdb, adminCacheKey)
			}

		}
	}
	resp := dto.AccommodationDetailResponse{
		ID:               accommodation.ID,
		Type:             accommodation.Type,
		Name:             accommodation.Name,
		Address:          accommodation.Address,
		CreateAt:         accommodation.CreateAt,
		UpdateAt:         accommodation.UpdateAt,
		Avatar:           accommodation.Avatar,
		Img:              accommodation.Img,
		ShortDescription: accommodation.ShortDescription,
		Description:      accommodation.Description,
		Status:           accommodation.Status,
		Num:              accommodation.Num,
		Furniture:        accommodation.Furniture,
		People:           accommodation.People,
		NumBed:           accommodation.NumBed,
		NumTolet:         accommodation.NumTolet,
		Benefits:         benefits,
		TimeCheckIn:      accommodation.TimeCheckIn,
		TimeCheckOut:     accommodation.TimeCheckOut,
		Longitude:        accommodation.Longitude,
		Latitude:         accommodation.Latitude,
	}

	response.Success(c, resp)
}

func ChangeAccommodationStatus(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		response.Unauthorized(c)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	currentUserID, currentUserRole, err := GetUserIDFromToken(tokenString)
	if err != nil {
		response.Unauthorized(c)
		return
	}

	var input struct {
		ID     uint `json:"id"`
		Status int  `json:"status"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "Dữ liệu đầu vào không hợp lệ")
		return
	}

	var accommodation models.Accommodation

	if err := config.DB.First(&accommodation, input.ID).Error; err != nil {
		response.NotFound(c)
		return
	}

	accommodation.Status = input.Status
	if err := config.DB.Save(&accommodation).Error; err != nil {
		response.ServerError(c)
		return
	}
	// Xử lý Redis cache
	rdb, redisErr := config.ConnectRedis()
	if redisErr == nil {
		switch currentUserRole {
		case 1: // Super Admin
			_ = services.DeleteFromRedis(config.Ctx, rdb, "accommodations:all")
		case 2: // Admin
			// Xóa cache của admin
			adminCacheKey := fmt.Sprintf("accommodations:admin:%d", currentUserID)
			_ = services.DeleteFromRedis(config.Ctx, rdb, adminCacheKey)
			var receptionistIDs []int
			if err := config.DB.Model(&models.User{}).Where("admin_id = ?", currentUserID).Pluck("id", &receptionistIDs).Error; err == nil {
				for _, receptionistID := range receptionistIDs {
					receptionistCacheKey := fmt.Sprintf("accommodations:receptionist:%d", receptionistID)
					_ = services.DeleteFromRedis(config.Ctx, rdb, receptionistCacheKey)
				}
			}
		case 3: // Receptionist
			var adminID int
			if err := config.DB.Model(&models.User{}).Select("admin_id").Where("id = ?", currentUserID).Scan(&adminID).Error; err == nil {
				receptionistCacheKey := fmt.Sprintf("accommodations:receptionist:%d", currentUserID)
				adminCacheKey := fmt.Sprintf("accommodations:admin:%d", adminID)
				_ = services.DeleteFromRedis(config.Ctx, rdb, receptionistCacheKey)
				_ = services.DeleteFromRedis(config.Ctx, rdb, adminCacheKey)
			}
		}
	}
	response.Success(c, accommodation)
}

// Hàm lấy giá thấp nhất từ danh sách phòng
func getLowestPriceFromRooms(rooms []models.Room) int {
	lowestPrice := math.MaxInt
	for _, room := range rooms {
		if room.Price < lowestPrice {
			lowestPrice = room.Price
		}
	}
	if lowestPrice == math.MaxInt {
		return 0
	}
	return lowestPrice
}
