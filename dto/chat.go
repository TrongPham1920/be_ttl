package dto

type HotelSummary struct {
	ID     uint   `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
	Num    int    `json:"num"`
	Price  int    `json:"price"`
}

type IncomingMessage struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"`
	Text      string `json:"text"`
	Sender    string `json:"sender"`
	Timestamp string `json:"timestamp"`
}
