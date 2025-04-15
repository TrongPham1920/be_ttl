package services

import (
	"context"
	"fmt"
	"time"

	"new/dto"

	"github.com/redis/go-redis/v9"
)

const lastFilterTTL = 2 * time.Hour

func getLastFilterKey(userID int) string {
	return fmt.Sprintf("last_filters:%d", userID)
}

func SaveLastFilters(ctx context.Context, rdb *redis.Client, userID int, filters *dto.SearchFilters) error {
	return SetToRedis(ctx, rdb, getLastFilterKey(userID), filters, lastFilterTTL)
}

func GetLastFilters(ctx context.Context, rdb *redis.Client, userID int) (*dto.SearchFilters, error) {
	var filters dto.SearchFilters
	err := GetFromRedis(ctx, rdb, getLastFilterKey(userID), &filters)
	if err != nil {
		return nil, err
	}
	return &filters, nil
}

func ClearLastFilters(ctx context.Context, rdb *redis.Client, userID int) error {
	return DeleteFromRedis(ctx, rdb, getLastFilterKey(userID))
}

// Merge yêu cầu cũ với yêu cầu mới
func MergeFilters(old *dto.SearchFilters, new *dto.SearchFilters) *dto.SearchFilters {
	new.Type = orIntPointer(new.Type, old.Type)
	new.Province = orString(new.Province, old.Province)
	new.District = orString(new.District, old.District)
	new.Name = orString(new.Name, old.Name)
	new.PriceMax = orIntPointer(new.PriceMax, old.PriceMax)
	new.NumTolet = orIntPointer(new.NumTolet, old.NumTolet)
	new.NumBed = orIntPointer(new.NumBed, old.NumBed)
	new.FromDate = orTimePointer(new.FromDate, old.FromDate)
	new.ToDate = orTimePointer(new.ToDate, old.ToDate)
	new.Status = orIntPointer(new.Status, old.Status)

	// Gộp BenefitIDs
	new.BenefitIDs = mergeUniqueInts(old.BenefitIDs, new.BenefitIDs)

	return new
}

func orString(newVal, oldVal string) string {
	if newVal != "" {
		return newVal
	}
	return oldVal
}

func orIntPointer(newVal, oldVal *int) *int {
	if newVal != nil {
		return newVal
	}
	return oldVal
}

func orStringPointer(newVal, oldVal *string) *string {
	if newVal != nil {
		return newVal
	}
	return oldVal
}

func orTimePointer(newVal, oldVal *time.Time) *time.Time {
	if newVal != nil {
		return newVal
	}
	return oldVal
}

func mergeUniqueInts(a, b []int) []int {
	seen := make(map[int]bool)
	var result []int

	for _, val := range a {
		if !seen[val] {
			seen[val] = true
			result = append(result, val)
		}
	}
	for _, val := range b {
		if !seen[val] {
			seen[val] = true
			result = append(result, val)
		}
	}
	return result
}
