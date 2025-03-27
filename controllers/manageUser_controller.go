package controllers

import (
	"errors"
	"fmt"
	"log"
	"math"
	"net/url"
	"new/config"
	"new/dto"
	"new/response"
	"new/services"
	"sort"
	"strconv"
	"strings"
	"time"

	"new/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UpdateBalanceRequest struct {
	UserID uint  `json:"userId" binding:"required"`
	Amount int64 `json:"amount" binding:"required"`
}

func GetUsersByAdminID(adminID uint) ([]models.User, error) {
	var users []models.User
	if err := config.DB.Where("admin_id = ?", adminID).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func GetCheckedInUsers(startDate string, endDate string, users []models.User) ([]models.CheckInRecord, error) {
	var CheckIn []models.CheckInRecord
	userIDs := getUserIDs(users)

	if err := config.DB.
		Table("check_in_records").
		Where("DATE(check_in_records.date) BETWEEN ? AND ?", startDate, endDate).
		Where("check_in_records.user_id IN (?)", userIDs).
		Find(&CheckIn).Error; err != nil {
		return nil, err
	}

	return CheckIn, nil
}

func GetUserSalaries(startDate string, endDate string, users []models.User) ([]models.UserSalary, error) {
	var salaries []models.UserSalary
	userIDs := getUserIDs(users)

	if err := config.DB.
		Table("user_salaries").
		Where("DATE(salary_date) BETWEEN ? AND ?", startDate, endDate).
		Where("user_id IN (?)", userIDs).
		Find(&salaries).Error; err != nil {
		return nil, err
	}

	return salaries, nil
}

func getUserIDs(users []models.User) []uint {
	ids := make([]uint, len(users))
	for i, user := range users {
		ids[i] = user.ID
	}
	return ids
}

func GetUserAcc(c *gin.Context) {
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

	if currentUserRole != 2 {
		response.Forbidden(c)
		return
	}

	cacheKey := fmt.Sprintf("accommodations:admin:%d", currentUserID)

	rdb, err := config.ConnectRedis()
	if err != nil {
		log.Printf("Không thể kết nối Redis: %v", err)
	}

	var allAccommodations []models.Accommodation

	tx := config.DB.Model(&models.Accommodation{}).
		Preload("Rooms").
		Preload("Rates").
		Preload("Benefits").
		Preload("User").
		Preload("User.Banks").
		Where("user_id = ?", currentUserID)

	if err := tx.Find(&allAccommodations).Error; err != nil {
		response.ServerError(c)
		return
	}

	accUser := make([]dto.AccommodationDetailResponse, 0)
	for _, acc := range allAccommodations {
		user := acc.User
		// Lấy thông tin ngân hàng nếu có
		bankShortName := ""
		accountNumber := ""
		bankName := ""
		if len(user.Banks) > 0 {
			bankShortName = user.Banks[0].BankShortName
			accountNumber = user.Banks[0].AccountNumber
			bankName = user.Banks[0].BankName
		}

		accUser = append(accUser, dto.AccommodationDetailResponse{
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
				BankShortName: bankShortName,
				AccountNumber: accountNumber,
				BankName:      bankName,
			},
		})
	}

	if rdb != nil {
		if err := services.SetToRedis(config.Ctx, rdb, cacheKey, accUser, 60*time.Minute); err != nil {
			log.Printf("Lỗi khi lưu danh sách chỗ ở vào Redis: %v", err)
		}
	}

	if rdb != nil {
		if err := services.GetFromRedis(config.Ctx, rdb, cacheKey, &allAccommodations); err == nil && len(allAccommodations) > 0 {
			goto RESPONSE
		}
	}

RESPONSE:
	accommodationsResponse := make([]gin.H, 0)
	for _, acc := range allAccommodations {
		accommodationsResponse = append(accommodationsResponse, gin.H{
			"id":   acc.ID,
			"name": acc.Name,
		})
	}

	response.Success(c, accommodationsResponse)
}

func (u UserController) UpdateUserBalance(c *gin.Context) {
	var req UpdateBalanceRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Dữ liệu không hợp lệ")
		return
	}

	if req.Amount > 2000000 {
		response.BadRequest(c, "Không được vượt quá 2.000.000")
		return
	} else if req.Amount < -1000000 {
		response.BadRequest(c, "Không được nhỏ hơn -1.000.000")
		return
	}

	var user models.User

	if err := config.DB.First(&user, req.UserID).Error; err != nil {
		response.NotFound(c)
		return
	}

	now := time.Now()

	user.Amount += req.Amount
	user.DateCheck = now

	if err := config.DB.Save(&user).Error; err != nil {
		response.ServerError(c)
		return
	}
	//Xóa redis cache
	rdb, redisErr := config.ConnectRedis()
	if redisErr == nil {
		_ = services.DeleteFromRedis(config.Ctx, rdb, "user:all")
	}

	response.Success(c, gin.H{
		"userId":    user.ID,
		"amount":    user.Amount,
		"dateCheck": user.DateCheck,
	})
}

func (u UserController) UpdateUserAccommodation(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		response.Unauthorized(c)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	currentUserID, err := GetIDFromToken(tokenString)
	if err != nil {
		response.Unauthorized(c)
		return
	}

	var req struct {
		UserID           uint    `json:"userId"`
		AccommodationIDs []int64 `json:"accommodationIds"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Dữ liệu không hợp lệ")
		return
	}

	if len(req.AccommodationIDs) > 10 {
		response.BadRequest(c, "Chỉ được gửi tối đa 5 địa điểm lưu trú")
		return
	}

	var user models.User
	if err := config.DB.First(&user, req.UserID).Error; err != nil {
		response.NotFound(c)
		return
	}

	if user.Role != 3 || user.AdminId == nil || *user.AdminId != currentUserID {
		response.Forbidden(c)
		return
	}

	var count int64
	if err := config.DB.Model(&models.Accommodation{}).
		Where("id IN ? AND user_id = ?", req.AccommodationIDs, currentUserID).
		Count(&count).Error; err != nil {
		response.ServerError(c)
		return
	}

	if count != int64(len(req.AccommodationIDs)) {
		response.Forbidden(c)
		return
	}

	user.AccommodationIDs = req.AccommodationIDs
	if err := config.DB.Save(&user).Error; err != nil {
		response.ServerError(c)
		return
	}

	//Xóa redis cache
	rdb, redisErr := config.ConnectRedis()
	if redisErr == nil {
		_ = services.DeleteFromRedis(config.Ctx, rdb, "user:all")
	}

	response.Success(c, gin.H{
		"userId":          user.ID,
		"accommodationId": user.AccommodationIDs,
	})
}

func (u UserController) CheckInUser(c *gin.Context) {
	var req struct {
		UserID    uint    `json:"userId"`
		Longitude float64 `json:"longitude"`
		Latitude  float64 `json:"latitude"`
	}

	var currentTime = time.Now()

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Dữ liệu không hợp lệ")
		return
	}

	var user models.User
	if err := config.DB.First(&user, req.UserID).Error; err != nil {
		response.NotFound(c)
		return
	}

	if user.AccommodationIDs == nil || len(user.AccommodationIDs) == 0 {
		response.BadRequest(c, "Người dùng chưa có thông tin lưu trú")
		return
	}

	var accommodations []models.Accommodation
	if err := config.DB.Where("id IN ?", user.AccommodationIDs).Find(&accommodations).Error; err != nil {
		response.ServerError(c)
		return
	}

	const earthRadiusKm = 6371.0

	distance := func(lat1, lon1, lat2, lon2 float64) float64 {
		lat1Rad, lon1Rad := lat1*(math.Pi/180), lon1*(math.Pi/180)
		lat2Rad, lon2Rad := lat2*(math.Pi/180), lon2*(math.Pi/180)
		dLat, dLon := lat2Rad-lat1Rad, lon2Rad-lon1Rad

		a := math.Sin(dLat/2)*math.Sin(dLat/2) +
			math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
		c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

		return earthRadiusKm * c
	}

	validLocation := false
	for _, acc := range accommodations {
		d := distance(acc.Latitude, acc.Longitude, req.Latitude, req.Longitude)
		if d <= 0.1 {
			validLocation = true
			break
		}
	}

	if !validLocation {
		response.BadRequest(c, "Vị trí không hợp lệ")
		return
	}

	var existingRecord models.CheckInRecord
	today := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, currentTime.Location())

	if err := config.DB.Where("user_id = ? AND DATE(date) = ?", req.UserID, today).First(&existingRecord).Error; err == nil {
		response.BadRequest(c, "Người dùng đã điểm danh hôm nay")
		return
	}

	user.Status = 1

	checkInRecord := models.CheckInRecord{
		UserID: req.UserID,
		Date:   currentTime,
	}

	if err := config.DB.Create(&checkInRecord).Error; err != nil {
		response.ServerError(c)
		return
	}

	//Xóa redis
	rdb, redisErr := config.ConnectRedis()
	if redisErr == nil {
		adminCacheKey := fmt.Sprintf("user_checkin:%d", *user.AdminId)
		_ = services.DeleteFromRedis(config.Ctx, rdb, adminCacheKey)
	}

	response.Success(c, gin.H{
		"userId":      user.ID,
		"status":      user.Status,
		"timeCheckIn": currentTime,
	})
}

func GetUserCalendar(c *gin.Context) {
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

	if currentUserRole != 2 {
		response.Forbidden(c)
		return
	}

	date := c.Query("date")

	if date == "" {
		response.BadRequest(c, "Thiếu tham số date")
		return
	}

	parsedDate, err := time.Parse("01/2006", date)
	if err != nil {
		response.BadRequest(c, "Định dạng date không hợp lệ")
		return
	}

	year, month, _ := parsedDate.Date()
	daysInMonth := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
	days := make([]gin.H, 0)

	users, err := GetUsersByAdminID(currentUserID)
	if err != nil {
		response.ServerError(c)
		return
	}

	startDate := fmt.Sprintf("%04d-%02d-01", year, month)
	endDate := fmt.Sprintf("%04d-%02d-%02d", year, month, daysInMonth)

	checkedInUsers, err := GetCheckedInUsers(startDate, endDate, users)
	if err != nil {
		response.ServerError(c)
		return
	}

	for day := 1; day <= daysInMonth; day++ {
		dateStr := fmt.Sprintf("%04d-%02d-%02d", year, month, day)
		userList := make([]gin.H, 0)

		for _, record := range checkedInUsers {
			if record.Date.Format("2006-01-02") == dateStr {
				for _, user := range users {
					if user.ID == record.UserID {
						userList = append(userList, gin.H{"id": user.ID, "name": user.Name})
					}
				}
			}
		}

		days = append(days, gin.H{
			"date":  dateStr,
			"users": userList,
		})
	}

	response.SuccessWithTotal(c,
		days,
		daysInMonth,
	)
}

func CalculateUserSalaryInit(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		response.Unauthorized(c)
		return
	}
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	currentUserID, err := GetIDFromToken(tokenString)
	if err != nil {
		response.Unauthorized(c)
		return
	}

	var req struct {
		UserID uint `json:"userId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Dữ liệu không hợp lệ")
		return
	}

	var user models.User
	if err := config.DB.First(&user, req.UserID).Error; err != nil {
		response.NotFound(c)
		return
	}
	if user.Role != 3 || user.AdminId == nil || *user.AdminId != currentUserID {
		response.Forbidden(c)
		return
	}

	// Xác định thời gian
	currentMonth := time.Now().Format("2006-01")
	startOfMonth, _ := time.Parse("2006-01-02", currentMonth+"-01")
	startOfNextMonth := startOfMonth.AddDate(0, 1, 0)

	// Lấy dữ liệu điểm danh
	startDate := currentMonth + "-01"
	endDate := startOfNextMonth.Add(-time.Hour * 24).Format("2006-01-02")

	checkIns, err := GetCheckedInUsers(startDate, endDate, []models.User{user})
	if err != nil {
		response.ServerError(c)
		return
	}

	totalDays := startOfNextMonth.Add(-time.Hour * 24).Day()
	attendanceCount := len(checkIns)
	absenceCount := totalDays - attendanceCount
	amount := int(math.Round(float64(user.Amount)/float64(totalDays)*float64(attendanceCount)/1000) * 1000)

	// Kiểm tra hoặc cập nhật lương
	var userSalary models.UserSalary
	if err := config.DB.Where("user_id = ? AND salary_date >= ? AND salary_date < ?", user.ID, startOfMonth, startOfNextMonth).First(&userSalary).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			userSalary = models.UserSalary{
				UserID:     user.ID,
				Amount:     amount,
				Attendance: attendanceCount,
				Absence:    absenceCount,
				SalaryDate: time.Now(),
			}
		} else {
			response.ServerError(c)
			return
		}
	} else {
		userSalary.Amount = amount
		userSalary.Attendance = attendanceCount
		userSalary.Absence = absenceCount
		userSalary.SalaryDate = time.Now()
	}

	if err := config.DB.Save(&userSalary).Error; err != nil {
		response.ServerError(c)
		return
	}

	response.Success(c, gin.H{
		"id":         userSalary.ID,
		"userId":     user.ID,
		"amount":     amount,
		"attendance": attendanceCount,
		"absence":    absenceCount,
		"date":       currentMonth,
		"totalDays":  totalDays,
	})
}

func CalculateUserSalary(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		response.Unauthorized(c)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	currentUserID, err := GetIDFromToken(tokenString)
	if err != nil {
		response.Unauthorized(c)
		return
	}

	var req struct {
		SalaryID uint `json:"salaryId"`
		UserID   uint `json:"userId"`
		Bonus    int  `json:"bonus"`
		Penalty  int  `json:"penalty"`
		Amount   int  `json:"amount"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Dữ liệu không hợp lệ")
		return
	}

	var user models.User
	if err := config.DB.Preload("Banks").First(&user, req.UserID).Error; err != nil {
		response.NotFound(c)
		return
	}

	var banks []dto.Bank
	for _, bank := range user.Banks {
		banks = append(banks, dto.Bank{
			BankName:      bank.BankName,
			AccountNumber: bank.AccountNumber,
			BankShortName: bank.BankShortName,
		})
	}

	if user.Role != 3 || user.AdminId == nil || *user.AdminId != currentUserID {
		response.Forbidden(c)
		return
	}

	baseSalary := req.Amount
	totalSalary := int(baseSalary) + req.Bonus - req.Penalty

	// Tìm hoặc tạo bản ghi usersalary
	var userSalary models.UserSalary
	if err := config.DB.Where("id = ?", req.SalaryID).First(&userSalary).Error; err != nil {
		response.NotFound(c)
		return
	}

	// Cập nhật thông tin lương
	userSalary.TotalSalary = int(totalSalary)
	userSalary.Bonus = req.Bonus
	userSalary.Penalty = req.Penalty

	if err := config.DB.Save(&userSalary).Error; err != nil {
		response.ServerError(c)
		return
	}

	response.Success(c, gin.H{
		"userId":      user.ID,
		"baseSalary":  user.Amount,
		"bonus":       req.Bonus,
		"penalty":     req.Penalty,
		"totalSalary": totalSalary,
		"bank":        banks,
	})
}

func UpdateSalaryStatus(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		response.Unauthorized(c)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	currentUserID, err := GetIDFromToken(tokenString)
	if err != nil {
		response.Unauthorized(c)
		return
	}

	var req struct {
		SalaryID uint `json:"salaryId"`
		Status   bool `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Dữ liệu không hợp lệ")
		return
	}

	var userSalary models.UserSalary
	if err := config.DB.First(&userSalary, req.SalaryID).Error; err != nil {
		response.NotFound(c)
		return
	}

	var user models.User
	if err := config.DB.First(&user, userSalary.UserID).Error; err != nil {
		response.NotFound(c)
		return
	}

	if user.AdminId == nil || *user.AdminId != currentUserID {
		response.Forbidden(c)
		return
	}

	// Cập nhật trạng thái
	userSalary.Status = req.Status
	if err := config.DB.Save(&userSalary).Error; err != nil {
		response.ServerError(c)
		return
	}

	response.Success(c, gin.H{
		"salaryId": req.SalaryID,
		"status":   req.Status,
	})
}

func GetUserCheckin(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		response.Unauthorized(c)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	currentUserID, err := GetIDFromToken(tokenString)
	if err != nil {
		response.Unauthorized(c)
		return
	}

	// Kết nối Redis và tạo cacheKey
	rdb, err := config.ConnectRedis()
	if err != nil {
		response.ServerError(c)
		return
	}
	cacheKey := fmt.Sprintf("user_checkin:%d", currentUserID)

	var checkinResponse []gin.H
	// Kiểm tra cache trong Redis
	if err := services.GetFromRedis(config.Ctx, rdb, cacheKey, &checkinResponse); err == nil && len(checkinResponse) > 0 {
		// Nếu dữ liệu có trong cache, không cần truy vấn lại DB
		log.Println("Dữ liệu lấy từ cache")
	} else {
		// Nếu không có dữ liệu trong cache, thực hiện truy vấn DB
		var user models.User
		if err := config.DB.First(&user, currentUserID).Error; err != nil {
			response.NotFound(c)
			return
		}

		date := c.Query("date")
		if date == "" {
			date = time.Now().Format("01/2006")
		}

		parsedDate, err := time.Parse("01/2006", date)
		if err != nil {
			response.BadRequest(c, "Định dạng date không hợp lệ")
			return
		}
		year, month, _ := parsedDate.Date()
		daysInMonth := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()

		users, err := GetUsersByAdminID(currentUserID)
		if err != nil {
			response.ServerError(c)
			return
		}

		sort.Slice(users, func(i, j int) bool {
			return users[i].UpdatedAt.After(users[j].UpdatedAt)
		})

		startDate := fmt.Sprintf("%04d-%02d-01", year, month)
		endDate := fmt.Sprintf("%04d-%02d-%02d", year, month, daysInMonth)

		checkedInUsers, err := GetCheckedInUsers(startDate, endDate, users)
		if err != nil {
			response.ServerError(c)
			return
		}

		for _, u := range users {
			checkinCount := 0
			var checkinDates []time.Time
			for _, ci := range checkedInUsers {
				if ci.UserID == u.ID {
					checkinCount++
					checkinDates = append(checkinDates, ci.Date)
				}
			}
			notCheckedInDays := daysInMonth - checkinCount

			checkinResponse = append(checkinResponse, gin.H{
				"id":               u.ID,
				"name":             u.Name,
				"phoneNumber":      u.PhoneNumber,
				"amount":           u.Amount,
				"checkinCount":     checkinCount,
				"notCheckedInDays": notCheckedInDays,
				"checkinDates":     checkinDates,
			})
		}

		// Lưu cache vào Redis
		err = services.SetToRedis(config.Ctx, rdb, cacheKey, checkinResponse, 30*time.Minute)
		if err != nil {
			log.Printf("Lỗi khi lưu dữ liệu vào Redis: %v", err)
		}
		log.Println("Dữ liệu đã được lưu vào Redis")
	}

	// Lọc dữ liệu sau khi đã có response
	nameFilter := c.Query("name")
	phoneFilter := c.Query("phone")
	var filteredResponse []gin.H
	for _, r := range checkinResponse {
		nameVal, _ := r["name"].(string)
		phoneVal, _ := r["phoneNumber"].(string)
		if (nameFilter == "" || strings.Contains(strings.ToLower(nameVal), strings.ToLower(nameFilter))) &&
			(phoneFilter == "" || strings.Contains(phoneVal, phoneFilter)) {
			filteredResponse = append(filteredResponse, r)
		}
	}

	total := len(filteredResponse)

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

	startIdx := page * limit
	endIdx := startIdx + limit
	if startIdx >= len(filteredResponse) {
		filteredResponse = []gin.H{}
	} else if endIdx > len(filteredResponse) {
		filteredResponse = filteredResponse[startIdx:]
	} else {
		filteredResponse = filteredResponse[startIdx:endIdx]
	}

	response.SuccessWithPagination(c, filteredResponse, page, limit, total)
}

func GetUserSalary(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		response.Unauthorized(c)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	currentUserID, err := GetIDFromToken(tokenString)
	if err != nil {
		response.Unauthorized(c)
		return
	}

	var user models.User
	if err := config.DB.First(&user, currentUserID).Error; err != nil {
		response.NotFound(c)
		return
	}

	date := c.Query("date")
	if date == "" {
		date = time.Now().Format("01/2006")
	}

	parsedDate, err := time.Parse("01/2006", date)
	if err != nil {
		response.BadRequest(c, "Định dạng date không hợp lệ")
		return
	}
	year, month, _ := parsedDate.Date()
	daysInMonth := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()

	users, err := GetUsersByAdminID(currentUserID)
	if err != nil {
		response.ServerError(c)
		return
	}

	sort.Slice(users, func(i, j int) bool {
		return users[i].UpdatedAt.After(users[j].UpdatedAt)
	})

	startDate := fmt.Sprintf("%04d-%02d-01", year, month)
	endDate := fmt.Sprintf("%04d-%02d-%02d", year, month, daysInMonth)

	userSalaries, err := GetUserSalaries(startDate, endDate, users)
	if err != nil {
		response.ServerError(c)
		return
	}

	userMap := make(map[uint]models.User)
	for _, u := range users {
		userMap[u.ID] = u
	}

	var salaryResponse []gin.H
	for _, s := range userSalaries {

		userInfo, ok := userMap[s.UserID]
		if !ok {
			continue
		}

		salaryResponse = append(salaryResponse, gin.H{
			"id":          s.UserID,
			"amount":      userInfo.Amount,
			"name":        userInfo.Name,
			"phoneNumber": userInfo.PhoneNumber,
			"totalSalary": s.TotalSalary,
			"bonus":       s.Bonus,
			"penalty":     s.Penalty,
			"status":      s.Status,
			"code":        s.ID,
		})
	}

	nameFilter := c.Query("name")
	phoneFilter := c.Query("phone")
	var filteredResponse []gin.H
	for _, r := range salaryResponse {
		nameVal, _ := r["name"].(string)
		phoneVal, _ := r["phoneNumber"].(string)
		if (nameFilter == "" || strings.Contains(strings.ToLower(nameVal), strings.ToLower(nameFilter))) &&
			(phoneFilter == "" || strings.Contains(phoneVal, phoneFilter)) {
			filteredResponse = append(filteredResponse, r)
		}
	}

	total := len(filteredResponse)

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

	startIdx := page * limit
	endIdx := startIdx + limit
	if startIdx >= len(filteredResponse) {
		filteredResponse = []gin.H{}
	} else if endIdx > len(filteredResponse) {
		filteredResponse = filteredResponse[startIdx:]
	} else {
		filteredResponse = filteredResponse[startIdx:endIdx]
	}

	response.SuccessWithPagination(c, filteredResponse, page, limit, total)
}

func GetAccommodationReceptionist(c *gin.Context) {
	var user models.User
	var accommodations []models.Accommodation

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

	//Tạo cache key Redis
	var cacheKey string
	if currentUserRole == 3 {
		cacheKey = fmt.Sprintf("accom_role_3:%d", currentUserID)
	}
	// Kết nối Redis
	rdb, err := config.ConnectRedis()
	if err != nil {
		response.ServerError(c)
		return
	}
	if err := services.GetFromRedis(config.Ctx, rdb, cacheKey, &accommodations); err == nil && len(accommodations) > 0 {
		// Nếu dữ liệu có trong cache, không cần truy vấn lại DB
		log.Println("Dữ liệu lấy từ cache")
	} else {
		// Lấy thông tin user từ DB
		if err := config.DB.First(&user, currentUserID).Error; err != nil {
			response.NotFound(c)
			return
		}

		// Kiểm tra xem user có danh sách accommodation không
		if len(user.AccommodationIDs) > 0 {
			var ids []int64
			for _, id := range user.AccommodationIDs {
				ids = append(ids, id)
			}

			if err := config.DB.Where("id IN (?)", ids).Find(&accommodations).Error; err != nil {
				response.ServerError(c)
				return
			}
		}

		// Lưu cache vào Redis
		err = services.SetToRedis(config.Ctx, rdb, cacheKey, accommodations, 30*time.Minute)
		if err != nil {
			log.Printf("Lỗi khi lưu dữ liệu vào Redis: %v", err)
		}
		log.Println("Dữ liệu đã được lưu vào Redis")
	}

	// Lọc dữ liệu sau khi đã có response
	typeFilter := c.Query("type")
	nameFilter := c.Query("name")
	statusFilter := c.Query("status")
	numBedFilter := c.Query("numBed")
	numToletFilter := c.Query("numTolet")
	peopleFilter := c.Query("people")
	provinceFilter := c.Query("province")
	filteredResponse := make([]models.Accommodation, 0)
	for _, acc := range accommodations {

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

		filteredResponse = append(filteredResponse, acc)
	}

	total := len(filteredResponse)

	//Xếp theo update mới nhất
	sort.Slice(filteredResponse, func(i, j int) bool {
		return filteredResponse[i].UpdateAt.After(filteredResponse[j].UpdateAt)
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
	if start >= len(filteredResponse) {
		filteredResponse = []models.Accommodation{}
	} else if end > len(filteredResponse) {
		filteredResponse = filteredResponse[start:]
	} else {
		filteredResponse = filteredResponse[start:end]
	}

	response.SuccessWithPagination(c, filteredResponse, page, limit, total)
}
