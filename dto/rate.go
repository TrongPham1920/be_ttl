package dto

import "time"

type RateResponse struct {
	ID              uint      `json:"id"`
	AccommodationID uint      `json:"accommodationId"`
	Comment         string    `json:"comment"`
	Star            int       `json:"star"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
	User            UserInfo  `json:"user"`
}

type RateUpdateResponse struct {
	ID              uint      `json:"id"`
	AccommodationID uint      `json:"accommodationId"`
	Comment         string    `json:"comment"`
	Star            int       `json:"star"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type UserInfo struct {
	ID     uint   `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type CreateRateRequest struct {
	AccommodationID uint   `json:"accommodationId" binding:"required"`
	Comment         string `json:"comment" binding:"required"`
	Star            int    `json:"star" binding:"required,min=1,max=5"`
}

type UpdateRateRequest struct {
	ID              uint   `json:"id" binding:"required"`
	AccommodationID uint   `json:"accommodationId" binding:"required"`
	Comment         string `json:"comment" binding:"required"`
	Star            int    `json:"star" binding:"required,min=1,max=5"`
}
