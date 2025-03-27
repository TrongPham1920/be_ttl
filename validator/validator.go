package validator

import (
	"encoding/json"
	"new/errors"
	"new/models"
	"regexp"
	"time"
)

// ValidateUser validate thông tin user
func ValidateUser(user *models.User) error {
	if user.Email == "" {
		return errors.NewAppError(errors.ErrCodeRequiredField, "Email không được để trống", nil)
	}

	if !isValidEmail(user.Email) {
		return errors.NewAppError(errors.ErrCodeInvalidEmail, "Email không hợp lệ", nil)
	}

	if user.Password == "" {
		return errors.NewAppError(errors.ErrCodeRequiredField, "Mật khẩu không được để trống", nil)
	}

	if len(user.Password) < 6 {
		return errors.NewAppError(errors.ErrCodeValidation, "Mật khẩu phải có ít nhất 6 ký tự", nil)
	}

	if user.PhoneNumber == "" {
		return errors.NewAppError(errors.ErrCodeRequiredField, "Số điện thoại không được để trống", nil)
	}

	if !isValidPhone(user.PhoneNumber) {
		return errors.NewAppError(errors.ErrCodeInvalidPhone, "Số điện thoại không hợp lệ", nil)
	}

	if user.Role < 0 || user.Role > 2 {
		return errors.NewAppError(errors.ErrCodeInvalidRole, "Role không hợp lệ", nil)
	}

	return nil
}

// ValidateBank validate thông tin ngân hàng
func ValidateBank(bank *models.BankFake) error {
	if bank.BankName == "" {
		return errors.NewAppError(errors.ErrCodeRequiredField, "Tên ngân hàng không được để trống", nil)
	}

	if bank.BankShortName == "" {
		return errors.NewAppError(errors.ErrCodeRequiredField, "Tên viết tắt ngân hàng không được để trống", nil)
	}

	if len(bank.AccountNumbers) == 0 {
		return errors.NewAppError(errors.ErrCodeRequiredField, "Danh sách số tài khoản không được để trống", nil)
	}

	var accountNumbers []string
	if err := json.Unmarshal(bank.AccountNumbers, &accountNumbers); err != nil {
		return errors.NewAppError(errors.ErrCodeInvalidFormat, "Định dạng danh sách số tài khoản không hợp lệ", err)
	}

	for _, account := range accountNumbers {
		if !isValidAccountNumber(account) {
			return errors.NewAppError(errors.ErrCodeInvalidAccount, "Số tài khoản không hợp lệ: "+account, nil)
		}
	}

	return nil
}

// ValidateAmount validate số tiền
func ValidateAmount(amount int64) error {
	if amount < 0 {
		return errors.NewAppError(errors.ErrCodeInvalidAmount, "Số tiền không được âm", nil)
	}
	return nil
}

// isValidEmail kiểm tra email hợp lệ
func isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// isValidPhone kiểm tra số điện thoại hợp lệ
func isValidPhone(phone string) bool {
	phoneRegex := regexp.MustCompile(`^[0-9]{10}$`)
	return phoneRegex.MatchString(phone)
}

// isValidAccountNumber kiểm tra số tài khoản hợp lệ
func isValidAccountNumber(account string) bool {
	accountRegex := regexp.MustCompile(`^[0-9]{10,14}$`)
	return accountRegex.MatchString(account)
}

func ValidateHoliday(holiday *models.Holiday) error {
	if holiday.Name == "" {
		return errors.NewAppError(errors.ErrCodeRequiredField, "Tên ngày nghỉ không được để trống", nil)
	}

	fromDate, err := time.Parse("2006-01-02", holiday.FromDate)
	if err != nil {
		return errors.NewAppError(errors.ErrCodeInvalidFormat, "Định dạng ngày bắt đầu không hợp lệ", err)
	}

	toDate, err := time.Parse("2006-01-02", holiday.ToDate)
	if err != nil {
		return errors.NewAppError(errors.ErrCodeInvalidFormat, "Định dạng ngày kết thúc không hợp lệ", err)
	}

	if toDate.Before(fromDate) {
		return errors.NewAppError(errors.ErrCodeValidation, "Ngày kết thúc phải sau ngày bắt đầu", nil)
	}

	if holiday.Price < 0 {
		return errors.NewAppError(errors.ErrCodeInvalidAmount, "Giá không được âm", nil)
	}

	return nil
}

func ValidateOrder(order *models.Order) error {
	if order.AccommodationID == 0 {
		return errors.NewAppError(errors.ErrCodeRequiredField, "ID chỗ ở không được để trống", nil)
	}

	checkInDate, err := time.Parse("02/01/2006", order.CheckInDate)
	if err != nil {
		return errors.NewAppError(errors.ErrCodeInvalidFormat, "Ngày nhận phòng không hợp lệ", err)
	}

	if checkInDate.Before(time.Now()) {
		return errors.NewAppError(errors.ErrCodeValidation, "Ngày nhận phòng không được nhỏ hơn ngày hiện tại", nil)
	}

	checkOutDate, err := time.Parse("02/01/2006", order.CheckOutDate)
	if err != nil {
		return errors.NewAppError(errors.ErrCodeInvalidFormat, "Ngày trả phòng không hợp lệ", err)
	}

	if checkOutDate.Before(checkInDate) {
		return errors.NewAppError(errors.ErrCodeValidation, "Ngày trả phòng phải sau ngày nhận phòng", nil)
	}

	if order.UserID == nil {
		if order.GuestName == "" {
			return errors.NewAppError(errors.ErrCodeRequiredField, "Tên khách không được để trống", nil)
		}
		if order.GuestPhone == "" {
			return errors.NewAppError(errors.ErrCodeRequiredField, "Số điện thoại khách không được để trống", nil)
		}
		if !isValidPhone(order.GuestPhone) {
			return errors.NewAppError(errors.ErrCodeInvalidPhone, "Số điện thoại khách không hợp lệ", nil)
		}
		if order.GuestEmail != "" && !isValidEmail(order.GuestEmail) {
			return errors.NewAppError(errors.ErrCodeInvalidEmail, "Email khách không hợp lệ", nil)
		}
	}

	return nil
}

func ValidateDiscount(discount *models.Discount) error {
	if discount.Name == "" {
		return errors.NewAppError(errors.ErrCodeRequiredField, "Tên mã giảm giá không được để trống", nil)
	}

	if discount.Discount < 0 || discount.Discount > 100 {
		return errors.NewAppError(errors.ErrCodeInvalidAmount, "Mức giảm giá phải nằm trong khoảng từ 0 đến 100", nil)
	}

	fromDate, err := time.Parse("02/01/2006", discount.FromDate)
	if err != nil {
		return errors.NewAppError(errors.ErrCodeInvalidFormat, "Định dạng ngày bắt đầu không hợp lệ", err)
	}

	toDate, err := time.Parse("02/01/2006", discount.ToDate)
	if err != nil {
		return errors.NewAppError(errors.ErrCodeInvalidFormat, "Định dạng ngày kết thúc không hợp lệ", err)
	}

	if !toDate.After(fromDate) {
		return errors.NewAppError(errors.ErrCodeValidation, "Ngày kết thúc phải sau ngày bắt đầu", nil)
	}

	if discount.Quantity < 0 {
		return errors.NewAppError(errors.ErrCodeInvalidAmount, "Số lượng mã giảm giá không được âm", nil)
	}

	return nil
}

func ValidateRate(rate *models.Rate) error {
	if rate.UserID == 0 {
		return errors.NewAppError(errors.ErrCodeRequiredField, "ID người dùng không được để trống", nil)
	}

	if rate.AccommodationID == 0 {
		return errors.NewAppError(errors.ErrCodeRequiredField, "ID chỗ ở không được để trống", nil)
	}

	if rate.Star < 1 || rate.Star > 5 {
		return errors.NewAppError(errors.ErrCodeValidation, "Số sao đánh giá phải từ 1 đến 5", nil)
	}

	if rate.Comment == "" {
		return errors.NewAppError(errors.ErrCodeRequiredField, "Nội dung đánh giá không được để trống", nil)
	}

	return nil
}

func ValidateInvoice(invoice *models.Invoice) error {
	if invoice.OrderID == 0 {
		return errors.NewAppError(errors.ErrCodeRequiredField, "ID đơn hàng không được để trống", nil)
	}

	if invoice.TotalAmount < 0 {
		return errors.NewAppError(errors.ErrCodeInvalidAmount, "Tổng tiền không được âm", nil)
	}

	return nil
}

// ValidateEmail kiểm tra email hợp lệ
func ValidateEmail(email string) error {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return errors.NewAppError(errors.ErrCodeInvalidEmail, "Email không hợp lệ", nil)
	}
	return nil
}

// ValidatePhone kiểm tra số điện thoại hợp lệ
func ValidatePhone(phone string) error {
	phoneRegex := regexp.MustCompile(`^[0-9]{10}$`)
	if !phoneRegex.MatchString(phone) {
		return errors.NewAppError(errors.ErrCodeInvalidPhone, "Số điện thoại không hợp lệ", nil)
	}
	return nil
}

// ValidatePassword kiểm tra mật khẩu hợp lệ
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return errors.NewAppError(errors.ErrCodeInvalidPassword, "Mật khẩu phải có ít nhất 8 ký tự", nil)
	}
	return nil
}

// ValidateOrderInput kiểm tra dữ liệu đầu vào của order
func ValidateOrderInput(order interface{}) error {
	// TODO: Implement order validation
	return nil
}

// ValidateRoomInput kiểm tra dữ liệu đầu vào của room
func ValidateRoomInput(room interface{}) error {
	// TODO: Implement room validation
	return nil
}
