package jsonrpc

import (
	"fmt"
	"time"

	"github.com/helpify-project/backend/internal/database/models"
)

// Sent by client
type InputMessage struct {
	RoomID  string `json:"roomId"`
	Message string `json:"message"`
}

type Message struct {
	ID        string    `json:"id"`
	Message   string    `json:"message"`
	RoomID    string    `json:"roomId"`
	UserType  uint      `json:"userType"`
	Timestamp time.Time `json:"timestamp"`
}

func MessageFromModel(msg models.Message) (m Message) {
	m.FromModel(msg)
	return
}

func (m *Message) FromModel(msg models.Message) {
	m.ID = fmt.Sprint(msg.ID)
	m.Message = msg.Message
	m.RoomID = fmt.Sprint(msg.RoomID)
	m.UserType = msg.UserType
	m.Timestamp = msg.Timestamp
}
