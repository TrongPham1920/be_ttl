package types

// InvoiceUserResponse là DTO cho thông tin user trong invoice
type InvoiceUserResponse struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phoneNumber"`
}
