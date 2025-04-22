package services

import (
	"context"
	"encoding/json"
	"time"

	"new/dto"

	"github.com/redis/go-redis/v9"
)

func SaveLastFilters(ctx context.Context, rdb *redis.Client, key string, filters *dto.SearchFilters) error {
	b, _ := json.Marshal(filters)
	return rdb.Set(ctx, "last_filters:"+key, b, 30*time.Minute).Err()
}

func GetLastFilters(ctx context.Context, rdb *redis.Client, key string) (*dto.SearchFilters, error) {
	val, err := rdb.Get(ctx, "last_filters:"+key).Result()
	if err != nil {
		return nil, err
	}
	var filters dto.SearchFilters
	json.Unmarshal([]byte(val), &filters)
	return &filters, nil
}

func ClearLastFilters(ctx context.Context, rdb *redis.Client, key string) error {
	return rdb.Del(ctx, "last_filters:"+key).Err()
}

// Merge yêu cầu cũ với yêu cầu mới
func MergeFilters(old *dto.SearchFilters, new *dto.SearchFilters) *dto.SearchFilters {
	new.Type = orIntPointer(new.Type, old.Type)
	new.Province = orString(new.Province, old.Province)
	new.District = orString(new.District, old.District)
	new.Name = orString(new.Name, old.Name)
	new.NumTolet = orIntPointer(new.NumTolet, old.NumTolet)
	new.NumBed = orIntPointer(new.NumBed, old.NumBed)
	new.FromDate = orTimePointer(new.FromDate, old.FromDate)
	new.ToDate = orTimePointer(new.ToDate, old.ToDate)
	new.Status = orIntPointer(new.Status, old.Status)

	// Gộp BenefitIDs
	new.BenefitIDs = mergeUniqueInts(old.BenefitIDs, new.BenefitIDs)

	//Xử lý case người dùng nhập lại PriceMax và PriceMin
	if new.PriceMin != nil && old.PriceMax != nil && *new.PriceMin > *old.PriceMax {
		new.PriceMax = nil
	} else {
		new.PriceMax = orIntPointer(new.PriceMax, old.PriceMax)
	}

	if new.PriceMax != nil && old.PriceMin != nil && *new.PriceMax < *old.PriceMin {
		new.PriceMin = nil
	} else {
		new.PriceMin = orIntPointer(new.PriceMin, old.PriceMin)
	}
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
