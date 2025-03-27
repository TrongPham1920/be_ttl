package notification

import (
	"fmt"

	"github.com/olahol/melody"
)

// Service interface định nghĩa các phương thức gửi thông báo
type Service interface {
	SendMessage(message string) error
}

// MelodyService implement Service interface cho Melody
type MelodyService struct {
	m *melody.Melody
}

// NewMelodyService tạo một instance mới của MelodyService
func NewMelodyService(m *melody.Melody) *MelodyService {
	return &MelodyService{m: m}
}

// SendMessage gửi thông báo qua Melody
func (s *MelodyService) SendMessage(message string) error {
	if s.m == nil {
		return fmt.Errorf("melody instance is nil")
	}
	return s.m.Broadcast([]byte(message))
}

// MessageBuilder giúp tạo message thông báo
type MessageBuilder struct {
	userID  uint
	revenue float64
}

// NewMessageBuilder tạo một instance mới của MessageBuilder
func NewMessageBuilder(userID uint, revenue float64) *MessageBuilder {
	return &MessageBuilder{
		userID:  userID,
		revenue: revenue,
	}
}

// Build tạo message thông báo
func (b *MessageBuilder) Build() string {
	return fmt.Sprintf("🔔 User %d đã được cộng %.2f vào tài khoản.", b.userID, b.revenue)
}
