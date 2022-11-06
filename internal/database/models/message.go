package models

import (
	"time"

	"github.com/uptrace/bun"
)

type Message struct {
	bun.BaseModel

	ID        uint `bun:",pk,autoincrement"`
	RoomID    uint
	Sender    string
	Timestamp time.Time
	UserType  uint
	Message   string
}
