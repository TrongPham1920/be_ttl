ENV=dev || qc để test
PORT=8083

REDIS_ADDR=13.214.89.85:6379
REDIS_USER=default  
REDIS_PASSWORD=

SECRET_KEY_ACCESS_TOKEN=access_token_secret
SECRET_KEY_REFRESH_TOKEN=refresh_token_secret

DEV_DB_HOST=13.214.89.85
DEV_DB_PORT=5432
DEV_DB_USER=postgres
DEV_DB_PASSWORD=admin
DEV_DB_NAME=ttl_db

MAPBOX_KEY=pk.eyJ1IjoidGFraWV1bG9uZyIsImEiOiJjbTNyYXR0Y3IwM2xjMmpzY2tsdXB1bDg1In0.N2Rp_nzqe3bZKvE6gQL-tw

QC_DB_HOST=13.214.89.85
QC_DB_PORT=5432
QC_DB_USER=postgres
QC_DB_PASSWORD=admin
QC_DB_NAME=test_db

# API Trothalo

API hệ thống đặt lưu trú

## Cấu Trúc Thư Mục

```
api_trothalo/
├── config/         # Cấu hình hệ thống
├── controllers/    # Xử lý request/response
├── models/         # Định nghĩa cấu trúc dữ liệu
├── services/       # Xử lý logic nghiệp vụ
├── middleware/     # Middleware xử lý request
├── routes/         # Định nghĩa routes
├── validator/      # Validate dữ liệu
├── errors/         # Xử lý lỗi
├── dto/            # Data Transfer Objects
├── utils/          # Các hàm tiện ích
└── docs/           # Tài liệu
```

## Cơ Chế Xử Lý Request

### Luồng Xử Lý

1. Request → Middleware → Controller → Service → Model → Database
2. Response ← Controller ← Service ← Model ← Database

### Middleware

- Authentication: Kiểm tra token
- Authorization: Kiểm tra quyền
- Validation: Kiểm tra dữ liệu đầu vào
- Logging: Ghi log request/response

### Controller

- Nhận request từ client
- Validate dữ liệu đầu vào
- Gọi service xử lý logic
- Trả về response

### Service

- Xử lý logic nghiệp vụ
- Tương tác với database
- Cache dữ liệu (nếu cần)

## Cơ Chế Validate

### Validate Dữ Liệu

```go
// Ví dụ validate user
func ValidateUser(user *models.User) error {
    if user.Email == "" {
        return errors.NewAppError(errors.ErrCodeRequiredField, "Email không được để trống", nil)
    }
    // ... các validation khác
}
```

### Các Loại Validate

- Required: Kiểm tra trường bắt buộc
- Format: Kiểm tra định dạng (email, phone, ...)
- Range: Kiểm tra giá trị trong khoảng
- Custom: Validate tùy chỉnh

## Cơ Chế Xử Lý Lỗi

### Error Code

```go
const (
    ErrCodeRequiredField = "REQUIRED_FIELD"
    ErrCodeInvalidFormat = "INVALID_FORMAT"
    ErrCodeValidation    = "VALIDATION_ERROR"
    // ...
)
```

### Error Message

- Tất cả message lỗi bằng tiếng Việt
- Message rõ ràng, dễ hiểu
- Bao gồm thông tin chi tiết về lỗi

### Error Response

```json
{
  "status": "error",
  "code": "REQUIRED_FIELD",
  "message": "Email không được để trống",
  "data": null
}
```

## Cơ Chế Cache

### Cache Strategy

- Cache-aside pattern
- Cache dữ liệu thường xuyên truy cập
- Tự động xóa cache khi có thay đổi

### Cache Key

- Format: `{module}:{action}:{id}`
- Ví dụ: `user:get:123`

## Cơ Chế Bảo Mật

### Authentication

- JWT token
- Token refresh
- Token blacklist

### Authorization

- Role-based access control
- Permission-based access control

## Cơ Chế Logging

### Log Levels

- INFO: Thông tin thông thường
- ERROR: Lỗi hệ thống
- DEBUG: Thông tin debug

### Log Format

```
[LEVEL] [TIME] [FILE:LINE] MESSAGE
```

## Cơ Chế Database

### Connection

- Singleton pattern
- Connection pool
- Retry mechanism

### Transaction

- ACID properties
- Rollback on error
- Commit on success

## Cơ Chế Testing

### Unit Test

- Test từng function
- Mock dependencies
- Test edge cases

### Integration Test

- Test API endpoints
- Test database operations
- Test cache operations

## Cơ Chế Deployment

### Environment

- Development
- Testing
- Production

### Configuration

- Environment variables
- Config files
- Secret management

## Best Practices

### Code Style

- Clean code
- SOLID principles
- DRY (Don't Repeat Yourself)

### Performance

- Optimize database queries
- Use appropriate indexes
- Implement caching

### Security

- Input validation
- Output encoding
- SQL injection prevention

## Cài Đặt và Chạy

### Yêu Cầu

- Go 1.23.0 trở lên
- PostgreSQL
- Redis

### Cài Đặt

1. Clone repository

```bash
git clone https://github.com/yourusername/api_trothalo.git
```

2. Cài đặt dependencies

```bash
go mod download
```

3. Cấu hình môi trường

```bash
cp .env.example .env
# Chỉnh sửa các biến môi trường trong file .env
```

4. Chạy ứng dụng

```bash
go run main.go
```

### API Documentation

Xem tài liệu API tại: `/docs/api.md`

## Đóng Góp

Mọi đóng góp đều được chào đón. Vui lòng tạo issue hoặc pull request.

## License

MIT License
