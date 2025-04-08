package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"new/config"
	"new/models"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// Cấu trúc phản hồi từ GPT
type GPTResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type GPTHotelSearchParams struct {
	Province string   `json:"province"`
	District string   `json:"district"`
	MaxPrice int      `json:"maxPrice"`
	Benefits []string `json:"benefits"`
	Name     string   `json:"name"`
	NumTolet int      `json:"numTolet"`
	NumBed   int      `json:"numBed"`
	FromDate string   `json:"fromDate"`
	ToDate   string   `json:"toDate"`
	Status   int      `json:"status"`
}

// Hàm gọi API GPT
func GetGPTResponse(userMessage string) (string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("API key không tồn tại")
	}

	url := "https://api.openai.com/v1/chat/completions"
	prompt := fmt.Sprintf(`Người dùng đang tìm khách sạn. Trích xuất thông tin từ câu hỏi này:
    - Địa điểm
    - Giá tối đa
    - Tiện ích mong muốn
    Câu hỏi: "%s"`, userMessage)

	requestBody, _ := json.Marshal(map[string]interface{}{
		"model": "gpt-4",
		"messages": []map[string]string{
			{"role": "system", "content": "Bạn là một trợ lý chuyên trích xuất thông tin tìm kiếm khách sạn."},
			{"role": "user", "content": prompt},
		},
		"max_tokens": 100,
	})

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Lỗi gửi request GPT: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GPT API lỗi: %s", resp.Status)
	}

	body, _ := io.ReadAll(resp.Body)
	fmt.Println("GPT Raw Response:", string(body))

	var gptResponse GPTResponse
	err = json.Unmarshal(body, &gptResponse)
	if err != nil {
		fmt.Println("Lỗi JSON:", err)
		return "", fmt.Errorf("Lỗi JSON: %v", err)
	}

	if len(gptResponse.Choices) > 0 {
		return gptResponse.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("GPT không phản hồi")
}

func GetHotelSearchParamsFromUserMessage(userMessage string) (map[string]string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("API key không tồn tại")
	}

	url := "https://api.openai.com/v1/chat/completions"
	prompt := fmt.Sprintf(`Người dùng đang tìm khách sạn. Trích xuất thông tin dưới dạng JSON như sau:
{
  "province": "tên tỉnh/thành",
  "district": "tên quận/huyện nếu có",
  "maxPrice": số_nguyên,
  "benefits": ["tên tiện ích 1", "tên tiện ích 2"],
  "name": "tên khách sạn (nếu có)",
  "numTolet": số phòng tắm (nếu có)",
  "numBed": số giường (nếu có)",
  "fromDate": "yyyy-MM-dd" (ngày bắt đầu, nếu có)",
  "toDate": "yyyy-MM-dd" (ngày kết thúc, nếu có)",
  "status": số trạng thái (1 = active, 0 = inactive, nếu có)
}
Câu hỏi: "%s"`, userMessage)

	requestBody, _ := json.Marshal(map[string]interface{}{
		"model": "gpt-4",
		"messages": []map[string]string{
			{"role": "system", "content": "Bạn là trợ lý khách sạn của web Trothalo, chỉ trả về JSON đúng định dạng yêu cầu."},
			{"role": "user", "content": prompt},
		},
		"max_tokens": 200,
	})

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("lỗi gửi request GPT: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GPT API lỗi: %s", resp.Status)
	}

	body, _ := io.ReadAll(resp.Body)
	fmt.Println("GPT Raw Response:", string(body))

	var gptResponse GPTResponse
	err = json.Unmarshal(body, &gptResponse)
	if err != nil || len(gptResponse.Choices) == 0 {
		return nil, fmt.Errorf("lỗi phân tích JSON: %v", err)
	}

	// Parse nội dung JSON trong GPT content
	content := gptResponse.Choices[0].Message.Content
	var extracted GPTHotelSearchParams
	if err := json.Unmarshal([]byte(content), &extracted); err != nil {
		return nil, fmt.Errorf("không parse được nội dung GPT: %v", err)
	}

	// Chuyển thành map[string]string
	params := map[string]string{}
	if extracted.Province != "" {
		params["province"] = extracted.Province
	}
	if extracted.District != "" {
		params["district"] = extracted.District
	}
	if extracted.MaxPrice > 0 {
		params["price"] = strconv.Itoa(extracted.MaxPrice)
	}
	if len(extracted.Benefits) > 0 {
		params["benefitId"] = MapBenefitNamesToIDs(extracted.Benefits)
	}

	// Bổ sung limit/page mặc định nếu cần
	params["page"] = "1"
	params["limit"] = "10"

	return params, nil
}
func MapBenefitNamesToIDs(names []string) string {
	var benefitIDs []string

	for _, name := range names {
		var benefit models.Benefit
		err := config.DB.Where("name ILIKE ?", name).First(&benefit).Error
		if err == nil {
			benefitIDs = append(benefitIDs, fmt.Sprintf("%d", benefit.Id))
		}
	}

	return strings.Join(benefitIDs, ",")
}

func ChatSearchHandler(c *gin.Context) {
	var request struct {
		Message string `json:"message"`
	}
	if err := c.ShouldBindJSON(&request); err != nil || request.Message == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Message không hợp lệ"})
		return
	}

	// Gọi GPT để trích xuất thông tin
	params, err := GetHotelSearchParamsFromUserMessage(request.Message)
	if err != nil {
		log.Println("Lỗi GPT:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không trích xuất được thông tin từ GPT"})
		return
	}

	// Tìm kiếm trong ElasticSearch
	accommodations,_, err := SearchAccommodationsWithFilters(params)
	if err != nil {
		log.Println("Lỗi tìm kiếm Elastic:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không tìm được kết quả phù hợp"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"results": accommodations,
	})
}
