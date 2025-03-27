package dto

import (
	"encoding/json"
	"time"
)

type RoomRequest struct {
	RoomId       uint            `json:"id"`
	RoomName     string          `json:"roomName"`
	Type         uint            `json:"type"`
	NumBed       int             `json:"numBed"`
	NumTolet     int             `json:"numTolet"`
	Acreage      int             `json:"acreage"`
	Price        int             `json:"price"`
	DaysPrice    json.RawMessage `json:"daysPrice"`
	HolidayPrice json.RawMessage `json:"holidayPrice"`
	Description  string          `json:"description"`
	Status       int             `json:"status"`
	Avatar       string          `json:"avatar"`
	Img          json.RawMessage `json:"img"`
	Num          int             `json:"num"`
	Furniture    json.RawMessage `json:"furniture" gorm:"type:json"`
	People       int             `json:"people"`
}

// DayPrice là DTO cho giá theo ngày
type DayPrice struct {
	Day   string `json:"day"`
	Price int    `json:"price"`
}

type RoomResponse struct {
	RoomId    uint      `json:"id"`
	RoomName  string    `json:"roomName"`
	Type      uint      `json:"type"`
	NumBed    int       `json:"numBed"`
	NumTolet  int       `json:"numTolet"`
	Acreage   int       `json:"acreage"`
	Price     int       `json:"price"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Status    int       `json:"status"`
	Avatar    string    `json:"avatar"`
	People    int       `json:"people"`
	Parents   Parents   `json:"parents"`
}

// Parents là DTO cho thông tin cha của room
type Parents struct {
	Id   uint   `json:"id"`
	Name string `json:"name"`
}

// RoomDetail là DTO cho thông tin chi tiết của room
type RoomDetail struct {
	RoomId      uint            `json:"id" gorm:"primaryKey"`
	RoomName    string          `json:"roomName"`
	Type        uint            `json:"type"`
	NumBed      int             `json:"numBed"`
	NumTolet    int             `json:"numTolet"`
	Acreage     int             `json:"acreage"`
	Price       int             `json:"price"`
	Description string          `json:"description"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
	Status      int             `json:"status"`
	Avatar      string          `json:"avatar"`
	Img         json.RawMessage `json:"img" gorm:"type:json"`
	Num         int             `json:"num"`
	Furniture   json.RawMessage `json:"furniture" gorm:"type:json"`
	People      int             `json:"people"`
	Parent      Parents         `json:"parent"`
}

// CreateRoomRequest là DTO cho request tạo room
type CreateRoomRequest struct {
	AccommodationID uint   `json:"accommodationId" binding:"required"`
	RoomName        string `json:"roomName" binding:"required"`
	Price           int    `json:"price" binding:"required"`
	Description     string `json:"description"`
	Image           string `json:"image"`
}

// RoomStatusRequest là DTO cho request cập nhật trạng thái room
type RoomStatusRequest struct {
	Status int `json:"status" binding:"required"`
}

// RoomListResponse là DTO cho response danh sách room
type RoomListResponse struct {
	Data  []RoomResponse `json:"data"`
	Page  int            `json:"page"`
	Limit int            `json:"limit"`
	Total int64          `json:"total"`
}
