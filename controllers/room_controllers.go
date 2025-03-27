package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"new/config"
	"new/dto"
	"new/models"
	"new/response"
	"new/services"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
)

var CacheKey2 = "accommodations:all"

func getRoomStatuses(fromDate, toDate time.Time) ([]models.RoomStatus, error) {
	var (
		statuses         []models.RoomStatus
		filteredStatuses []models.RoomStatus
	)

	// Tạo cache key
	cacheKey := "rooms:statuses"

	// Kết nối Redis
	rdb, err := config.ConnectRedis()
	if err != nil {
		return nil, fmt.Errorf("không thể kết nối Redis: %v", err)
	}

	// Thử lấy dữ liệu từ Redis
	if err := services.GetFromRedis(config.Ctx, rdb, cacheKey, &statuses); err == nil && len(statuses) > 0 {
		// Lọc dữ liệu theo ngày
		filteredStatuses = filterRoomStatusesByDate(statuses, fromDate, toDate)
		return filteredStatuses, nil
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

	// Lọc dữ liệu theo ngày
	filteredStatuses = filterRoomStatusesByDate(statuses, fromDate, toDate)

	return filteredStatuses, nil
}

// Hàm phụ để lọc danh sách trạng thái theo khoảng thời gian
func filterRoomStatusesByDate(statuses []models.RoomStatus, fromDate, toDate time.Time) []models.RoomStatus {
	var filteredStatuses []models.RoomStatus
	for _, status := range statuses {
		if !(status.ToDate.Before(fromDate) || status.FromDate.After(toDate)) {
			filteredStatuses = append(filteredStatuses, status)
		}

	}
	return filteredStatuses
}

func GetRoomBookingDates(c *gin.Context) {
	roomID := c.DefaultQuery("id", "")
	date := c.DefaultQuery("date", "")

	if roomID == "" || date == "" {
		response.BadRequest(c, "id và date là bắt buộc")
		return
	}

	layout := "01/2006"
	parsedDate, err := time.Parse(layout, date)
	if err != nil {
		response.BadRequest(c, "Ngày không hợp lệ, vui lòng sử dụng định dạng mm/yyyy")
		return
	}

	// Tính toán ngày đầu tháng và ngày cuối tháng
	firstDay := time.Date(parsedDate.Year(), parsedDate.Month(), 1, 0, 0, 0, 0, time.UTC)
	lastDay := firstDay.AddDate(0, 1, -1)

	// Tạo danh sách tất cả các ngày trong tháng
	var allDates []time.Time
	for day := firstDay; day.Before(lastDay.AddDate(0, 0, 1)); day = day.AddDate(0, 0, 1) {
		allDates = append(allDates, day)
	}

	// Lấy trạng thái phòng trong tháng yêu cầu
	var statuses []models.RoomStatus
	db := config.DB

	err = db.Where("room_id = ?", roomID).Find(&statuses).Error
	if err != nil {
		log.Printf("Error retrieving room statuses: %v", err)
		response.ServerError(c)
		return
	}

	// Lấy thông tin khách đặt phòng (chỉ lấy guest_name và guest_phone)
	orderMap, err := getGuestBookingsForRoom(roomID)
	if err != nil {
		log.Printf("Error retrieving guest bookings: %v", err)
		response.ServerError(c)
		return
	}

	dateFormat := "02/01/2006"
	var roomResponses []map[string]interface{}

	// Duyệt qua tất cả các ngày trong tháng
	for _, currentDate := range allDates {
		dateStr := currentDate.Format(dateFormat)
		var statusFound bool
		var status int

		// Kiểm tra trạng thái của phòng
		for _, roomStatus := range statuses {
			if currentDate.After(roomStatus.FromDate.AddDate(0, 0, -1)) && !currentDate.After(roomStatus.ToDate) {
				status = roomStatus.Status
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

		// Nếu ngày này có khách đặt phòng, thêm thông tin khách vào response
		if guest, exists := orderMap[dateStr]; exists {
			roomResponse["guest"] = guest
		}

		roomResponses = append(roomResponses, roomResponse)
	}

	response.Success(c, roomResponses)
}

// getGuestBookingsForRoom lấy thông tin khách đặt phòng theo room_id
func getGuestBookingsForRoom(roomID string) (map[string]map[string]string, error) {
	db := config.DB

	// Truy vấn accommodation_id từ bảng Room
	var room models.Room
	err := db.Where("room_id = ?", roomID).First(&room).Error
	if err != nil {
		return nil, fmt.Errorf("lỗi khi lấy thông tin phòng: %v", err)
	}

	accommodationID := room.AccommodationID
	log.Println("accommodation", accommodationID)
	// Truy vấn danh sách đặt phòng từ bảng Orders dựa vào accommodation_id
	var orders []models.Order
	err = db.Where("accommodation_id = ?", accommodationID).Find(&orders).Error
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

func GetAllRooms(c *gin.Context) {
	// Xác thực token
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

	// Lấy các tham số filter
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

	typeFilter := c.Query("type")
	statusFilter := c.Query("status")
	nameFilter := c.Query("name")
	accommodationFilter := c.Query("accommodation")
	numBedFilter := c.Query("numBed")
	numToletFilter := c.Query("numTolet")
	peopleFilter := c.Query("people")

	// Tạo cache key động
	var cacheKey string
	if currentUserRole == 2 {
		cacheKey = fmt.Sprintf("rooms:admin:%d", currentUserID)
	} else if currentUserRole == 3 {
		cacheKey = fmt.Sprintf("rooms:receptionist:%d", currentUserID)
	} else {
		cacheKey = "rooms:all"
	}

	// Kết nối Redis
	rdb, err := config.ConnectRedis()
	if err != nil {
		response.ServerError(c)

	}

	var allRooms []models.Room

	// Lấy dữ liệu từ Redis
	if err := services.GetFromRedis(config.Ctx, rdb, cacheKey, &allRooms); err != nil || len(allRooms) == 0 {
		tx := config.DB.Model(&models.Room{}).Preload("Parent")

		if currentUserRole == 2 {
			// Lấy phòng theo admin
			tx = tx.Joins("JOIN accommodations ON accommodations.id = rooms.accommodation_id").Where("accommodations.user_id = ?", currentUserID)
		} else if currentUserRole == 3 {
			// Lấy phòng theo admin (vị trí receptionist)
			var adminID int
			if err := config.DB.Model(&models.User{}).Select("admin_id").Where("id = ?", currentUserID).Scan(&adminID).Error; err != nil || adminID == 0 {
				response.Forbidden(c)
				return
			}
			tx = tx.Joins("JOIN accommodations ON accommodations.id = rooms.accommodation_id").Where("accommodations.user_id = ?", adminID)
		}

		if err := tx.Find(&allRooms).Error; err != nil {
			response.ServerError(c)
			return
		}

		var roomDetails []dto.RoomDetail
		for _, room := range allRooms {
			roomDetails = append(roomDetails, dto.RoomDetail{
				RoomId:      room.RoomId,
				RoomName:    room.RoomName,
				Type:        room.Type,
				NumBed:      room.NumBed,
				NumTolet:    room.NumTolet,
				Acreage:     room.Acreage,
				Price:       room.Price,
				Description: room.Description,

				CreatedAt: room.CreatedAt,
				UpdatedAt: room.UpdatedAt,
				Status:    room.Status,
				Avatar:    room.Avatar,
				Img:       room.Img,
				Num:       room.Num,
				Furniture: room.Furniture,
				People:    room.People,
				Parent: dto.Parents{
					Id:   room.Parent.ID,
					Name: room.Parent.Name,
				},
			})
		}

		// Lưu dữ liệu ép kiểu vào Redis
		if err := services.SetToRedis(config.Ctx, rdb, cacheKey, roomDetails, 10*time.Minute); err != nil {
			log.Printf("Lỗi khi lưu danh sách phòng vào Redis: %v", err)
		}
	}

	// Áp dụng filter trên dữ liệu từ Redis
	filteredRooms := make([]models.Room, 0)
	for _, room := range allRooms {
		if typeFilter != "" {
			parsedTypeFilter, err := strconv.Atoi(typeFilter)
			if err == nil && room.Type != uint(parsedTypeFilter) {
				continue
			}
		}
		if statusFilter != "" {
			parsedStatus, _ := strconv.Atoi(statusFilter)
			if room.Status != parsedStatus {
				continue
			}
		}
		if nameFilter != "" {
			decodedNameFilter, _ := url.QueryUnescape(nameFilter)
			if !strings.Contains(strings.ToLower(room.RoomName), strings.ToLower(decodedNameFilter)) {
				continue
			}
		}
		if accommodationFilter != "" {
			decodedAccommodationFilter, _ := url.QueryUnescape(accommodationFilter)
			if !strings.Contains(strings.ToLower(room.Parent.Name), strings.ToLower(decodedAccommodationFilter)) {
				continue
			}
		}
		if numBedFilter != "" {
			numBed, _ := strconv.Atoi(numBedFilter)
			if room.NumBed != numBed {
				continue
			}
		}
		if numToletFilter != "" {
			numTolet, _ := strconv.Atoi(numToletFilter)
			if room.NumTolet != numTolet {
				continue
			}
		}
		if peopleFilter != "" {
			people, _ := strconv.Atoi(peopleFilter)
			if room.People != people {
				continue
			}
		}
		filteredRooms = append(filteredRooms, room)
	}

	// Tính toán tổng số phòng sau khi lọc
	total := len(filteredRooms)

	// Pagination
	start := page * limit
	end := start + limit
	if start >= total {
		filteredRooms = []models.Room{}
	} else if end > total {
		filteredRooms = filteredRooms[start:]
	} else {
		filteredRooms = filteredRooms[start:end]
	}

	// Chuẩn bị response
	roomResponses := make([]dto.RoomResponse, 0)
	for _, room := range filteredRooms {
		roomResponses = append(roomResponses, dto.RoomResponse{
			RoomId:    room.RoomId,
			RoomName:  room.RoomName,
			Type:      room.Type,
			NumBed:    room.NumBed,
			NumTolet:  room.NumTolet,
			Acreage:   room.Acreage,
			Price:     room.Price,
			CreatedAt: room.CreatedAt,
			UpdatedAt: room.UpdatedAt,
			Status:    room.Status,
			Avatar:    room.Avatar,
			People:    room.People,
			Parents: dto.Parents{
				Id:   room.Parent.ID,
				Name: room.Parent.Name,
			},
		})
	}

	response.SuccessWithPagination(c, roomResponses, page, limit, total)
}

func isRoomMatch(room models.Room, typeFilter, statusFilter, nameFilter, accommodationFilter, accommodationIdFilter, numBedFilter, numToletFilter, peopleFilter string, fromDate, toDate time.Time, statusMap map[uint]bool) bool {
	if _, exists := statusMap[uint(room.RoomId)]; exists {
		return false
	}

	// Kiểm tra lọc theo typeFilter
	if typeFilter != "" {
		parsedTypeFilter, err := strconv.Atoi(typeFilter)
		if err == nil && room.Type != uint(parsedTypeFilter) {
			return false
		}
	}

	// Kiểm tra lọc theo statusFilter
	if statusFilter != "" {
		parsedStatus, _ := strconv.Atoi(statusFilter)
		if room.Status != parsedStatus {
			return false
		}
	}

	// Kiểm tra lọc theo nameFilter (chuỗi gần đúng)
	if nameFilter != "" {
		decodedNameFilter, _ := url.QueryUnescape(nameFilter)
		if !strings.Contains(strings.ToLower(room.RoomName), strings.ToLower(decodedNameFilter)) {
			return false
		}
	}

	// Kiểm tra lọc theo accommodationFilter
	if accommodationFilter != "" {
		decodedAccommodationFilter, _ := url.QueryUnescape(accommodationFilter)
		if !strings.Contains(strings.ToLower(room.Parent.Name), strings.ToLower(decodedAccommodationFilter)) {
			return false
		}
	}

	// Kiểm tra lọc theo accommodationIdFilter
	if accommodationIdFilter != "" {
		parsedAccommodationId, err := strconv.Atoi(accommodationIdFilter)
		if err == nil {
			if room.AccommodationID != uint(parsedAccommodationId) && room.Parent.ID != uint(parsedAccommodationId) {
				return false
			}
		}
	}

	// Kiểm tra lọc theo numBedFilter
	if numBedFilter != "" {
		numBed, _ := strconv.Atoi(numBedFilter)
		if room.NumBed != numBed {
			return false
		}
	}

	// Kiểm tra lọc theo numToletFilter
	if numToletFilter != "" {
		numTolet, _ := strconv.Atoi(numToletFilter)
		if room.NumTolet != numTolet {
			return false
		}
	}

	// Kiểm tra lọc theo peopleFilter
	if peopleFilter != "" {
		people, _ := strconv.Atoi(peopleFilter)
		if room.People != people {
			return false
		}
	}

	return true
}

func GetAllRoomsUser(c *gin.Context) {
	var totalRooms int64

	pageStr := c.Query("page")
	limitStr := c.Query("limit")
	typeFilter := c.Query("type")
	statusFilter := c.Query("status")
	nameFilter := c.Query("name")
	accommodationFilter := c.Query("accommodation")
	accommodationIdFilter := c.Query("accommodationId")
	numBedFilter := c.Query("numBed")
	numToletFilter := c.Query("numTolet")
	peopleFilter := c.Query("people")

	fromDateStr := c.Query("fromDate")
	toDateStr := c.Query("toDate")

	limit := 10
	page := 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p >= 0 {
			page = p
		}
	}

	var fromDate, toDate time.Time
	var err error

	if fromDateStr != "" {
		fromDate, err = time.Parse("02/01/2006", fromDateStr)
		if err != nil {
			response.BadRequest(c, "Dữ liệu fromDate không hợp lệ")
			return
		}
	}

	if toDateStr != "" {
		toDate, err = time.Parse("02/01/2006", toDateStr)
		if err != nil {
			response.BadRequest(c, "Dữ liệu toDate không hợp lệ")
			return
		}
	}

	statuses, err := getRoomStatuses(fromDate, toDate)
	if err != nil {
		response.ServerError(c)
		return
	}

	statusMap := make(map[uint]bool)
	for _, status := range statuses {
		statusMap[status.RoomID] = true
	}

	offset := page * limit

	// Kết nối Redis
	rdb, err := config.ConnectRedis()
	if err != nil {
		response.ServerError(c)
		return
	}

	cacheKey := "rooms:all"
	var allRooms []models.Room

	// Lấy dữ liệu từ Redis
	if err := services.GetFromRedis(config.Ctx, rdb, cacheKey, &allRooms); err != nil || len(allRooms) == 0 {
		// Nếu Redis không có dữ liệu, lấy từ DB
		if err := config.DB.Model(&models.Room{}).Preload("Parent").Find(&allRooms).Error; err != nil {
			response.ServerError(c)
			return
		}

		var allRoomsDetails []dto.RoomDetail
		for _, room := range allRooms {
			roomDetail := dto.RoomDetail{
				RoomId:   room.RoomId,
				RoomName: room.RoomName,
				Type:     room.Type,
				NumBed:   room.NumBed,
				NumTolet: room.NumTolet,
				Acreage:  room.Acreage,
				Price:    room.Price,

				CreatedAt: room.CreatedAt,
				UpdatedAt: room.UpdatedAt,
				Status:    room.Status,
				Avatar:    room.Avatar,
				People:    room.People,
				Parent: dto.Parents{
					Id:   room.Parent.ID,
					Name: room.Parent.Name,
				},
				Img:       room.Img,
				Furniture: room.Furniture,
			}
			allRoomsDetails = append(allRoomsDetails, roomDetail)
		}

		// Lưu dữ liệu vào Redis
		if err := services.SetToRedis(config.Ctx, rdb, cacheKey, allRoomsDetails, 10*time.Minute); err != nil {
			log.Printf("Lỗi khi lưu danh sách phòng vào Redis: %v", err)
		}
	}

	// Áp dụng filter trên dữ liệu từ Redis

	filteredRooms := make([]models.Room, 0)
	for _, room := range allRooms {
		if isRoomMatch(room, typeFilter, statusFilter, nameFilter, accommodationFilter, accommodationIdFilter, numBedFilter, numToletFilter, peopleFilter, fromDate, toDate, statusMap) {
			filteredRooms = append(filteredRooms, room)
		}
	}

	// Đếm tổng số phòng sau khi lọc
	totalRooms = int64(len(filteredRooms))

	// Phân trang
	startIndex := offset
	endIndex := offset + limit
	if startIndex > len(filteredRooms) {
		startIndex = len(filteredRooms)
	}
	if endIndex > len(filteredRooms) {
		endIndex = len(filteredRooms)
	}
	paginatedRooms := filteredRooms[startIndex:endIndex]

	roomResponses := make([]dto.RoomResponse, 0)
	for _, room := range paginatedRooms {
		roomResponses = append(roomResponses, dto.RoomResponse{
			RoomId:    room.RoomId,
			RoomName:  room.RoomName,
			Type:      room.Type,
			NumBed:    room.NumBed,
			NumTolet:  room.NumTolet,
			Acreage:   room.Acreage,
			Price:     room.Price,
			CreatedAt: room.CreatedAt,
			UpdatedAt: room.UpdatedAt,
			Status:    room.Status,
			Avatar:    room.Avatar,
			People:    room.People,
			Parents: dto.Parents{
				Id:   room.Parent.ID,
				Name: room.Parent.Name,
			},
		})
	}

	response.SuccessWithPagination(c, roomResponses, page, limit, int(totalRooms))
}

// cập nhật giá phòng thấp nhất cho price của accommodation
func UpdateLowestPriceForAccommodation(accommodationID uint) {
	var lowestPrice int
	if err := config.DB.Model(&models.Room{}).
		Where("accommodation_id = ?", accommodationID).
		Order("price ASC").
		Limit(1).
		Pluck("price", &lowestPrice).Error; err != nil {
		fmt.Printf("Lỗi khi lấy giá phòng thấp nhất cho accommodation ID %d: %v\n", accommodationID, err)
		return
	}

	if lowestPrice > 0 {
		if err := config.DB.Model(&models.Accommodation{}).
			Where("id = ?", accommodationID).
			Update("price", lowestPrice).Error; err != nil {
			fmt.Printf("Lỗi khi cập nhật giá thấp nhất cho accommodation ID %d: %v\n", accommodationID, err)
		} else {
			fmt.Printf("Đã cập nhật giá thấp nhất cho accommodation ID %d: %d\n", accommodationID, lowestPrice)
		}
	}
}

func CreateRoom(c *gin.Context) {
	var newRoom models.Room
	// Xác thực token
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
	if err := c.ShouldBindJSON(&newRoom); err != nil {
		response.BadRequest(c, "Dữ liệu không hợp lệ")
		return
	}

	if err := newRoom.ValidateStatus(); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	furnitureJSON, err := json.Marshal(newRoom.Furniture)
	if err != nil {
		response.ServerError(c)
		return
	}

	imgJSON, err := json.Marshal(newRoom.Img)
	if err != nil {
		response.ServerError(c)
		return
	}
	var accommodation models.Accommodation
	if err := config.DB.First(&accommodation, newRoom.AccommodationID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c)
			return
		}
		response.ServerError(c)
		return
	}
	newRoom.Parent = accommodation
	newRoom.Img = imgJSON
	newRoom.Furniture = furnitureJSON

	if err := config.DB.Create(&newRoom).Error; err != nil {
		response.ServerError(c)
		return
	}

	// Gọi hàm cập nhật giá thấp nhất
	go UpdateLowestPriceForAccommodation(newRoom.AccommodationID)
	//Xóa redis
	rdb, redisErr := config.ConnectRedis()
	if redisErr == nil {
		switch currentUserRole {
		case 1: // Super Admin
			_ = services.DeleteFromRedis(config.Ctx, rdb, "rooms:all")
			_ = services.DeleteFromRedis(config.Ctx, rdb, CacheKey2)
		case 2: // Admin
			adminCacheKey := fmt.Sprintf("rooms:admin:%d", currentUserID)
			accommodationsCacheKey := fmt.Sprintf("accommodations:admin:%d", currentUserID)
			_ = services.DeleteFromRedis(config.Ctx, rdb, accommodationsCacheKey)
			_ = services.DeleteFromRedis(config.Ctx, rdb, adminCacheKey)
			_ = services.DeleteFromRedis(config.Ctx, rdb, CacheKey2)
			var receptionistIDs []int
			if err := config.DB.Model(&models.User{}).Where("admin_id = ?", currentUserID).Pluck("id", &receptionistIDs).Error; err == nil {
				for _, receptionistID := range receptionistIDs {
					receptionistCacheKey := fmt.Sprintf("rooms:receptionist:%d", receptionistID)
					_ = services.DeleteFromRedis(config.Ctx, rdb, receptionistCacheKey)
					_ = services.DeleteFromRedis(config.Ctx, rdb, CacheKey2)
				}
			}
		case 3: // Receptionist
			var adminID int
			if err := config.DB.Model(&models.User{}).Select("admin_id").Where("id = ?", currentUserID).Scan(&adminID).Error; err == nil {
				adminCacheKey := fmt.Sprintf("rooms:admin:%d", adminID)
				receptionistCacheKey := fmt.Sprintf("rooms:receptionist:%d", currentUserID)
				_ = services.DeleteFromRedis(config.Ctx, rdb, adminCacheKey)
				_ = services.DeleteFromRedis(config.Ctx, rdb, receptionistCacheKey)
				_ = services.DeleteFromRedis(config.Ctx, rdb, CacheKey2)
			}
		}
	}

	response.Success(c, newRoom)
}

func GetRoomDetail(c *gin.Context) {
	roomId := c.Param("id")

	// Kết nối Redis
	rdb, redisErr := config.ConnectRedis()
	if redisErr != nil {
		response.ServerError(c)
		return
	}

	// Key cache cho tất cả rooms
	cacheKey := "rooms:all"

	// Lấy danh sách rooms từ cache
	var cachedRooms []models.Room
	if err := services.GetFromRedis(config.Ctx, rdb, cacheKey, &cachedRooms); err == nil {
		// Tìm room theo ID trong cache
		for _, room := range cachedRooms {
			if fmt.Sprintf("%d", room.RoomId) == roomId {
				response.Success(c, buildRoomDetailResponse(room))
				return
			}
		}
	}

	// Nếu không tìm thấy trong cache, truy vấn từ database
	var room models.Room
	if err := config.DB.Preload("Parent").First(&room, roomId).Error; err != nil {
		response.NotFound(c)
		return
	}

	response.Success(c, buildRoomDetailResponse(room))
}

func UpdateRoom(c *gin.Context) {
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

	var request dto.RoomRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "Dữ liệu không hợp lệ")
		return
	}

	var room models.Room

	if err := config.DB.First(&room, request.RoomId).Error; err != nil {
		response.NotFound(c)
		return
	}

	if err := room.ValidateStatus(); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	imgJSON, err := json.Marshal(request.Img)
	if err != nil {
		response.ServerError(c)
		return
	}

	furnitureJSON, err := json.Marshal(request.Furniture)
	if err != nil {
		response.ServerError(c)
		return
	}

	if request.RoomName != "" {
		room.RoomName = request.RoomName
	}

	if request.Type > 0 {
		room.Type = request.Type
	}

	if request.NumBed != 0 {
		room.NumBed = request.NumBed
	}

	if request.NumTolet != 0 {
		room.NumTolet = request.NumTolet
	}

	if request.Acreage != 0 {
		room.Acreage = request.Acreage
	}

	if request.Price != 0 {
		room.Price = request.Price
	}

	if request.Description != "" {
		room.Description = request.Description
	}

	if request.Status != 0 {
		room.Status = request.Status
	}

	if request.Avatar != "" {
		room.Avatar = request.Avatar
	}

	if len(request.Img) > 0 {
		room.Img = imgJSON
	}

	if len(request.Furniture) > 0 {
		room.Furniture = furnitureJSON
	}

	if err := config.DB.Save(&room).Error; err != nil {
		response.ServerError(c)
		return
	}

	//cập nhật price của accommodation
	go UpdateLowestPriceForAccommodation(room.AccommodationID)

	//Xóa redis
	rdb, redisErr := config.ConnectRedis()
	if redisErr == nil {
		switch currentUserRole {
		case 1: // Super Admin
			_ = services.DeleteFromRedis(config.Ctx, rdb, "rooms:all")
			_ = services.DeleteFromRedis(config.Ctx, rdb, CacheKey2)
		case 2: // Admin
			adminCacheKey := fmt.Sprintf("rooms:admin:%d", currentUserID)
			accommodationsCacheKey := fmt.Sprintf("accommodations:admin:%d", currentUserID)
			_ = services.DeleteFromRedis(config.Ctx, rdb, accommodationsCacheKey)
			_ = services.DeleteFromRedis(config.Ctx, rdb, adminCacheKey)
			_ = services.DeleteFromRedis(config.Ctx, rdb, CacheKey2)
			_ = services.DeleteFromRedis(config.Ctx, rdb, "rooms:all")

			var receptionistIDs []int
			if err := config.DB.Model(&models.User{}).Where("admin_id = ?", currentUserID).Pluck("id", &receptionistIDs).Error; err == nil {
				for _, receptionistID := range receptionistIDs {
					receptionistCacheKey := fmt.Sprintf("rooms:receptionist:%d", receptionistID)
					_ = services.DeleteFromRedis(config.Ctx, rdb, receptionistCacheKey)
				}
			}
		case 3: // Receptionist
			var adminID int
			if err := config.DB.Model(&models.User{}).Select("admin_id").Where("id = ?", currentUserID).Scan(&adminID).Error; err == nil {
				adminCacheKey := fmt.Sprintf("rooms:admin:%d", adminID)
				_ = services.DeleteFromRedis(config.Ctx, rdb, "rooms:all")
				receptionistCacheKey := fmt.Sprintf("rooms:receptionist:%d", currentUserID)
				_ = services.DeleteFromRedis(config.Ctx, rdb, adminCacheKey)
				_ = services.DeleteFromRedis(config.Ctx, rdb, receptionistCacheKey)
				_ = services.DeleteFromRedis(config.Ctx, rdb, CacheKey2)
			}
		}
	}

	response.Success(c, room)
}

func ChangeRoomStatus(c *gin.Context) {
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
		RoomId uint `json:"id"`
		Status int  `json:"status"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "Dữ liệu không hợp lệ")
		return
	}

	var room models.Room

	if err := config.DB.First(&room, input.RoomId).Error; err != nil {
		response.NotFound(c)
		return
	}

	room.Status = input.Status
	if err := config.DB.Save(&room).Error; err != nil {
		response.ServerError(c)
		return
	}

	//Xóa redis
	rdb, redisErr := config.ConnectRedis()
	if redisErr == nil {
		switch currentUserRole {
		case 1: // Super Admin
			_ = services.DeleteFromRedis(config.Ctx, rdb, "rooms:all")
		case 2: // Admin
			adminCacheKey := fmt.Sprintf("rooms:admin:%d", currentUserID)
			_ = services.DeleteFromRedis(config.Ctx, rdb, adminCacheKey)
			_ = services.DeleteFromRedis(config.Ctx, rdb, CacheKey2)
			_ = services.DeleteFromRedis(config.Ctx, rdb, "rooms:all")
			var receptionistIDs []int
			if err := config.DB.Model(&models.User{}).Where("admin_id = ?", currentUserID).Pluck("id", &receptionistIDs).Error; err == nil {
				for _, receptionistID := range receptionistIDs {
					receptionistCacheKey := fmt.Sprintf("rooms:receptionist:%d", receptionistID)
					_ = services.DeleteFromRedis(config.Ctx, rdb, receptionistCacheKey)
				}
			}
		case 3: // Receptionist
			var adminID int
			if err := config.DB.Model(&models.User{}).Select("admin_id").Where("id = ?", currentUserID).Scan(&adminID).Error; err == nil {
				adminCacheKey := fmt.Sprintf("rooms:admin:%d", adminID)
				receptionistCacheKey := fmt.Sprintf("rooms:receptionist:%d", currentUserID)
				_ = services.DeleteFromRedis(config.Ctx, rdb, adminCacheKey)
				_ = services.DeleteFromRedis(config.Ctx, rdb, receptionistCacheKey)
				_ = services.DeleteFromRedis(config.Ctx, rdb, CacheKey2)
			}
		}
	}

	response.Success(c, room)
}

// Hàm set response cho details
func buildRoomDetailResponse(room models.Room) dto.RoomDetail {
	return dto.RoomDetail{
		RoomId:      room.RoomId,
		RoomName:    room.RoomName,
		Type:        room.Type,
		NumBed:      room.NumBed,
		NumTolet:    room.NumTolet,
		Acreage:     room.Acreage,
		Price:       room.Price,
		Description: room.Description,
		CreatedAt:   room.CreatedAt,
		UpdatedAt:   room.UpdatedAt,
		Status:      room.Status,
		Avatar:      room.Avatar,
		Img:         room.Img,
		Num:         room.Num,
		Furniture:   room.Furniture,
		People:      room.People,
		Parent: dto.Parents{
			Id:   room.Parent.ID,
			Name: room.Parent.Name,
		},
	}
}
