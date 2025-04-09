package dto

import (
	"time"
)

type SearchFilters struct {
	Name       string
	Province   string
	District   string
	Ward       string
	Type       *int
	Status     *int
	People     *int
	NumBed     *int
	NumTolet   *int
	PriceMin   *int
	PriceMax   *int
	BenefitIDs []int
	FromDate   *time.Time
	ToDate     *time.Time
	Page       int
	Limit      int
}
