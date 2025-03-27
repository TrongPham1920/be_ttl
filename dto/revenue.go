package dto

type RevenueResponse struct {
	TotalRevenue         float64        `json:"totalRevenue"`
	CurrentMonthRevenue  float64        `json:"currentMonthRevenue"`
	LastMonthRevenue     float64        `json:"lastMonthRevenue"`
	CurrentWeekRevenue   float64        `json:"currentWeekRevenue"`
	MonthlyRevenue       []MonthRevenue `json:"monthlyRevenue"`
	VAT                  float64        `json:"vat"`
	ActualMonthlyRevenue float64        `json:"actualMonthlyRevenue"`
}

type MonthRevenue struct {
	Month      string  `json:"month"`
	Revenue    float64 `json:"revenue"`
	OrderCount int     `json:"orderCount"`
}

type UserRevenueResponse struct {
	ID         uint        `json:"id"`
	Date       string      `json:"date"`
	OrderCount int         `json:"order_count"`
	Revenue    float64     `json:"revenue"`
	User       UserRevenue `json:"user"`
}

type UserRevenue struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phoneNumber"`
}
