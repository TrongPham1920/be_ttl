package notification

import (
	"fmt"

	"github.com/olahol/melody"
)

type Service interface {
	SendMessage(message string) error
}

type MelodyService struct {
	m *melody.Melody
}

func NewMelodyService(m *melody.Melody) *MelodyService {
	return &MelodyService{m: m}
}

func (s *MelodyService) SendMessage(message string) error {
	if s.m == nil {
		return fmt.Errorf("melody instance is nil")
	}
	return s.m.Broadcast([]byte(message))
}

// func (s *MelodyService) SendMessage(message string, userID *uint) error {
// 	if s.m == nil {
// 		return fmt.Errorf("melody instance is nil")
// 	}
// 	if userID == nil {

// 		return s.m.Broadcast([]byte(message))
// 	}

// 	return s.m.BroadcastFilter([]byte(message), func(session *melody.Session) bool {
// 		if userIDStr, exists := session.Get("userID"); exists {
// 			return userIDStr == fmt.Sprintf("%d", *userID)
// 		}
// 		return false
// 	})
// }

type MessageBuilder struct {
	userID  uint
	revenue float64
}

func NewMessageBuilder(userID uint, revenue float64) *MessageBuilder {
	return &MessageBuilder{
		userID:  userID,
		revenue: revenue,
	}
}

func (b *MessageBuilder) Build() string {
	return fmt.Sprintf("🔔 User %d đã được cộng %.2f vào tài khoản.", b.userID, b.revenue)
}
