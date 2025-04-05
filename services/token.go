package services

import (
	"encoding/json"
	"new/errors"
	"strings"

	"github.com/dgrijalva/jwt-go"
)

func GetUserIDFromToken(tokenString string) (uint, int, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return 0, 0, errors.NewAppError(errors.ErrCodeInvalidToken, "Token không hợp lệ", nil)
	}

	payload, err := jwt.DecodeSegment(parts[1])
	if err != nil {
		return 0, 0, errors.NewAppError(errors.ErrCodeInvalidToken, "Không thể giải mã token", err)
	}

	claimsMap := jwt.MapClaims{}
	if err := json.Unmarshal(payload, &claimsMap); err != nil {
		return 0, 0, errors.NewAppError(errors.ErrCodeInvalidToken, "Không thể parse token", err)
	}

	userInfo, ok := claimsMap["userinfo"].(map[string]interface{})
	if !ok {
		return 0, 0, errors.NewAppError(errors.ErrCodeInvalidToken, "Không tìm thấy thông tin user trong token", nil)
	}

	userID, okID := userInfo["userid"].(float64)
	if !okID {
		return 0, 0, errors.NewAppError(errors.ErrCodeInvalidToken, "Không tìm thấy ID user trong token", nil)
	}

	role, okRole := userInfo["role"].(float64)
	if !okRole {
		return 0, 0, errors.NewAppError(errors.ErrCodeInvalidToken, "Không tìm thấy role trong token", nil)
	}

	return uint(userID), int(role), nil
}

func GetIDFromToken(tokenString string) (uint, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return 0, errors.NewAppError(errors.ErrCodeInvalidToken, "Token không hợp lệ", nil)
	}

	payload, err := jwt.DecodeSegment(parts[1])
	if err != nil {
		return 0, errors.NewAppError(errors.ErrCodeInvalidToken, "Không thể giải mã token", err)
	}

	claimsMap := jwt.MapClaims{}
	if err := json.Unmarshal(payload, &claimsMap); err != nil {
		return 0, errors.NewAppError(errors.ErrCodeInvalidToken, "Không thể parse token", err)
	}

	userInfo, ok := claimsMap["userinfo"].(map[string]interface{})
	if !ok {
		return 0, errors.NewAppError(errors.ErrCodeInvalidToken, "Không tìm thấy thông tin user trong token", nil)
	}

	userID, okID := userInfo["userid"].(float64)
	if !okID {
		return 0, errors.NewAppError(errors.ErrCodeInvalidToken, "Không tìm thấy ID user trong token", nil)
	}

	return uint(userID), nil
}
