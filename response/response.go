package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response định nghĩa cấu trúc response
type Response struct {
	Code       int         `json:"code"`
	Mess       string      `json:"mess"`
	Data       interface{} `json:"data,omitempty"`
	Pagination *Pagination `json:"pagination,omitempty"`
}

// Pagination định nghĩa cấu trúc phân trang
type Pagination struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Total int `json:"total"`
}

type ResponseTotal struct {
	Code       int         `json:"code"`
	Mess       string      `json:"mess"`
	Data       interface{} `json:"data,omitempty"`
	Total      int         `json:"total"`
	Pagination *Pagination `json:"pagination,omitempty"`
}

// Success trả về response thành công
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code: 1,
		Mess: "Thành công",
		Data: data,
	})
}

func SuccessWithTotal(c *gin.Context, data interface{}, total int) {
	c.JSON(http.StatusOK, ResponseTotal{
		Code:  1,
		Mess:  "Thành công",
		Total: total,
		Data:  data,
	})
}

// SuccessWithPagination trả về response thành công có phân trang
func SuccessWithPagination(c *gin.Context, data interface{}, page, limit, total int) {
	c.JSON(http.StatusOK, Response{
		Code: 1,
		Mess: "Thành công",
		Data: data,
		Pagination: &Pagination{
			Page:  page,
			Limit: limit,
			Total: total,
		},
	})
}

// Error trả về response lỗi
func Error(c *gin.Context, code int, message string) {
	c.JSON(http.StatusBadRequest, Response{
		Code: code,
		Mess: message,
	})
}

// ServerError trả về response lỗi server
func ServerError(c *gin.Context) {
	c.JSON(http.StatusInternalServerError, Response{
		Code: 0,
		Mess: "Lỗi server",
	})
}

// Unauthorized trả về response chưa xác thực
func Unauthorized(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, Response{
		Code: 0,
		Mess: "Chưa xác thực",
	})
}

// Forbidden trả về response không có quyền
func Forbidden(c *gin.Context) {
	c.JSON(http.StatusForbidden, Response{
		Code: 0,
		Mess: "Không có quyền truy cập",
	})
}

// NotFound trả về response không tìm thấy
func NotFound(c *gin.Context) {
	c.JSON(http.StatusNotFound, Response{
		Code: 0,
		Mess: "Không tìm thấy",
	})
}

// ValidationError trả về response lỗi validation
func ValidationError(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, Response{
		Code: 0,
		Mess: message,
	})
}

// BadRequest trả về response lỗi bad request
func BadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, Response{
		Code: 0,
		Mess: message,
	})
}

// Conflict trả về response conflict (409)
func Conflict(c *gin.Context) {
	c.JSON(http.StatusConflict, Response{
		Code: 0,
		Mess: "Xung đột dữ liệu",
	})
}
