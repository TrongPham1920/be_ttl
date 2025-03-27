package controllers

import (
	"fmt"
	"log"
	"net/http"
	"new/config"
	"new/dto"
	"new/response"
	"sort"
	"strconv"
	"strings"
	"time"

	"new/models"
	"new/services"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type UserController struct {
	DB    *gorm.DB
	Redis *redis.Client
}

func NewUserController(mySQL *gorm.DB, redisCli *redis.Client) UserController {
	return UserController{
		DB:    mySQL,
		Redis: redisCli,
	}
}

func (u UserController) GetUsers(c *gin.Context) {
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

	pageStr := c.Query("page")
	limitStr := c.Query("limit")
	statusStr := c.Query("status")
	name := c.Query("name")
	roleStr := c.Query("role")

	page := 0
	limit := 10

	if pageStr != "" {
		page, _ = strconv.Atoi(pageStr)
	}
	if limitStr != "" {
		limit, _ = strconv.Atoi(limitStr)
	}

	// Tạo cache key dựa trên vai trò và bộ lọc
	var cacheKey string
	if currentUserRole == 1 {
		cacheKey = "users:all"
	} else if currentUserRole == 2 {
		cacheKey = fmt.Sprintf("users:role_3:admin_%d", currentUserID)
	} else {
		response.Forbidden(c)
		return
	}

	// Kết nối Redis
	rdb, err := config.ConnectRedis()
	if err != nil {
		log.Printf("Không thể kết nối Redis: %v", err)
	}

	var allUsers []models.User

	// Kiểm tra cache
	if err := services.GetFromRedis(config.Ctx, rdb, cacheKey, &allUsers); err != nil || len(allUsers) == 0 {
		// Nếu không có dữ liệu trong cache, truy vấn từ DB
		query := u.DB.Preload("Banks").Preload("Children")

		if currentUserRole == 3 {
			var adminID int
			if err := u.DB.Model(&models.User{}).Select("admin_id").Where("id = ?", currentUserID).Scan(&adminID).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"code": 0, "mess": "Không thể xác định admin_id của người dùng hiện tại"})
				return
			}

			query = query.Where("role = 3 AND admin_id = ?", adminID)
		} else if currentUserRole == 2 {
			query = query.Where("role = 3 AND admin_id = ?", currentUserID)
		}

		if err := query.Find(&allUsers).Error; err != nil {
			response.ServerError(c)
			return
		}

		// Lưu dữ liệu vào Redis
		if err := services.SetToRedis(config.Ctx, rdb, cacheKey, allUsers, 10*time.Minute); err != nil {
			log.Printf("Lỗi khi lưu danh sách người dùng vào Redis: %v", err)
		}
	}

	var filteredUsers []models.User
	for _, user := range allUsers {
		// Lọc theo status
		if statusStr != "" {
			status, _ := strconv.Atoi(statusStr)
			if user.Status != status {
				continue
			}
		}

		// Lọc theo name
		if name != "" && !strings.Contains(strings.ToLower(user.Name), strings.ToLower(name)) &&
			!strings.Contains(strings.ToLower(user.PhoneNumber), strings.ToLower(name)) &&
			!strings.Contains(strings.ToLower(user.Email), strings.ToLower(name)) {
			continue
		}

		// Lọc theo role
		if roleStr != "" {
			role, _ := strconv.Atoi(roleStr)
			if user.Role != role {
				continue
			}
		}

		filteredUsers = append(filteredUsers, user)
	}
	// Lọc và chuẩn bị response
	var userResponses []dto.UserResponse
	for _, user := range filteredUsers {

		if currentUserRole == 1 && user.Role == 3 {
			continue
		}

		if user.ID == uint(currentUserID) {
			continue
		}

		if currentUserRole == 2 {
			if user.Role != 3 || user.AdminId == nil || *user.AdminId != uint(currentUserID) {
				continue
			}
		}

		var banks []dto.Bank
		for _, bank := range user.Banks {
			banks = append(banks, dto.Bank{
				BankName:      bank.BankName,
				AccountNumber: bank.AccountNumber,
				BankShortName: bank.BankShortName,
			})
		}

		var childrenResponses []dto.UserResponse
		for _, child := range user.Children {
			var childBanks []dto.Bank
			for _, bank := range child.Banks {
				childBanks = append(childBanks, dto.Bank{
					BankName:      bank.BankName,
					AccountNumber: bank.AccountNumber,
					BankShortName: bank.BankShortName,
				})
			}

			childrenResponses = append(childrenResponses, dto.UserResponse{
				ID:          child.ID,
				Name:        child.Name,
				Email:       child.Email,
				IsVerified:  child.IsVerified,
				PhoneNumber: child.PhoneNumber,
				Role:        child.Role,
				Avatar:      child.Avatar,
				Banks:       childBanks,
				Status:      child.Status,
				UpdatedAt:   child.UpdatedAt,
				CreatedAt:   child.CreatedAt,
				Amount:      child.Amount,
			})
		}

		userResponses = append(userResponses, dto.UserResponse{
			ID:          user.ID,
			Name:        user.Name,
			Email:       user.Email,
			IsVerified:  user.IsVerified,
			PhoneNumber: user.PhoneNumber,
			Role:        user.Role,
			UpdatedAt:   user.UpdatedAt,
			CreatedAt:   user.CreatedAt,
			Avatar:      user.Avatar,
			Banks:       banks,
			Status:      user.Status,
			Children:    childrenResponses,
			AdminId:     user.AdminId,
			Amount:      user.Amount,
		})
	}

	// Sắp xếp và phân trang
	sort.Slice(userResponses, func(i, j int) bool {
		return userResponses[i].ID < userResponses[j].ID
	})

	start := page * limit
	end := start + limit
	if end > len(userResponses) {
		end = len(userResponses)
	}

	paginatedUsers := userResponses[start:end]

	response.SuccessWithPagination(c, paginatedUsers, page, limit, len(userResponses))
}

func (u *UserController) CreateUser(c *gin.Context) {
	// authHeader := c.GetHeader("Authorization")
	// if authHeader == "" {
	// 	response.Unauthorized(c)
	// 	return
	// }

	// tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	// currentUserID, err := GetIDFromToken(tokenString)
	// if err != nil {
	// 	response.Unauthorized(c)
	// 	return
	// }

	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if req.Role == 1 || req.Role == 2 || req.Role == 3 {
		// if req.Role == 3 {
		// 	var admin models.User
		// 	if err := u.DB.Where("id = ?", currentUserID).First(&admin).Error; err != nil {
		// 		response.BadRequest(c, "Không tìm thấy admin với ID: "+fmt.Sprint(currentUserID))
		// 		return
		// 	}
		// }

		var bankFake models.BankFake
		if err := u.DB.Where("id = ?", req.BankID).First(&bankFake).Error; err != nil {
			response.BadRequest(c, "Không tìm thấy ngân hàng giả")
			return
		}

		var existingBank models.Bank
		if err := u.DB.Where("account_number = ?", req.AccountNumber).First(&existingBank).Error; err == nil {
			response.Conflict(c)
			return
		}

		userValues := models.User{
			Email:       req.Email,
			Password:    req.Password,
			PhoneNumber: req.PhoneNumber,
			Role:        req.Role,
			Name:        req.Username,
			Amount:      req.Amount,
		}

		user, err := services.CreateUser(userValues)
		if err != nil {
			response.BadRequest(c, err.Error())
			return
		}

		bank := models.Bank{
			UserId:        user.ID,
			BankName:      bankFake.BankName,
			AccountNumber: req.AccountNumber,
			BankShortName: bankFake.BankShortName,
		}

		if err := u.DB.Create(&bank).Error; err != nil {
			response.ServerError(c)
			return
		}

		user.Banks = append(user.Banks, bank)
		if err := u.DB.Save(&user).Error; err != nil {
			response.ServerError(c)
			return
		}

		// if req.Role == 3 {
		// 	var admin models.User
		// 	if err := u.DB.Where("id = ?", currentUserID).First(&admin).Error; err != nil {
		// 		c.JSON(http.StatusBadRequest, gin.H{"code": 0, "mess": "Không tìm thấy admin với ID: " + fmt.Sprint(currentUserID)})
		// 		return
		// 	}

		// 	admin.Children = append(admin.Children, user)
		// 	if err := u.DB.Save(&admin).Error; err != nil {
		// 		c.JSON(http.StatusInternalServerError, gin.H{"code": 0, "mess": "Không thể cập nhật thông tin admin: " + err.Error()})
		// 		return
		// 	}
		// }

		rdb, redisErr := config.ConnectRedis()
		if redisErr == nil {
			switch req.Role {
			case 1, 2, 3:
				_ = services.DeleteFromRedis(config.Ctx, rdb, "user:all")
			}
		}

		response.Success(c, user)
	} else {
		response.BadRequest(c, "Vai trò không hợp lệ")
		return
	}
}

func (u UserController) GetUserByID(c *gin.Context) {
	var user models.User
	id := c.Param("id")

	if err := u.DB.First(&user, id).Error; err != nil {
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

	userResponse := dto.UserResponse{
		ID:          user.ID,
		Name:        user.Name,
		Email:       user.Email,
		IsVerified:  user.IsVerified,
		PhoneNumber: user.PhoneNumber,
		Role:        user.Role,
		Avatar:      user.Avatar,
		Banks:       banks,
		Status:      user.Status,
	}

	response.Success(c, userResponse)
}

func (u UserController) UpdateUser(c *gin.Context) {
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

	var updateUser dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&updateUser); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var user models.User
	if err := u.DB.Preload("Banks").First(&user, currentUserID).Error; err != nil {
		response.NotFound(c)
		return
	}

	if updateUser.Name != "" && updateUser.Name != " " {
		user.Name = updateUser.Name
	}
	if updateUser.PhoneNumber != "" && updateUser.PhoneNumber != " " {
		user.PhoneNumber = updateUser.PhoneNumber
	}
	if updateUser.Avatar != "" && updateUser.Avatar != " " {
		user.Avatar = updateUser.Avatar
	}

	user.Gender = updateUser.Gender

	if updateUser.DateOfBirth != "" && updateUser.DateOfBirth != " " {
		user.DateOfBirth = updateUser.DateOfBirth
	}

	if err := u.DB.Save(&user).Error; err != nil {
		response.ServerError(c)
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

	userResponse := dto.UserResponseUpdate{
		ID:          user.ID,
		Name:        user.Name,
		Email:       user.Email,
		IsVerified:  user.IsVerified,
		PhoneNumber: user.PhoneNumber,
		Role:        user.Role,
		Avatar:      user.Avatar,
		Banks:       banks,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
		Status:      user.Status,
		AdminId:     user.AdminId,
		Gender:      user.Gender,
		DateOfBirth: user.DateOfBirth,
	}

	//Xóa redis
	rdb, redisErr := config.ConnectRedis()
	if redisErr == nil {
		switch currentUserRole {
		case 1, 2:
			_ = services.DeleteFromRedis(config.Ctx, rdb, "user:all")
		case 3:
			adminCacheKey := fmt.Sprintf("users:role_3:admin_%d", currentUserID)
			_ = services.DeleteFromRedis(config.Ctx, rdb, adminCacheKey)
		}
	}

	response.Success(c, userResponse)
}

func (u UserController) ChangeUserStatus(c *gin.Context) {
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

	var statusRequest dto.UserStatusRequest
	if err := c.ShouldBindJSON(&statusRequest); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var user models.User

	if currentUserRole == 2 {
		if err := u.DB.Where("id = ? AND admin_id = ?", statusRequest.ID, currentUserID).First(&user).Error; err != nil {
			response.NotFound(c)
			return
		}
	} else if currentUserRole == 1 {
		if err := u.DB.First(&user, statusRequest.ID).Error; err != nil {
			response.NotFound(c)
			return
		}

		if user.Role == 2 {
			var childUsers []models.User
			if err := u.DB.Where("admin_id = ?", user.ID).Find(&childUsers).Error; err != nil {
				response.ServerError(c)
				return
			}

			for _, child := range childUsers {
				child.Status = statusRequest.Status
				if err := u.DB.Save(&child).Error; err != nil {
					response.ServerError(c)
					return
				}
			}
		}
	} else {
		response.Forbidden(c)
		return
	}

	user.Status = statusRequest.Status
	if err := u.DB.Save(&user).Error; err != nil {
		response.ServerError(c)
		return
	}

	//Xóa redis
	rdb, redisErr := config.ConnectRedis()
	if redisErr == nil {
		switch currentUserRole {
		case 1:
			_ = services.DeleteFromRedis(config.Ctx, rdb, "user:all")
		case 3:
			adminCacheKey := fmt.Sprintf("users:role_3:admin_%d", currentUserID)
			_ = services.DeleteFromRedis(config.Ctx, rdb, adminCacheKey)
		}
	}

	response.Success(c, user)
}

// Get Detail Receptionist
func (u UserController) GetReceptionistByID(c *gin.Context) {
	var user models.User
	id := c.Param("id")

	err := u.DB.Table("users").
		Where("users.id = ?", id).
		First(&user).Error

	if err != nil {
		response.NotFound(c)
		return
	}

	var banks []dto.Bank
	u.DB.Where("user_id = ?", id).Find(&banks)

	var accommodations []struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}

	var ids []int64
	for _, v := range user.AccommodationIDs {
		ids = append(ids, v)
	}

	if len(ids) > 0 {
		u.DB.Table("accommodations").
			Select("id, name").
			Where("id IN (?)", ids).
			Find(&accommodations)
	}

	userResponse := dto.UserResponse{
		ID:               user.ID,
		Name:             user.Name,
		Email:            user.Email,
		IsVerified:       user.IsVerified,
		PhoneNumber:      user.PhoneNumber,
		Role:             user.Role,
		Avatar:           user.Avatar,
		Banks:            banks,
		Status:           user.Status,
		DateOfBirth:      user.DateOfBirth,
		Amount:           user.Amount,
		AccommodationIDs: user.AccommodationIDs,
		CreatedAt:        user.CreatedAt,
		UpdatedAt:        user.UpdatedAt,
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 1,
		"mess": "Lấy thông tin lễ tân thành công",
		"data": gin.H{
			"user":           userResponse,
			"accommodations": accommodations,
		},
	})
}

// Get Bank SA
func (u UserController) GetBankSuperAdmin(c *gin.Context) {
	var user models.User

	err := u.DB.Table("users").
		Where("users.role = ?", 1).
		First(&user).Error

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 0, "mess": "Không tìm thấy tài khoản SA"})
		return
	}

	var bank []dto.Bank
	u.DB.Where("user_id = ?", user.ID).Find(&bank)

	c.JSON(http.StatusOK, gin.H{
		"code": 1,
		"mess": "Lấy thông tin tài khoản SA thành công",
		"data": gin.H{
			"sabank": bank,
		},
	})
}

// get Profile
func (u UserController) GetProfile(c *gin.Context) {
	var user models.User
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 0, "mess": "Authorization header is missing"})
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	id, _, err := GetUserIDFromToken(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 0, "mess": "Invalid token"})
		return
	}

	if err := u.DB.Preload("Banks").First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 0, "mess": "Người dùng không tồn tại"})
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

	userResponse := dto.UserResponse{
		ID:               user.ID,
		Name:             user.Name,
		Email:            user.Email,
		IsVerified:       user.IsVerified,
		PhoneNumber:      user.PhoneNumber,
		Role:             user.Role,
		Avatar:           user.Avatar,
		Banks:            banks,
		Status:           user.Status,
		DateOfBirth:      user.DateOfBirth,
		Amount:           user.Amount,
		AccommodationIDs: user.AccommodationIDs,
		CreatedAt:        user.CreatedAt,
		UpdatedAt:        user.UpdatedAt,
	}

	c.JSON(http.StatusOK, gin.H{"code": 1, "mess": "Lấy người dùng thành công", "data": userResponse})
}
