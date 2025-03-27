package controllers

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"new/config"
	"new/dto"
	"new/models"
	"new/response"
	"new/services"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func convertToOrderAccommodationResponse(accommodation models.Accommodation) dto.OrderAccommodationResponse {
	return dto.OrderAccommodationResponse{
		ID:       accommodation.ID,
		Type:     accommodation.Type,
		Name:     accommodation.Name,
		Address:  accommodation.Address,
		Ward:     accommodation.Ward,
		District: accommodation.District,
		Province: accommodation.Province,
		Price:    accommodation.Price,
		Avatar:   accommodation.Avatar,
	}
}

func convertToOrderRoomResponse(room models.Room) dto.OrderRoomResponse {
	return dto.OrderRoomResponse{
		ID:              room.RoomId,
		AccommodationID: room.AccommodationID,
		RoomName:        room.RoomName,
		Price:           room.Price,
	}
}

// Chuyển chuỗi ngày string thành dạng timestamp
func ConvertDateToISOFormat(dateStr string) (time.Time, error) {
	parsedDate, err := time.Parse("02/01/2006", dateStr)
	if err != nil {
		return time.Time{}, err
	}
	return parsedDate, nil
}

func GetOrders(c *gin.Context) {
	// Lấy Authorization Header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		response.Unauthorized(c)
		return
	}

	// Xử lý token
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	currentUserID, currentUserRole, err := GetUserIDFromToken(tokenString)
	if err != nil {
		response.Unauthorized(c)
		return
	}

	// Kết nối Redis
	cacheKey := fmt.Sprintf("orders:all:user:%d", currentUserID)
	rdb, err := config.ConnectRedis()
	if err != nil {
		response.ServerError(c)

	}

	var allOrders []models.Order

	// Lấy dữ liệu từ Redis Cache
	if err := services.GetFromRedis(config.Ctx, rdb, cacheKey, &allOrders); err != nil || len(allOrders) == 0 {
		// Nếu không có cache hoặc Redis gặp lỗi, thực hiện truy vấn từ DB
		baseTx := config.DB.Model(&models.Order{}).
			Preload("Accommodation").
			Preload("Room").
			Preload("User")

		// Áp dụng quyền truy cập
		if currentUserRole == 2 {
			// Admin: Lọc theo các chỗ ở thuộc về Admin
			baseTx = baseTx.Where("orders.accommodation_id IN (?)",
				config.DB.Model(&models.Accommodation{}).Select("id").Where("user_id = ?", currentUserID))
		} else if currentUserRole == 3 {
			// Receptionist: Lọc theo các chỗ ở thuộc về Admin của Receptionist
			var adminID int
			if err := config.DB.Model(&models.User{}).Select("admin_id").Where("id = ?", currentUserID).Scan(&adminID).Error; err != nil || adminID == 0 {
				response.Forbidden(c)
				return
			}
			baseTx = baseTx.Where("orders.accommodation_id IN (?)",
				config.DB.Model(&models.Accommodation{}).Select("id").Where("user_id = ?", adminID))
		}

		// Truy vấn tất cả đơn hàng từ DB
		if err := baseTx.Find(&allOrders).Error; err != nil {
			response.ServerError(c)
			return
		}

		// Lưu vào Redis Cache
		if err := services.SetToRedis(config.Ctx, rdb, cacheKey, allOrders, 10*time.Minute); err != nil {
			log.Printf("Lỗi khi lưu danh sách đơn hàng vào Redis: %v", err)
		}
	}

	// Lấy các tham số filter từ query
	pageStr := c.Query("page")
	limitStr := c.Query("limit")
	nameFilter := c.Query("name")
	phoneStr := c.Query("phoneNumber")
	priceStr := c.Query("price")
	fromDateStr := c.Query("fromDate")
	toDateStr := c.Query("toDate")
	statusFilter := c.Query("status")

	// Xử lý phân trang
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

	// Áp dụng bộ lọc
	filteredOrders := make([]models.Order, 0)
	for _, order := range allOrders {
		if nameFilter != "" {
			decodedName, _ := url.QueryUnescape(nameFilter)
			if !strings.Contains(strings.ToLower(order.Accommodation.Name), strings.ToLower(decodedName)) {
				continue
			}
		}
		if phoneStr != "" {
			if order.User != nil && !strings.Contains(strings.ToLower(order.User.PhoneNumber), strings.ToLower(phoneStr)) {
				continue
			}
			if order.User == nil && !strings.Contains(strings.ToLower(order.GuestPhone), strings.ToLower(phoneStr)) {
				continue
			}
		}
		if priceStr != "" {
			price, err := strconv.ParseFloat(priceStr, 64)
			if err == nil && order.TotalPrice < price {
				continue
			}
		}
		if fromDateStr != "" {
			fromDateISO, err := ConvertDateToISOFormat(fromDateStr)
			if err != nil {
				response.BadRequest(c, "Sai định dạng fromDate")
				return
			}
			if order.CreatedAt.Before(fromDateISO) {
				continue
			}
		}
		if toDateStr != "" {
			toDateISO, err := ConvertDateToISOFormat(toDateStr)
			if err != nil {
				response.BadRequest(c, "Sai định dạng toDate")
				return
			}
			if order.UpdatedAt.After(toDateISO) {
				continue
			}
		}
		if statusFilter != "" {
			parsedStatusFilter, err := strconv.Atoi(statusFilter)
			if err == nil && order.Status != parsedStatusFilter {
				continue
			}
		}
		filteredOrders = append(filteredOrders, order)
	}

	// Tính toán lại tổng số đơn hàng sau khi lọc
	totalFiltered := len(filteredOrders)

	//Xếp theo update mới nhất
	sort.Slice(filteredOrders, func(i, j int) bool {
		return filteredOrders[i].UpdatedAt.After(filteredOrders[j].UpdatedAt)
	})
	// Áp dụng phân trang
	start := page * limit
	end := start + limit
	if start >= totalFiltered {
		filteredOrders = []models.Order{}
	} else if end > totalFiltered {
		filteredOrders = filteredOrders[start:]
	} else {
		filteredOrders = filteredOrders[start:end]
	}

	// Chuẩn bị phản hồi
	var orderResponses []dto.OrderUserResponse
	for _, order := range filteredOrders {
		var user dto.ActorResponse
		if order.UserID != nil {
			user = dto.ActorResponse{Name: order.User.Name, Email: order.User.Email, PhoneNumber: order.User.PhoneNumber}
		} else {
			user = dto.ActorResponse{Name: order.GuestName, Email: order.GuestEmail, PhoneNumber: order.GuestPhone}
		}

		accommodationResponse := convertToOrderAccommodationResponse(order.Accommodation)
		var roomResponses []dto.OrderRoomResponse
		for _, room := range order.Room {
			roomResponse := convertToOrderRoomResponse(room)
			roomResponses = append(roomResponses, roomResponse)
		}

		orderResponse := dto.OrderUserResponse{
			ID:               order.ID,
			User:             user,
			Accommodation:    accommodationResponse,
			Room:             roomResponses,
			CheckInDate:      order.CheckInDate,
			CheckOutDate:     order.CheckOutDate,
			Status:           order.Status,
			CreatedAt:        order.CreatedAt,
			UpdatedAt:        order.UpdatedAt,
			Price:            order.Price,
			HolidayPrice:     order.HolidayPrice,
			CheckInRushPrice: order.CheckInRushPrice,
			SoldOutPrice:     order.SoldOutPrice,
			DiscountPrice:    order.DiscountPrice,
			TotalPrice:       order.TotalPrice,
		}
		orderResponses = append(orderResponses, orderResponse)
	}

	response.SuccessWithPagination(c, orderResponses, page, limit, totalFiltered)
}

func CreateOrder(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")

	var request dto.CreateOrderRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "Dữ liệu không hợp lệ")
		return
	}

	var currentUserID uint
	var userId *uint
	var actor dto.ActorResponse

	if authHeader != "" {
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		userID, _, err := GetUserIDFromToken(tokenString)

		if err == nil {
			currentUserID = userID
		} else {
			response.Unauthorized(c)
			return
		}
	} else {
		if request.UserID != 0 {
			currentUserID = request.UserID
			var userInfo models.User
			if err := config.DB.Where("id = ?", request.UserID).First(&userInfo).Error; err == nil {
				userId = new(uint)
				*userId = userInfo.ID

				actor = dto.ActorResponse{
					Name:        userInfo.Name,
					Email:       userInfo.Email,
					PhoneNumber: userInfo.PhoneNumber,
				}
			} else {
				response.NotFound(c)
				return
			}
		} else {
			var userInfo models.User
			if err := config.DB.Where("phone_number = ?", request.GuestPhone).First(&userInfo).Error; err == nil {

				userId = new(uint)
				*userId = userInfo.ID

				actor = dto.ActorResponse{
					Name:        userInfo.Name,
					Email:       userInfo.Email,
					PhoneNumber: userInfo.PhoneNumber,
				}
			} else {
				userId = nil
				actor = dto.ActorResponse{
					Name:        request.GuestName,
					Email:       request.GuestEmail,
					PhoneNumber: request.GuestPhone,
				}
			}
		}
	}

	checkInDate, err := time.Parse("02/01/2006", request.CheckInDate)
	if err != nil {
		response.BadRequest(c, "Ngày nhận phòng không hợp lệ")
		return
	}

	if checkInDate.Before(time.Now()) {
		response.BadRequest(c, "Ngày nhận phòng không được nhỏ hơn ngày hiện tại")
		return
	}

	checkOutDate, err := time.Parse("02/01/2006", request.CheckOutDate)
	if err != nil {
		response.BadRequest(c, "Ngày trả phòng không hợp lệ")
		return
	}

	order := models.Order{
		UserID:          userId,
		AccommodationID: request.AccommodationID,
		RoomID:          request.RoomID,
		CheckInDate:     request.CheckInDate,
		CheckOutDate:    request.CheckOutDate,
		GuestName:       request.GuestName,
		GuestEmail:      request.GuestEmail,
		GuestPhone:      request.GuestPhone,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	numDays := int(checkOutDate.Sub(checkInDate).Hours() / 24)
	if numDays <= 0 {
		response.BadRequest(c, "Ngày trả phòng phải sau ngày nhận phòng")
		return
	}

	price := 0
	soldOutPrice := 0.0

	var accommodation models.Accommodation
	if err := config.DB.First(&accommodation, request.AccommodationID).Error; err != nil {
		response.ServerError(c)
		return
	}

	if accommodation.Type == 0 && len(order.RoomID) > 0 {
		var rooms []models.Room
		if err := config.DB.Where("room_id IN ?", order.RoomID).Find(&rooms).Error; err != nil || len(rooms) != len(order.RoomID) {
			response.ServerError(c)
			return
		}

		for _, room := range rooms {
			if room.AccommodationID != request.AccommodationID {
				response.BadRequest(c, "AccommodationID không hợp lệ")
				return
			}

			var roomStatus []models.RoomStatus
			err := config.DB.Where("room_id = ? AND status = 1 AND ((from_date < ? AND to_date > ?) OR (from_date < ? AND to_date > ?))",
				room.RoomId, checkOutDate, checkInDate, checkOutDate, checkInDate).Find(&roomStatus).Error

			if err != nil {
				response.ServerError(c)
				return
			}

			if len(roomStatus) > 0 {
				response.BadRequest(c, "Phòng đã được đặt hoặc không khả dụng trong khoảng thời gian này")
				return
			}
			price += room.Price * numDays

		}
	} else {

		var accommodationStatus []models.AccommodationStatus
		if err := config.DB.Where("accommodation_id = ? AND status = 1 AND ((from_date < ? AND to_date > ?) OR (from_date < ? AND to_date > ?))",
			request.AccommodationID, checkOutDate, checkInDate, checkOutDate, checkInDate).Find(&accommodationStatus).Error; err != nil {
			response.ServerError(c)
			return
		}

		if len(accommodationStatus) > 0 {
			response.BadRequest(c, "Chỗ ở đã được đặt hoặc không khả dụng trong khoảng thời gian này")
			return
		}

		price = accommodation.Price * numDays
	}

	order.Price = price
	order.SoldOutPrice = soldOutPrice

	if request.UserID != 0 {
		order.UserID = &request.UserID
		isEligibleForDiscount := services.CheckUserEligibilityForDiscount(request.UserID)
		if isEligibleForDiscount {
			var user models.User
			if err := config.DB.First(&user, request.UserID).Error; err == nil {
				discountPrice, err := services.ApplyDiscountForUser(user)
				if err != nil {
					fmt.Println(err.Error())
					return
				}
				order.DiscountPrice = float64(price) * discountPrice / 100
			} else {
				fmt.Println("Không tìm thấy người dùng")
				return
			}
		}
	}

	var holidays []models.Holiday
	if err := config.DB.Find(&holidays).Error; err != nil {
		response.ServerError(c)
		return
	}

	holidayPrice := 0
	for _, holiday := range holidays {
		fromDate, err := time.Parse("02/01/2006", holiday.FromDate)
		if err != nil {
			response.ServerError(c)
			return
		}

		toDate, err := time.Parse("02/01/2006", holiday.ToDate)
		if err != nil {
			response.ServerError(c)
			return
		}

		if (checkInDate.Before(toDate) && checkOutDate.After(fromDate)) ||
			checkInDate.Equal(fromDate) ||
			checkOutDate.Equal(toDate) {
			holidayPrice += holiday.Price
		}
	}
	order.HolidayPrice = float64(price*holidayPrice) / 100

	numDaysToCheckIn := int(checkInDate.Sub(order.CreatedAt).Hours() / 24)

	if numDaysToCheckIn <= 3 {
		order.CheckInRushPrice = float64(price*5) / 100
	} else {
		order.CheckInRushPrice = 0
	}

	order.TotalPrice = float64(price) + order.HolidayPrice + order.CheckInRushPrice + order.SoldOutPrice - order.DiscountPrice

	if len(request.RoomID) > 0 {
		order.RoomID = request.RoomID
	} else {
		order.RoomID = []uint{}
	}

	if err := config.DB.Create(&order).Error; err != nil {
		response.ServerError(c)
		return
	}

	if accommodation.Type == 0 && len(order.RoomID) > 0 {
		var roomsToAppend []models.Room
		for _, roomID := range request.RoomID {
			roomsToAppend = append(roomsToAppend, models.Room{RoomId: roomID})
		}

		if err := config.DB.Model(&order).Association("Room").Append(roomsToAppend); err != nil {
			response.ServerError(c)
			return
		}

		for _, roomID := range request.RoomID {
			roomStatus := models.RoomStatus{
				RoomID:   roomID,
				Status:   1,
				FromDate: checkInDate,
				ToDate:   checkOutDate,
			}
			if err := config.DB.Create(&roomStatus).Error; err != nil {
				response.ServerError(c)
				return
			}
		}
	} else {
		roomStatus := models.AccommodationStatus{
			AccommodationID: request.AccommodationID,
			Status:          1,
			FromDate:        checkInDate,
			ToDate:          checkOutDate,
		}
		if err := config.DB.Create(&roomStatus).Error; err != nil {
			response.ServerError(c)
			return
		}
	}

	if err := config.DB.Preload("User").Preload("Accommodation").Preload("Room").First(&order, order.ID).Error; err != nil {
		response.ServerError(c)
		return
	}

	var user dto.ActorResponse
	if order.UserID != nil {
		user = actor
	} else {
		user = actor
	}

	accommodationResponse := convertToOrderAccommodationResponse(order.Accommodation)
	var roomResponses []dto.OrderRoomResponse
	if len(request.RoomID) > 0 {
		for _, room := range order.Room {
			roomResponse := convertToOrderRoomResponse(room)
			roomResponses = append(roomResponses, roomResponse)
		}
	}

	orderResponse := dto.OrderUserResponse{
		ID:               order.ID,
		User:             user,
		Accommodation:    accommodationResponse,
		Room:             roomResponses,
		CheckInDate:      order.CheckInDate,
		CheckOutDate:     order.CheckOutDate,
		Status:           order.Status,
		CreatedAt:        order.CreatedAt,
		UpdatedAt:        order.UpdatedAt,
		Price:            price,
		HolidayPrice:     order.HolidayPrice,
		CheckInRushPrice: order.CheckInRushPrice,
		SoldOutPrice:     order.SoldOutPrice,
		DiscountPrice:    order.DiscountPrice,
		TotalPrice:       order.TotalPrice,
	}

	if orderResponse.User.Email != "" {
		if err := services.SendOrderEmail(orderResponse.User.Email, order.ID, order.TotalPrice, order.CheckInDate, order.CheckOutDate); err != nil {
			fmt.Println("Gửi email không thành công:", err)
		}
	}

	//Xóa redis
	rdb, redisErr := config.ConnectRedis()
	if redisErr == nil {
		cacheKey := "orders:all"
		cacheKeyUser := fmt.Sprintf("orders:all:user:%d", currentUserID)

		_ = services.DeleteFromRedis(config.Ctx, rdb, cacheKey)
		_ = services.DeleteFromRedis(config.Ctx, rdb, "invoices:all")
		_ = services.DeleteFromRedis(config.Ctx, rdb, "accommodations:statuses")
		_ = services.DeleteFromRedis(config.Ctx, rdb, "rooms:statuses")
		_ = services.DeleteFromRedis(config.Ctx, rdb, cacheKeyUser)
	}

	response.Success(c, orderResponse)
}

func ChangeOrderStatus(c *gin.Context) {
	type StatusUpdateRequest struct {
		ID         uint    `json:"id"`
		Status     int     `json:"status"`
		PaidAmount float64 `json:"paidAmount"`
	}

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

	var req StatusUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Dữ liệu không hợp lệ")
		return
	}

	var order models.Order
	if err := config.DB.
		Preload("Accommodation.User").
		First(&order, req.ID).Error; err != nil {
		response.NotFound(c)
		return
	}

	timeSinceCreation := time.Since(order.CreatedAt).Hours()

	if currentUserRole == 0 && req.Status == 2 {
		if timeSinceCreation > 24 {
			response.BadRequest(c, "Liên hệ Admin để được hủy đơn")
			return
		}
	}

	if req.Status == 2 {
		if order.Status == 1 {
			var invoice models.Invoice
			if err := config.DB.Where("order_id = ?", order.ID).First(&invoice).Error; err == nil {
				// Xóa invoice
				if err := config.DB.Delete(&invoice).Error; err != nil {
					response.ServerError(c)
					return
				}

				today := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Local)

				var userRevenue models.UserRevenue
				if err := config.DB.Where("user_id = ? AND date = ?", invoice.AdminID, today).First(&userRevenue).Error; err == nil {
					newRevenue := userRevenue.Revenue - invoice.TotalAmount
					newOrderCount := userRevenue.OrderCount - 1
					if newOrderCount < 0 {
						newOrderCount = 0
					}

					if err := config.DB.Model(&userRevenue).Updates(map[string]interface{}{
						"revenue":     newRevenue,
						"order_count": newOrderCount,
					}).Error; err != nil {
						response.ServerError(c)
						return
					}
				} else {
					response.NotFound(c)
					return
				}
			} else {
				response.NotFound(c)
				return
			}
		}

		if len(order.RoomID) > 0 {
			for _, room := range order.Room {
				var roomStatus models.RoomStatus
				if err := config.DB.Where("room_id = ? AND status = ?", room.RoomId, 1).First(&roomStatus).Error; err == nil {
					roomStatus.Status = 0
					if err := config.DB.Save(&roomStatus).Error; err != nil {
						response.ServerError(c)
						return
					}
				}
			}
		} else {
			var accommodationStatus models.AccommodationStatus
			if err := config.DB.Where("accommodation_id = ? AND status = ?", order.AccommodationID, 1).First(&accommodationStatus).Error; err == nil {
				accommodationStatus.Status = 0
				if err := config.DB.Save(&accommodationStatus).Error; err != nil {
					response.ServerError(c)
					return
				}
			}
		}
	}

	if req.Status == 1 {
		var existingInvoice models.Invoice
		if err := config.DB.Where("order_id = ?", order.ID).First(&existingInvoice).Error; err == nil {
			response.Conflict(c)
			return
		}

		var Remaining = order.TotalPrice - req.PaidAmount

		invoice := models.Invoice{
			OrderID:         order.ID,
			TotalAmount:     order.TotalPrice,
			PaidAmount:      req.PaidAmount,
			RemainingAmount: Remaining,
			AdminID:         order.Accommodation.UserID,
		}

		if err := config.DB.Create(&invoice).Error; err != nil {
			response.ServerError(c)
			return
		}

		today := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Local)

		var userRevenue models.UserRevenue
		if err := config.DB.Where("user_id = ? AND date = ?", invoice.AdminID, today).First(&userRevenue).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {

				newUserRevenue := models.UserRevenue{
					UserID:     invoice.AdminID,
					Date:       today,
					Revenue:    invoice.TotalAmount,
					OrderCount: 1,
				}
				if err := config.DB.Create(&newUserRevenue).Error; err != nil {
					response.ServerError(c)
					return
				}
			} else {
				response.ServerError(c)
				return
			}
		} else {

			if err := config.DB.Model(&userRevenue).Updates(map[string]interface{}{
				"revenue":     userRevenue.Revenue + invoice.TotalAmount,
				"order_count": userRevenue.OrderCount + 1,
			}).Error; err != nil {
				response.ServerError(c)
				return
			}
		}

	}

	order.Status = req.Status
	order.UpdatedAt = time.Now()

	if err := config.DB.Save(&order).Error; err != nil {
		response.ServerError(c)
		return
	}

	rdb, redisErr := config.ConnectRedis()
	if redisErr == nil {
		// Xóa tất cả các key con của "invoices"
		err := DeleteKeysByPattern(config.Ctx, rdb, "invoices:*")
		if err != nil {
			fmt.Printf("Lỗi khi xóa các key con của invoices: %v\n", err)
		}

		// Xóa các key khác
		cacheKey := "orders:all"
		cacheKeyUser := fmt.Sprintf("orders:all:user:%d", currentUserID)

		_ = services.DeleteFromRedis(config.Ctx, rdb, cacheKey)
		_ = services.DeleteFromRedis(config.Ctx, rdb, "accommodations:statuses")
		_ = services.DeleteFromRedis(config.Ctx, rdb, "rooms:statuses")
		_ = services.DeleteFromRedis(config.Ctx, rdb, cacheKeyUser)
	}

	response.Success(c, gin.H{"message": "Trạng thái đơn hàng đã được cập nhật"})
}

func GetOrderDetail(c *gin.Context) {
	orderId := c.Param("id")

	var order models.Order
	if err := config.DB.Preload("User").
		Preload("Accommodation").
		Preload("Room").
		Where("id = ?", orderId).
		First(&order).Error; err != nil {
		response.NotFound(c)
		return
	}
	var user dto.ActorResponse
	if order.UserID != nil {
		user = dto.ActorResponse{Name: order.User.Name, Email: order.User.Email, PhoneNumber: order.User.PhoneNumber}
	} else {
		user = dto.ActorResponse{Name: order.GuestName, Email: order.GuestEmail, PhoneNumber: order.GuestPhone}
	}

	accommodationResponse := convertToOrderAccommodationResponse(order.Accommodation)

	var roomResponses []dto.OrderRoomResponse
	for _, room := range order.Room {
		roomResponse := convertToOrderRoomResponse(room)
		roomResponses = append(roomResponses, roomResponse)
	}
	orderResponse := dto.OrderUserResponse{
		ID:               order.ID,
		User:             user,
		Accommodation:    accommodationResponse,
		Room:             roomResponses,
		CheckInDate:      order.CheckInDate,
		CheckOutDate:     order.CheckOutDate,
		Status:           order.Status,
		CreatedAt:        order.CreatedAt,
		UpdatedAt:        order.UpdatedAt,
		Price:            order.Price,
		HolidayPrice:     order.HolidayPrice,
		CheckInRushPrice: order.CheckInRushPrice,
		SoldOutPrice:     order.SoldOutPrice,
		DiscountPrice:    order.DiscountPrice,
		TotalPrice:       order.TotalPrice,
	}
	response.Success(c, orderResponse)
}

func GetOrdersByUserId(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		response.Unauthorized(c)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	currentUserID, _, err := GetUserIDFromToken(tokenString)
	if err != nil {
		response.Unauthorized(c)
		return
	}

	var user models.User
	if err := config.DB.First(&user, currentUserID).Error; err != nil {
		response.ServerError(c)
		return
	}

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

	if user.PhoneNumber == "" {
		response.BadRequest(c, "User phone number is missing")
		return
	}

	var ordersToUpdate []models.Order
	if err := config.DB.Where("guest_phone = ? AND user_id IS NULL", user.PhoneNumber).Find(&ordersToUpdate).Error; err != nil {
		response.ServerError(c)
		return
	}

	for _, order := range ordersToUpdate {
		if order.GuestPhone == user.PhoneNumber {
			order.UserID = &currentUserID
			if err := config.DB.Save(&order).Error; err != nil {
				response.ServerError(c)
				return
			}
		}
	}

	var totalOrders int64
	if err := config.DB.Model(&models.Order{}).Where("user_id = ?", currentUserID).Count(&totalOrders).Error; err != nil {
		response.ServerError(c)
		return
	}

	var orders []models.Order
	result := config.DB.Preload("User").
		Preload("Accommodation").
		Preload("Room").
		Where("user_id = ?", currentUserID).
		Order("created_at DESC").
		Offset(page * limit).
		Limit(limit).
		Find(&orders)

	if result.Error != nil {
		response.ServerError(c)
		return
	}
	if len(orders) == 0 {
		response.Success(c, []models.Order{})
		return
	}

	orderResponses := make([]dto.OrderUserResponse, 0)
	for _, order := range orders {
		var user dto.ActorResponse
		if order.UserID != nil {
			user = dto.ActorResponse{Name: order.User.Name, Email: order.User.Email, PhoneNumber: order.User.PhoneNumber}
		} else {
			user = dto.ActorResponse{Name: order.GuestName, Email: order.GuestEmail, PhoneNumber: order.GuestPhone}
		}

		accommodationResponse := convertToOrderAccommodationResponse(order.Accommodation)
		var roomResponses []dto.OrderRoomResponse
		for _, room := range order.Room {
			roomResponse := convertToOrderRoomResponse(room)
			roomResponses = append(roomResponses, roomResponse)
		}

		var invoiceCode string
		if order.Status == 1 {
			var invoice models.Invoice
			if err := config.DB.Where("order_id = ?", order.ID).First(&invoice).Error; err == nil {
				invoiceCode = invoice.InvoiceCode
			}
		}

		orderResponse := dto.OrderUserResponse{
			ID:               order.ID,
			User:             user,
			Accommodation:    accommodationResponse,
			Room:             roomResponses,
			CheckInDate:      order.CheckInDate,
			CheckOutDate:     order.CheckOutDate,
			Status:           order.Status,
			CreatedAt:        order.CreatedAt,
			UpdatedAt:        order.UpdatedAt,
			Price:            order.Price,
			HolidayPrice:     order.HolidayPrice,
			CheckInRushPrice: order.CheckInRushPrice,
			SoldOutPrice:     order.SoldOutPrice,
			DiscountPrice:    order.DiscountPrice,
			TotalPrice:       order.TotalPrice,
			InvoiceCode:      invoiceCode,
		}
		orderResponses = append(orderResponses, orderResponse)
	}

	response.SuccessWithPagination(c, orderResponses, page, limit, int(totalOrders))
}
