package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
)

// GeocodingResponseGoong định nghĩa cấu trúc phản hồi từ Goong
type GeocodingResponseGoong struct {
	Results []struct {
		FormattedAddress string `json:"formatted_address"`
		Geometry        struct {
			Location struct {
				Lat float64 `json:"lat"`
				Lng float64 `json:"lng"`
			} `json:"location"`
		} `json:"geometry"`
	} `json:"results"`
}

// GetBestCoordinatesFromResponseGoong lấy tọa độ từ phản hồi API Goong
func GetBestCoordinatesFromResponseGoong(body io.Reader) (float64, float64, error) {
	var response GeocodingResponseGoong
	if err := json.NewDecoder(body).Decode(&response); err != nil {
		return 0, 0, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(response.Results) == 0 {
		return 0, 0, errors.New("no results found")
	}

	bestResult := response.Results[0] // Chọn kết quả đầu tiên
	lat, lng := bestResult.Geometry.Location.Lat, bestResult.Geometry.Location.Lng
	return lat, lng, nil
}

// GetCoordinatesFromAddress sử dụng API Goong để lấy tọa độ
func GetCoordinatesFromAddress(address, district, province, ward, goongAPIKey string) (float64, float64, error) {
	fullAddress := fmt.Sprintf("%s, %s, %s, %s", address, ward, district, province)
	encodedAddress := url.QueryEscape(fullAddress)
	log.Println("encodedAddress:", encodedAddress)

	apiURL := fmt.Sprintf(
		"https://rsapi.goong.io/geocode?address=%s&api_key=%s",
		encodedAddress,
		goongAPIKey,
	)

	resp, err := http.Get(apiURL)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	return GetBestCoordinatesFromResponseGoong(resp.Body)
}
