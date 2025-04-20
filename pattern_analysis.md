## 1. Factory Pattern

- Mục đích: Tạo instance của service
- Vị trí áp dụng:
  - `services/auth_service.go`:
    - GenerateToken() - Tạo token
    - generateVerificationCode() - Tạo mã xác thực
    - NewAuthService() - Tạo instance AuthService

## 2. Adapter Pattern

- Mục đích: Chuyển đổi interface
- Vị trí áp dụng:
  - `services/auth_service.go`:
    - Claims struct - Chuyển đổi giữa JWT và user info
    - UserInfo struct - Chuyển đổi giữa user và token
  - `services/user_service.go`:
    - UserServiceAdapter - Chuyển đổi interface UserService

## 3. Singleton Pattern

- Mục đích: Quản lý connection
- Vị trí áp dụng:
  - `config/database.go`:
    - GetDB() - Lấy instance database
  - `config/redis.go`:
    - GetRedis() - Lấy instance redis
  - `config/init.go`:
    - InitConfig() - Khởi tạo cấu hình

## 4. Observer Pattern

- Mục đích: Cập nhật tự động
- Vị trí áp dụng:
  - `services/auth_service.go`:
    - UpdateAccommodationRating() - Cập nhật rating
    - ApplyDiscountForUser() - Cập nhật giảm giá
  - `ws_service.go`:
    - WebSocket notifications - Thông báo realtime

## 5. Repository Pattern

- Mục đích: Tách biệt logic truy cập dữ liệu
- Vị trí áp dụng:
  - `services/auth_service.go`:
    - GetUserByEmail() - Lấy user theo email
    - GetUserByPhoneNumber() - Lấy user theo số điện thoại
    - CreateUser() - Tạo user mới
    - UpdateUser() - Cập nhật user

## 6. Service Pattern

- Mục đích: Đóng gói business logic
- Vị trí áp dụng:
  - `services/auth_service.go`:
    - AuthService - Xử lý authentication
  - `services/user_service.go`:
    - UserService - Xử lý user management
  - `services/ws_service.go`:
    - WSService - Xử lý WebSocket
