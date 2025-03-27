package dto

import (
	"time"
)

type RegisterInput struct {
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required"`
	PhoneNumber string `json:"phoneNumber" binding:"required"`
}

type LoginInput struct {
	Identifier string `json:"identifier" binding:"required"`
	Password   string `json:"password" binding:"required"`
}

type UserLoginResponse struct {
	UserID       uint      `json:"id"`
	UserName     string    `json:"name"`
	UserEmail    string    `json:"email"`
	UserVerified bool      `json:"verified"`
	UserPhone    string    `json:"phone"`
	UserRole     int       `json:"role"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	UserStatus   int       `json:"status"`
	UserAvatar   string    `json:"avatar"`
	UserBanks    []Bank    `json:"banks"`
	Gender       int       `json:"gender"`
	DateOfBirth  string    `json:"dateOfBirth"`
	AdminId      *uint     `json:"adminId"`
	Amount       int64     `json:"amount"`
}

type GoogleUser struct {
	Name          string `json:"name"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verifiedEmail"`
	Picture       string `json:"picture"`
}

type VerifyCodeInput struct {
	Email string `json:"email" binding:"required"`
	Code  string `json:"code" binding:"required"`
}

type ResendVerificationInput struct {
	Identifier string `json:"identifier" binding:"required"`
}

type ForgetPasswordInput struct {
	Identifier string `json:"identifier" binding:"required"`
}
