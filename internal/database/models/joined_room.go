package models

import "github.com/uptrace/bun"

type JoinedRoom struct {
	bun.BaseModel

	ID     uint `bun:",pk,autoincrement"`
	UserID string
	RoomID uint
}
