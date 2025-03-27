package models

import "time"

type Province struct {
	ProvinceId   int       `json:"provinceId" gorm:"primaryKey"`
	ProvinceName string    `json:"provinceName"`
	ProvinceImg  string    `json:"provinceImg"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}
