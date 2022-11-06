package jsonrpc

import (
	"context"
	"strconv"

	"github.com/uptrace/bun"

	"github.com/helpify-project/backend/internal/database/models"
)

type baseService struct {
	DB *bun.DB
}

func (s *baseService) findRoom(ctx context.Context, roomID string) (room models.Room, err error) {
	var intRoomID int
	if intRoomID, err = strconv.Atoi(roomID); err != nil {
		return
	}

	err = s.DB.NewSelect().
		Model(&room).
		Where("id = ?", intRoomID).
		Scan(ctx)
	return
}

func (s *baseService) inRoom(ctx context.Context, sid string, roomID string) (ok bool, err error) {
	var intRoomID int
	if intRoomID, err = strconv.Atoi(roomID); err != nil {
		return
	}

	ok, err = s.DB.NewSelect().
		Model((*models.JoinedRoom)(nil)).
		Where("room_id = ?", intRoomID).
		Where("user_id = ?", sid).
		Exists(ctx)

	return
}
