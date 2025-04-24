package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"net/http"
	"new/config"
	"new/dto"
	"new/models"
	"os"
)

type GPTResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type GPTHotelSearchParams struct {
	Type     *int     `json:"type,omitempty"`
	Province string   `json:"province,omitempty"`
	District string   `json:"district,omitempty"`
	MaxPrice *int     `json:"maxPrice,omitempty"`
	Benefits []string `json:"benefits,omitempty"`
	Name     string   `json:"name,omitempty"`
	NumTolet *int     `json:"numTolet,omitempty"`
	NumBed   *int     `json:"numBed,omitempty"`
	FromDate string   `json:"fromDate,omitempty"`
	ToDate   string   `json:"toDate,omitempty"`
	Status   *int     `json:"status,omitempty"`
}

// =========================
// GPT REQUEST
// =========================
func CheckForContactIntent(message string) (bool, string) {
	lowerMsg := strings.ToLower(message)

	keywords := []string{
		"Liên hệ", "liên hệ", "lien he", "admin", "hỗ trợ", "gặp tư vấn", "hotline",
	}

	for _, keyword := range keywords {
		if strings.Contains(lowerMsg, keyword) {
			// Có thể customize message này
			contactMsg := "Bạn có thể liên hệ với chúng tôi qua:\n📞 Hotline: 0123 456 789\n✉️ Email: TROTHALO@email.com"
			return true, contactMsg
		}
	}
	return false, ""
}

func ExtractSearchFiltersFromGPTWS(userMessage string) (*dto.SearchFilters, string, error) {
	if ok, msg := CheckForContactIntent(userMessage); ok {
		return nil, msg, nil
	}
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, "", fmt.Errorf("API key không tồn tại")
	}

	url := "https://api.openai.com/v1/chat/completions"

	systemPrompt := `
Bạn là trợ lý ảo chuyên tư vấn hệ thống đặt phòng khách sạn.
- Nếu người dùng đặt câu hỏi thông thường (như hỏi về cách sử dụng, tài khoản, thanh toán, chính sách hủy, v.v...), hãy trả lời một cách thân thiện, KHÔNG trả JSON.
- Chỉ khi người dùng có nhu cầu tìm kiếm khách sạn (có ý định rõ ràng), hãy trích xuất thông tin tìm kiếm và trả về JSON đúng format sau:

{
  "type": int,               // 0: hotel, 1: homestay, 2: villa
  "province": "string",
  "district": "string",
  "maxPrice": int,
  "benefits": ["string"],
  "name": "string",
  "numTolet": int,
  "numBed": int,
  "nums": int, // số sao đánh giá
  "fromDate": "yyyy-MM-dd",
  "toDate": "yyyy-MM-dd",
  "status": int
}

Ghi chú:
- Nếu có từ như "gần biển", gợi ý các tỉnh ven biển như Vũng Tàu, Nha Trang, Đà Nẵng.
- Giá tiền như "400k", "2 triệu" hãy convert về số nguyên (vd: 400000, 2000000).
- Nếu không có thông tin thì KHÔNG đưa vào JSON.
- Tuyệt đối không kèm bất kỳ lời thoại nào ngoài JSON nếu có yêu cầu tìm kiếm.
`

	requestBody, _ := json.Marshal(map[string]interface{}{
		"model": "gpt-4",
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userMessage},
		},
		"max_tokens":  500,
		"temperature": 0.2,
	})

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Println("GPT raw response:", string(body))

	var gptResp GPTResponse
	if err := json.Unmarshal(body, &gptResp); err != nil || len(gptResp.Choices) == 0 {
		return nil, "", fmt.Errorf("GPT trả về lỗi hoặc không hợp lệ")
	}

	responseContent := strings.TrimSpace(gptResp.Choices[0].Message.Content)

	// Nếu response bắt đầu bằng dấu { thì giả định là JSON
	if strings.HasPrefix(responseContent, "{") {
		var gptData GPTHotelSearchParams
		if err := json.Unmarshal([]byte(responseContent), &gptData); err != nil {
			log.Printf("GPT trả về JSON nhưng lỗi khi parse: %s\n", responseContent)
			return nil, "", fmt.Errorf("lỗi parse JSON GPT: %v", err)
		}

		// Parse ngày
		layout := "2006-01-02"
		var fromDate, toDate *time.Time
		if gptData.FromDate != "" {
			t, err := time.Parse(layout, gptData.FromDate)
			if err == nil {
				fromDate = &t
			}
		}
		if gptData.ToDate != "" {
			t, err := time.Parse(layout, gptData.ToDate)
			if err == nil {
				toDate = &t
			}
		}

		// Convert sang SearchFilters
		filters := &dto.SearchFilters{
			Type:     gptData.Type,
			Province: gptData.Province,
			District: gptData.District,
			Name:     gptData.Name,
			PriceMax: gptData.MaxPrice,
			NumTolet: gptData.NumTolet,
			NumBed:   gptData.NumBed,
			FromDate: fromDate,
			ToDate:   toDate,
			Status:   gptData.Status,
			Page:     1,
			Limit:    10,
		}

		if len(gptData.Benefits) > 0 {
			filters.BenefitIDs = mapBenefitNamesToIDs(gptData.Benefits)
		}

		return filters, "Đây là danh sách các khách sạn phù hợp với yêu cầu của bạn:", nil
	}

	// Nếu không phải JSON, coi như GPT đang tư vấn tự nhiên
	return nil, responseContent, nil
}

// Hàm xử lý cho Tìm kiếm nâng cao
func ExtractSearchFiltersFromGPT(userMessage string) (*dto.SearchFilters, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("API key không tồn tại")
	}

	url := "https://api.openai.com/v1/chat/completions"
	prompt := fmt.Sprintf(`Trích xuất thông tin dưới dạng JSON như sau:
{
  "type": int,
  "province": "string",
  "district": "string",
  "maxPrice": int,
  "benefits": ["string"],
  "name": "string",
  "numTolet": int,
  "numBed": int,
  "fromDate": "yyyy-MM-dd",
  "toDate": "yyyy-MM-dd",
  "status": int
}
  Ghi chú:
- Nếu người dùng có nhập từ khóa "gần biển" hoặc tương tự thì cung cấp các province có giáp biển ở Việt Nam như Vũng Tàu, Nha Trang, Đà Nẵng.
- Nếu người dùng nhập các từ đồng nghĩa với "hotel" hay "khách sạn" thì trả "type": 0, còn đồng nghĩa với "homestay" thì trả "type" : 1, còn đồng nghĩa với "villa" thì trả "type":2
- Nếu người dùng nhập giá tiền như "400k", "bốn trăm", "4 trăm", "2 triệu", thì hãy tự động chuyển về số nguyên đơn vị đồng (vd: 400000, 2000000).
- Trường nào không có thì bỏ qua.
Người dùng: "%s"`, userMessage)

	requestBody, _ := json.Marshal(map[string]interface{}{
		"model": "gpt-4",
		"messages": []map[string]string{
			{"role": "system", "content": "Bạn là trợ lý chuyên gợi ý khách sạn. Khi nào người dùng muốn tìm kiếm sẽ trả về JSON."},
			{"role": "user", "content": prompt},
		},
		"max_tokens": 300,
	})

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Println("GPT raw response:", string(body))
	var gptResp GPTResponse
	if err := json.Unmarshal(body, &gptResp); err != nil || len(gptResp.Choices) == 0 {
		return nil, fmt.Errorf("GPT trả về lỗi")
	}

	var gptData GPTHotelSearchParams
	if err := json.Unmarshal([]byte(gptResp.Choices[0].Message.Content), &gptData); err != nil {
		log.Printf("GPT trả JSON nhưng lỗi khi parse: %s\n", gptResp.Choices[0].Message.Content)
		return nil, fmt.Errorf("không parse JSON GPT: %v", err)
	}

	layout := "02/01/2006"
	var fromDate, toDate *time.Time

	if gptData.FromDate != "" {
		t, err := time.Parse(layout, gptData.FromDate)
		if err == nil {
			fromDate = &t
		}
	}
	if gptData.ToDate != "" {
		t, err := time.Parse(layout, gptData.ToDate)
		if err == nil {
			toDate = &t
		}
	}
	// Convert sang SearchFilters
	filters := &dto.SearchFilters{
		Type:     gptData.Type,
		Province: gptData.Province,
		District: gptData.District,
		Name:     gptData.Name,
		PriceMax: gptData.MaxPrice,
		NumTolet: gptData.NumTolet,
		NumBed:   gptData.NumBed,
		FromDate: fromDate,
		ToDate:   toDate,
		Status:   gptData.Status,
		Page:     1,
		Limit:    10,
	}

	// Mapping benefit names to IDs
	if len(gptData.Benefits) > 0 {
		ids := mapBenefitNamesToIDs(gptData.Benefits)
		filters.BenefitIDs = ids
	}

	return filters, nil
}

func mapBenefitNamesToIDs(names []string) []int {
	var benefitIDs []int
	for _, name := range names {
		var benefit models.Benefit
		err := config.DB.Where("name ILIKE ?", name).First(&benefit).Error
		if err == nil {
			benefitIDs = append(benefitIDs, benefit.Id)
		}
	}
	return benefitIDs
}
