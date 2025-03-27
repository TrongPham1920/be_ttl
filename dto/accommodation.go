package dto

import (
	"encoding/json"
	"new/models"
	"time"
)

type AccommodationRequest struct {
	ID               uint             `json:"id"`
	Type             int              `json:"type"`
	Name             string           `json:"name"`
	Address          string           `json:"address"`
	Avatar           string           `json:"avatar"`
	Img              json.RawMessage  `json:"img" gorm:"type:json"`
	ShortDescription string           `json:"shortDescription"`
	Description      string           `json:"description"`
	Status           int              `json:"status"`
	Num              int              `json:"num"`
	Furniture        json.RawMessage  `json:"furniture" gorm:"type:json"`
	Benefits         []models.Benefit `json:"benefits" gorm:"many2many:accommodation_benefits;"`
	People           int              `json:"people"`
	Price            int              `json:"price"`
	TimeCheckOut     string           `json:"timeCheckOut"`
	TimeCheckIn      string           `json:"timeCheckIn"`
	Province         string           `json:"province"`
	District         string           `json:"district"`
	Ward             string           `json:"ward"`
	Longitude        float64          `json:"longitude"`
	Latitude         float64          `json:"latitude"`
}

type Actor struct {
	Name          string `json:"name"`
	Email         string `json:"email"`
	PhoneNumber   string `json:"phoneNumber"`
	BankName      string `json:"bankName"`
	AccountNumber string `json:"accountNumber"`
	BankShortName string `json:"bankShortName"`
}

type AccommodationResponse struct {
	ID               uint   `json:"id"`
	Type             int    `json:"type"`
	Province         string `json:"province"`
	Name             string `json:"name"`
	Address          string `json:"address"`
	CreateAt         time.Time
	UpdateAt         time.Time
	Avatar           string           `json:"avatar"`
	ShortDescription string           `json:"shortDescription"`
	Status           int              `json:"status"`
	Num              int              `json:"num"`
	People           int              `json:"people"`
	Price            int              `json:"price"`
	NumBed           int              `json:"numBed"`
	NumTolet         int              `json:"numTolet"`
	District         string           `json:"district"`
	Ward             string           `json:"ward"`
	Longitude        float64          `json:"longitude"`
	Latitude         float64          `json:"latitude"`
	Benefits         []models.Benefit `json:"benefits"`
}

type AccommodationResponseTest struct {
	ID       uint             `json:"id"`
	Type     int              `json:"type"`
	Province string           `json:"province"`
	Name     string           `json:"name"`
	Status   int              `json:"status"`
	Num      int              `json:"num"`
	People   int              `json:"people"`
	Price    int              `json:"price"`
	NumBed   int              `json:"numBed"`
	NumTolet int              `json:"numTolet"`
	District string           `json:"district"`
	Ward     string           `json:"ward"`
	Benefits []models.Benefit `json:"benefits"`
}

type AccommodationDetailResponse struct {
	ID               uint   `json:"id"`
	Type             int    `json:"type"`
	Province         string `json:"province"`
	District         string `json:"district"`
	Ward             string `json:"ward"`
	Name             string `json:"name"`
	Address          string `json:"address"`
	CreateAt         time.Time
	UpdateAt         time.Time
	Avatar           string           `json:"avatar"`
	ShortDescription string           `json:"shortDescription"`
	Description      string           `json:"description"`
	Status           int              `json:"status"`
	User             Actor            `json:"user"`
	Num              int              `json:"num"`
	People           int              `json:"people"`
	Price            int              `json:"price"`
	NumBed           int              `json:"numBed"`
	NumTolet         int              `json:"numTolet"`
	Furniture        json.RawMessage  `json:"furniture" gorm:"type:json"`
	Img              json.RawMessage  `json:"img"`
	Benefits         []models.Benefit `json:"benefits"`
	Rates            []RateResponse   `json:"rates"`
	TimeCheckOut     string           `json:"timeCheckOut"`
	TimeCheckIn      string           `json:"timeCheckIn"`
	Longitude        float64          `json:"longitude"`
	Latitude         float64          `json:"latitude"`
}

type ScoredAccommodation struct {
	Accommodation models.Accommodation `json:"accommodation"`
	Score         int                  `json:"score"`
}
