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
