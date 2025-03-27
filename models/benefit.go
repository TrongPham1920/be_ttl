package models

import (
	"fmt"
	"time"
)

type Benefit struct {
	Id        int       `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name"`
	Status    int       `gorm:"default:0" json:"status"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

func (b *Benefit) ValidateStatus() error {
	if b.Status < 0 || b.Status > 1 {
		return fmt.Errorf("invalid Status: %d, must be either 0 or 1", b.Status)
	}
	return nil
}
