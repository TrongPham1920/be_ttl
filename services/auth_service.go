package services

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net/smtp"
	"new/config"
	"new/models"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserInfo struct {
	UserId uint `json:"userid"`
	Role   int  `json:"role"`
}

type Claims struct {
	UserInfo UserInfo `json:"userinfo"`
	jwt.StandardClaims
}

func generateVerificationCode() (string, error) {
	code := ""

	for i := 0; i < 6; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		code += n.String()
	}

	return code, nil
}

func sendVerificationEmail(email string, token string) error {
	from := "takieulong@gmail.com"
	password := "audj brda qhbq lpxu"

	host := "smtp.gmail.com"
	port := "587"
	to := []string{email}
	subject := "Subject: Mã dùng một lần của bạn\n"
	body := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8">
			<title>Mã xác thực</title>
		</head>
		<body>
			<p>Xin chào %s,</p>
			<p>Chúng tôi đã nhận yêu cầu mã dùng một lần để dùng cho tài khoản của bạn.</p>
			<p>Mã dùng một lần của bạn là: <strong>%s</strong></p>
			<p>Nếu không yêu cầu mã này thì bạn có thể bỏ qua email này một cách an toàn. Có thể ai đó khác đã nhập địa chỉ email của bạn do nhầm lẫn.</p>
			<p>Bạn có thể bấm vào nút sau để xác nhận tài khoản</p>
			<p>
				<a href="https://trothalo.click/verify-email?token=%s" style="display: inline-block; padding: 10px 20px; background-color: #1a73e8; color: white; text-decoration: none; border-radius: 5px;">
					Xác nhận email
				</a>
			</p>
			<p>Xin cám ơn,<br>Nhóm tài khoản</p>
		</body>
		</html>
	`, email, token, token)

	msg := []byte("MIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n" + subject + "\n" + body)

	auth := smtp.PlainAuth("", from, password, host)

	err := smtp.SendMail(host+":"+port, auth, from, to, msg)
	return err
}

func sendcodeEmail(email string, token string) error {
	from := "takieulong@gmail.com"
	password := "audj brda qhbq lpxu"

	host := "smtp.gmail.com"
	port := "587"
	to := []string{email}
	subject := "Subject: Mã đăng nhập\n"
	body := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8">
			<title>Mã xác thực</title>
		</head>
		<body>
			<p>Xin chào %s,</p>
			<p>Chúng tôi đã nhận yêu cầu mã dùng một lần để dùng cho tài khoản của bạn.</p>
			<p>Mã đăng nhập là: <strong>%s</strong></p>
			<p>Nếu không yêu cầu mã này thì bạn có thể bỏ qua email này một cách an toàn. Có thể ai đó khác đã nhập địa chỉ email của bạn do nhầm lẫn.</p>
			<p>Bạn có thể bấm vào nút sau để xác nhận tài khoản</p>
			<p>Xin cám ơn,<br>Nhóm tài khoản</p>
		</body>
		</html>
	`, email, token)

	msg := []byte("MIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n" + subject + "\n" + body)

	auth := smtp.PlainAuth("", from, password, host)

	err := smtp.SendMail(host+":"+port, auth, from, to, msg)
	return err
}

func sendUserEmail(email string, phone string, pass string) error {
	from := "takieulong@gmail.com"
	password := "audj brda qhbq lpxu"

	host := "smtp.gmail.com"
	port := "587"
	to := []string{email}
	subject := "Subject: Bạn đã tạo tài khoản mới\n"
	body := fmt.Sprintf(`<!DOCTYPE html>
	<html>
	<head>
		<meta charset="UTF-8">
		<title>Tạo tài khoản thành công</title>
	</head>
	<body>
		<p>Xin chào,</p>
		<p>Chúc mừng! Bạn đã tạo tài khoản thành công.</p>
		<p>Thông tin tài khoản của bạn như sau:</p>
		<ul>
			<li>Email: <strong>%s</strong></li>
			<li>Số điện thoại: <strong>%s</strong></li>
			<li>Mật khẩu: <strong>%s</strong></li>
		</ul>
		<p>Nếu không yêu cầu tạo tài khoản này thì bạn có thể bỏ qua email này một cách an toàn. Có thể ai đó khác đã nhập địa chỉ email của bạn do nhầm lẫn.</p>
		<p>Xin cảm ơn,<br>Nhóm tài khoản</p>
	</body>
	</html>`, email, phone, pass)

	msg := []byte("MIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n" + subject + "\n" + body)

	auth := smtp.PlainAuth("", from, password, host)

	err := smtp.SendMail(host+":"+port, auth, from, to, msg)
	return err
}

func SendOrderEmail(email string, orderId uint, totalPrice float64, checkInDate string, checkOutDate string) error {
	from := "takieulong@gmail.com"
	password := "audj brda qhbq lpxu"

	host := "smtp.gmail.com"
	port := "587"
	to := []string{email}
	subject := "Subject: Đặt đơn hàng thành công\n"

	priceFormatted := formatCurrency(totalPrice)

	body := fmt.Sprintf(`<!DOCTYPE html>
	<html>
	<head>
		<meta charset="UTF-8">
		<title>Đặt đơn hàng thành công</title>
	</head>
	<body>
		<p>Xin chào,</p>
		<p>Chúc mừng! Bạn đã đặt đơn hàng thành công.</p>
		<p>Thông tin đơn hàng của bạn như sau:</p>
		<ul>
			<li>Mã đơn hàng: <strong>%d</strong></li>
			<li>Ngày nhận phòng: <strong>%s</strong></li>
			<li>Ngày trả phòng: <strong>%s</strong></li>
			<li>Tổng giá trị đơn hàng: <strong>%s VND</strong></li>
		</ul>
		<p>Chúng tôi sẽ gửi cho bạn thông tin chi tiết về đơn hàng khi có sự thay đổi.</p>
		<p>Cảm ơn bạn đã sử dụng dịch vụ của chúng tôi!</p>
		<p>Xin cảm ơn,<br>Nhóm hỗ trợ</p>
	</body>
	</html>`, orderId, checkInDate, checkOutDate, priceFormatted)

	msg := []byte("MIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n" + subject + "\n" + body)

	auth := smtp.PlainAuth("", from, password, host)

	err := smtp.SendMail(host+":"+port, auth, from, to, msg)
	return err
}

func formatCurrency(amount float64) string {
	return fmt.Sprintf("%0.2f", amount)
}

func GetUserByEmail(email string) (models.User, error) {
	var user models.User
	result := config.DB.Where("email = ?", email).First(&user)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return user, fmt.Errorf("không tìm thấy người dùng với email %s", email)
	}

	if result.Error != nil {
		return user, result.Error
	}

	return user, nil
}

func GetUserByPhoneNumber(phoneNumber string) (models.User, error) {
	var user models.User
	result := config.DB.Where("phone_number = ?", phoneNumber).First(&user)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return user, fmt.Errorf("không tìm thấy người dùng với số điện thoại %s", phoneNumber)
	}

	if result.Error != nil {
		return user, result.Error
	}

	return user, nil
}

func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

var secretKey = []byte(config.GetEnv("SECRET_KEY_ACCESS_TOKEN"))
var refreshSecretKey = []byte(config.GetEnv("SECRET_KEY_REFRESH_TOKEN"))

func GenerateToken(userInfo UserInfo, expiryMinutes int, isAccessToken bool) (string, error) {
	claims := &Claims{
		UserInfo: userInfo,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Minute * time.Duration(expiryMinutes)).Unix(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	var secretKeyToUse []byte
	if isAccessToken {
		secretKeyToUse = secretKey
	} else {
		secretKeyToUse = refreshSecretKey
	}

	return token.SignedString(secretKeyToUse)
}

func SetTokenCookies(c *gin.Context, accessToken string) {
	c.SetCookie(
		"access_token",
		accessToken,
		3*24*60*60,
		"/",
		"",
		true,
		false,
	)

}

func CreateUser(input models.User) (models.User, error) {
	if input.Email == "" || input.Password == "" || input.PhoneNumber == "" {
		return models.User{}, errors.New("không được để trống email, password, phone")
	}

	existingEmail, err := GetUserByEmail(input.Email)
	if err == nil {
		return models.User{}, fmt.Errorf("email %s đã được sử dụng", existingEmail.Email)
	}

	existingPhone, err := GetUserByPhoneNumber(input.PhoneNumber)
	if err == nil {
		return models.User{}, fmt.Errorf("số điện thoại %s đã được sử dụng", existingPhone.PhoneNumber)
	}

	hashedPassword, err := HashPassword(input.Password)
	if err != nil {
		return models.User{}, err
	}

	token, err := generateVerificationCode()
	if err != nil {
		return models.User{}, err
	}

	user := models.User{
		Email:         input.Email,
		Password:      hashedPassword,
		PhoneNumber:   input.PhoneNumber,
		IsVerified:    false,
		Code:          token,
		CodeCreatedAt: time.Now(),
		Role:          input.Role,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		Name:          input.Name,
		Amount:        input.Amount,
	}

	result := config.DB.Create(&user)
	if result.Error != nil {
		return user, result.Error
	}

	if user.Role != 0 {
		err = sendVerificationEmail(input.Email, token)
	} else {
		err = sendUserEmail(input.Email, input.PhoneNumber, input.Password)
	}

	if err != nil {
		return user, err
	}

	return user, nil
}

func RegenerateVerificationCode(userID uint) error {

	var user models.User
	result := config.DB.First(&user, userID)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return fmt.Errorf("không tìm thấy người dùng với ID %d", userID)
	}

	if result.Error != nil {
		return result.Error
	}

	newCode, err := generateVerificationCode()
	if err != nil {
		return fmt.Errorf("không thể tạo mã xác minh mới: %v", err)
	}

	user.Code = newCode
	user.CodeCreatedAt = time.Now()

	if err := config.DB.Save(&user).Error; err != nil {
		return fmt.Errorf("không thể cập nhật mã xác minh: %v", err)
	}
	err = sendVerificationEmail(user.Email, newCode)
	if err != nil {
		return fmt.Errorf("không thể gửi email xác minh: %v", err)
	}

	return nil
}

func ResetPass(user models.User) error {

	newCode, err := generateVerificationCode()
	if err != nil {
		return fmt.Errorf("không thể tạo mã xác minh mới: %v", err)
	}

	user.Code = newCode
	user.CodeCreatedAt = time.Now()

	if err := config.DB.Save(&user).Error; err != nil {
		return fmt.Errorf("không thể cập nhật mã xác minh: %v", err)
	}

	err = sendVerificationEmail(user.Email, newCode)
	if err != nil {
		return fmt.Errorf("không thể gửi email xác minh: %v", err)
	}

	return nil
}

func sendNews(email string, title string, mess string) error {
	from := "takieulong@gmail.com"
	password := "audj brda qhbq lpxu"

	host := "smtp.gmail.com"
	port := "587"
	to := []string{email}
	subject := fmt.Sprintf("Subject: %s\n", title)
	body := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8">
			<title>%s</title>
		</head>
		<body>
			<p>Xin chào %s,</p>
			<p>%s</p>
			<p>Xin cám ơn,<br>Nhóm tài khoản</p>
		</body>
		</html>
	`, title, email, mess)

	msg := []byte("MIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n" + subject + "\n" + body)

	auth := smtp.PlainAuth("", from, password, host)

	err := smtp.SendMail(host+":"+port, auth, from, to, msg)
	return err
}

func NewPass(user models.User, newPassword string) error {

	hashedPassword, err := HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("không thể băm mật khẩu: %v", err)
	}

	user.Password = hashedPassword

	if err := config.DB.Save(&user).Error; err != nil {
		return fmt.Errorf("không thể cập nhật mật khẩu mới: %v", err)
	}

	err = sendNews(user.Email, "Đổi mật khẩu", "Mật khẩu của bạn đã được cập nhật thành công.")
	if err != nil {
		return fmt.Errorf("không thể gửi email xác nhận: %v", err)
	}

	return nil
}

func UpdateAccommodationRating(accommodationId uint) error {

	var rates []models.Rate
	if err := config.DB.Where("accommodation_id = ?", accommodationId).Find(&rates).Error; err != nil {
		return err
	}

	var totalStars int
	var totalCount int

	for _, rate := range rates {
		totalStars += rate.Star
		totalCount++
	}

	var average float64
	if totalCount > 0 {
		average = float64(totalStars) / float64(totalCount)
	}

	if err := config.DB.Model(&models.Accommodation{}).
		Where("id = ?", accommodationId).
		Update("num", average).Error; err != nil {
		return err
	}

	return nil
}

func CreateGoogleUser(name, email, avatar string) (models.User, error) {

	existingEmail, err := GetUserByEmail(email)
	if err == nil {
		return models.User{}, fmt.Errorf("email %s đã được sử dụng", existingEmail.Email)
	}

	user := models.User{
		Name:       name,
		Email:      email,
		Password:   "",
		Avatar:     avatar,
		IsVerified: true,
		Role:       0,
	}

	result := config.DB.Create(&user)
	if result.Error != nil {
		return user, result.Error
	}

	return user, nil
}

func ApplyDiscountForUser(user models.User) (float64, error) {
	var discounts []models.Discount
	var userDiscounts []models.UserDiscount

	if err := config.DB.Where("status = ? AND quantity > 0 ", 1).Order("discount DESC").Find(&discounts).Error; err != nil {
		return 0, fmt.Errorf("Không thể lấy danh sách mã giảm giá: %v", err)
	}

	if err := config.DB.Where("user_id = ?", user.ID).Find(&userDiscounts).Error; err != nil {
		return 0, fmt.Errorf("Lỗi khi kiểm tra lịch sử sử dụng mã giảm giá: %v", err)
	}

	userDiscountUsage := make(map[uint]int)
	for _, userDiscount := range userDiscounts {
		userDiscountUsage[userDiscount.DiscountID] = userDiscount.UsageCount
	}

	var applicableDiscount models.Discount

	for _, discount := range discounts {
		if discount.ID == 1 {
			applicableDiscount = discount
			break
		}
		if usageCount, used := userDiscountUsage[discount.ID]; !used || usageCount < discount.Quantity {
			applicableDiscount = discount
			break
		}
	}

	if applicableDiscount.ID == 0 {
		return 0, nil
	}

	var userDiscount models.UserDiscount
	if err := config.DB.Where("user_id = ? AND discount_id = ?", user.ID, applicableDiscount.ID).First(&userDiscount).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, fmt.Errorf("Lỗi khi kiểm tra lịch sử sử dụng mã giảm giá: %v", err)
	}

	if userDiscount.ID == 0 {
		userDiscount.UserID = user.ID
		userDiscount.DiscountID = applicableDiscount.ID
		userDiscount.UsageCount = 1
	} else {
		userDiscount.UsageCount += 1
	}

	if err := config.DB.Save(&userDiscount).Error; err != nil {
		return 0, fmt.Errorf("Không thể cập nhật thông tin sử dụng mã giảm giá: %v", err)
	}

	return float64(applicableDiscount.Discount), nil
}

func CheckUserEligibilityForDiscount(userID uint) bool {
	if userID == 0 {
		return false
	}

	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		return false
	}

	return true
}

func SendPayEmail(email string, vat, vatLastMonth, totalVat int, qrCodeURL string) error {
	from := "takieulong@gmail.com"
	password := "audj brda qhbq lpxu"

	host := "smtp.gmail.com"
	port := "587"
	to := []string{email}
	subject := "Subject: Thông báo phí thường niên\r\n"

	emailContent := fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<head>
		<meta charset="UTF-8">
		<title>Thông báo nhắc đóng phí</title>
	</head>
	<body>
		<p>Xin chào bạn,</p>
		<p>Đây là thông báo nhắc nhở bạn hoàn thành việc đóng phí đúng hẹn.</p>
		<p><strong>Thông tin doanh thu của bạn:</strong></p>
		<ul>
			<li>VAT hiện tại: <strong>%d</strong></li>
			<li>VAT tháng trước: <strong>%d</strong></li>
			<li><strong>Tổng số thanh toán:</strong> <span style="color: red; font-size: 20px; font-weight: bold;">%d</span></li>
		</ul>
		<p>Bạn vui lòng quét mã QR bên dưới để hoàn tất thanh toán:</p>
		<p>
			<img alt="QR Code for Payment" src="%s" width="400">
		</p>
		<p><strong>Thông tin tài khoản ngân hàng:</strong></p>
		<p>Số tài khoản: SACOMBANK - 060915374450</p>
		<p><strong>Vui lòng kiểm tra và hoàn tất thanh toán theo số tài khoản trên.</strong></p>
		<p>Chúng tôi rất cảm ơn bạn đã sử dụng dịch vụ của chúng tôi.</p>
		<p>Trân trọng,<br>Nhóm hỗ trợ</p>
	</body>
	</html>
	`, vat, vatLastMonth, totalVat, qrCodeURL)

	msg := []byte("MIME-Version: 1.0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		subject + "\r\n" +
		emailContent)

	auth := smtp.PlainAuth("", from, password, host)

	err := smtp.SendMail(host+":"+port, auth, from, to, msg)
	return err
}
