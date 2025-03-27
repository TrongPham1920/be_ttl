package routes

import (
	"context"
	"fmt"
	"net/http"
	"new/config"
	"new/controllers"
	middlewares "new/middleware"

	"github.com/gin-gonic/gin"
	"github.com/olahol/melody"

	"github.com/redis/go-redis/v9"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"gorm.io/gorm"
)

func SetupRoutes(router *gin.Engine, db *gorm.DB, redisCli *redis.Client, cld *cloudinary.Cloudinary, m *melody.Melody) {

	userController := controllers.NewUserController(db, redisCli)

	v1 := router.Group("/api/v1")
	v1.GET("/users", middlewares.AuthMiddleware(1, 2), userController.GetUsers)
	v1.POST("/users", middlewares.AuthMiddleware(1, 2), userController.CreateUser)
	v1.GET("/users/:id", userController.GetUserByID)
	v1.PUT("/users", middlewares.AuthMiddleware(1, 2, 3, 0), userController.UpdateUser)
	v1.PUT("/userStatus", middlewares.AuthMiddleware(1, 2), userController.ChangeUserStatus)
	v1.GET("/receptionist/:id", userController.GetReceptionistByID)
	v1.GET("/sabank", userController.GetBankSuperAdmin)
	v1.GET("/profile", userController.GetProfile)

	v1.GET("/userAcc", middlewares.AuthMiddleware(2), controllers.GetUserAcc)
	v1.PUT("/updateBalance", middlewares.AuthMiddleware(1, 2), userController.UpdateUserBalance)
	v1.PUT("/updateUserAcc", middlewares.AuthMiddleware(2), userController.UpdateUserAccommodation)
	v1.POST("/checkin", userController.CheckInUser)
	v1.GET("/userCalendar", middlewares.AuthMiddleware(2), controllers.GetUserCalendar)
	v1.POST("/userSalaryInit", middlewares.AuthMiddleware(2), controllers.CalculateUserSalaryInit)
	v1.POST("/userSalaryCommit", middlewares.AuthMiddleware(2), controllers.CalculateUserSalary)
	v1.PUT("/userSalaryStatus", middlewares.AuthMiddleware(2), controllers.UpdateSalaryStatus)
	v1.GET("/userCheckin", middlewares.AuthMiddleware(2), controllers.GetUserCheckin)
	v1.GET("/salaryHistory", middlewares.AuthMiddleware(2), controllers.GetUserSalary)

	v1.GET("/verify-email", controllers.VerifyEmail)
	v1.POST("/auth/login", controllers.Login)
	v1.DELETE("/auth/logout", controllers.Logout)
	v1.POST("/auth/register", controllers.RegisterUser)
	v1.POST("/resendCode", controllers.ResendVerificationCode)
	v1.POST("/forgetPassword", controllers.ForgetPassword)
	v1.POST("/newPassword", controllers.ResetPassword)
	v1.POST("/verifyCode", controllers.VerifyCode)
	v1.POST("/auth/google", controllers.AuthGoogle)

	v1.GET("/room", controllers.GetAllRooms)
	v1.GET("/roomUser", controllers.GetAllRoomsUser)
	v1.POST("/room", controllers.CreateRoom)
	v1.GET("/room/:id", controllers.GetRoomDetail)
	v1.PUT("/roomUpdate", controllers.UpdateRoom)
	v1.PUT("/roomStatus", controllers.ChangeRoomStatus)
	v1.GET("/checkRoom", controllers.GetRoomBookingDates)

	v1.GET("/accommodationUser", controllers.GetAllAccommodationsForUser)
	v1.GET("/accommodation", controllers.GetAllAccommodations)
	v1.POST("/accommodation", controllers.CreateAccommodation)
	v1.GET("/accommodation/:id", controllers.GetAccommodationDetail)
	v1.PUT("/accommodationUpdate", controllers.UpdateAccommodation)
	v1.PUT("/accommodationStatus", controllers.ChangeAccommodationStatus)
	v1.GET("/checkAcc", controllers.GetAccBookingDates)
	v1.GET("/accommodationReceptionist", controllers.GetAccommodationReceptionist)

	v1.GET("/banks", controllers.GetAllBanks)
	v1.POST("/add-banks", controllers.CreateBank)
	v1.PUT("/update-banks", controllers.AddAccountNumbers)
	v1.DELETE("/del-banks", controllers.DeleteAllBanks)

	v1.GET("/benefit", controllers.GetAllBenefit)
	v1.POST("/benefit", controllers.CreateBenefit)
	v1.GET("/benefit/:id", controllers.GetBenefitDetail)
	v1.PUT("/benefitUpdate", controllers.UpdateBenefit)
	v1.PUT("/benefitStatus", controllers.ChangeBenefitStatus)

	v1.GET("/rates", controllers.GetAllRates)
	v1.POST("/rates", controllers.CreateRate)
	v1.GET("/rates/:id", controllers.GetRateDetail)
	v1.PUT("/ratesUpdate", controllers.UpdateRate)

	v1.GET("/order", controllers.GetOrders)
	v1.POST("/order", controllers.CreateOrder)
	v1.PUT("/orderUpdate", controllers.ChangeOrderStatus)
	v1.GET("/order/:id", controllers.GetOrderDetail)
	v1.GET("/orderHistory", controllers.GetOrdersByUserId)

	v1.GET("/holidays", controllers.GetHolidays)
	v1.POST("/holidays", controllers.CreateHoliday)
	v1.PUT("/holidaysUpdate", controllers.UpdateHoliday)
	v1.GET("/holidays/:id", controllers.GetDetailHoliday)
	v1.DELETE("/holidays", controllers.DeleteHoliday)

	v1.GET("/discount", controllers.GetDiscounts)
	v1.GET("/discount/:id", controllers.GetDiscountDetail)
	v1.POST("/discount", controllers.CreateDiscount)
	v1.PUT("/discountUpdate", controllers.UpdateDiscount)
	v1.DELETE("/discount/:id", controllers.DeleteDiscount)
	v1.PUT("/discountStatus", controllers.ChangeDiscountStatus)

	v1.GET("/invoices", controllers.GetInvoices)
	v1.GET("/invoices/:id", controllers.GetDetailInvoice)

	v1.POST("/sendpay", controllers.SendPay)
	v1.PUT("/paymentStatus", controllers.UpdatePaymentStatus)

	//doanh thu
	v1.GET("/revenue", controllers.GetTotalRevenue)
	v1.GET("/revenue/detail", controllers.GetTotal)
	v1.GET("/today", controllers.GetToday)
	v1.GET("/todayUser", middlewares.AuthMiddleware(1), controllers.GetTodayUser)
	v1.GET("/userRevenue", middlewares.AuthMiddleware(1), controllers.GetUserRevene)

	//Đơn rút tiền
	v1.POST("/createWithdrawalHistory", controllers.CreateWithdrawalHistory)
	v1.GET("/getWithdrawalHistory", controllers.GetWithdrawalHistory)
	v1.POST("/confirmWithdrawalHistory", controllers.ConfirmWithdrawalHistory)

	v1.POST("/img/multi-upload", func(c *gin.Context) {
		form, er := c.MultipartForm()
		if er != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không có file"})
		}
		files := form.File["files"]
		if len(files) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không có file"})
			return
		}

		var urls []string
		for _, file := range files {
			src, err := file.Open()
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Lỗi khi mở file"})
				return
			}
			defer src.Close()

			ctx := context.Background()
			resp, err := config.Cloudinary.Upload.Upload(ctx, src, uploader.UploadParams{Folder: "uploads"})
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Upload thất bại"})
				return
			}
			urls = append(urls, resp.SecureURL)
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Upload thành công",
			"urls":    urls,
		})
	})

	v1.POST("/img/upload", func(c *gin.Context) {
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không có file"})
			return
		}

		src, err := file.Open()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Lỗi khi mở file"})
			return
		}
		defer src.Close()

		ctx := context.Background()
		resp, err := config.Cloudinary.Upload.Upload(ctx, src, uploader.UploadParams{Folder: "avatars"})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Upload thất bại"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Upload avatar thành công",
			"url":     resp.SecureURL,
		})
	})

	//ws
	v1.GET("/test-broadcast", func(c *gin.Context) {
		message := []byte("Thông báo từ backend: Tin nhắn mới!")
		fmt.Println("Broadcasting message:", string(message))
		m.Broadcast(message)
		c.String(200, "Broadcast message sent!")
	})

}
