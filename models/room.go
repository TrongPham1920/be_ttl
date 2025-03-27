package models

import (
	"encoding/json"
	"fmt"
	"time"
)

type Room struct {
	RoomId          uint            `json:"id" gorm:"primaryKey"`
	AccommodationID uint            `json:"accommodationId"`
	RoomName        string          `json:"roomName"`
	Type            uint            `json:"type"`
	NumBed          int             `json:"numBed"`
	NumTolet        int             `json:"numTolet"`
	Acreage         int             `json:"acreage"`
	Price           int             `json:"price"`
	Description     string          `json:"description"`
	CreatedAt       time.Time       `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt       time.Time       `gorm:"autoUpdateTime" json:"updatedAt"`
	Status          int             `json:"status" gorm:"default:0"`
	Avatar          string          `json:"avatar"`
	Img             json.RawMessage `json:"img" gorm:"type:json"`
	Num             int             `json:"num"`
	Furniture       json.RawMessage `json:"furniture" gorm:"type:json"`
	People          int             `json:"people"`
	Parent          Accommodation   `json:"parent" gorm:"foreignKey:AccommodationID"`
	RoomStatuses    []RoomStatus    `gorm:"foreignKey:RoomID"`
}

func (r *Room) ValidateStatus() error {
	if r.Status < 0 || r.Status > 4 {
		return fmt.Errorf("invalid status: %d, must be between 0 and 4", r.Status)
	}
	return nil
}
