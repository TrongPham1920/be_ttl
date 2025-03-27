package dto

// UpdateBenefitRequest là DTO cho yêu cầu cập nhật benefit
type UpdateBenefitRequest struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// CreateBenefitRequest là DTO cho yêu cầu tạo mới benefit
type CreateBenefitRequest struct {
	Name string `json:"name" binding:"required"`
}

// ChangeBenefitStatusRequest là DTO cho yêu cầu thay đổi trạng thái benefit
type ChangeBenefitStatusRequest struct {
	ID     uint `json:"id"`
	Status int  `json:"status"`
}

// BenefitResponse là DTO cho response của benefit
type BenefitResponse struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}
