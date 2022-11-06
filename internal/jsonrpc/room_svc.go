package jsonrpc

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/uptrace/bun"

	"github.com/helpify-project/backend/internal/cctx"
	"github.com/helpify-project/backend/internal/database/models"
	"github.com/helpify-project/backend/internal/rpc"
)

func NewRoomService(db *bun.DB) *RoomService {
	return &RoomService{
		baseService: baseService{
			DB: db,
		},
	}
}

type RoomService struct {
	baseService
}

func (s *RoomService) Create(ctx context.Context) (roomID string, err error) {
	sid := ctx.Value(cctx.SessionID).(string)

	newRoom := models.Room{
		Owner:     sid,
		CreatedAt: time.Now(),
	}

	err = s.DB.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) (err error) {
		// Check if user has any active rooms before creating new one
		var existingRooms int
		existingRooms, err = tx.NewSelect().
			Model((*models.Room)(nil)).
			Where("owner = ?", sid).
			Where("archived_at IS NULL").
			Count(ctx)
		if err != nil {
			return
		} else if existingRooms > 0 {
			err = fmt.Errorf("too many open rooms")
			return
		}

		// Create new room
		_, err = tx.NewInsert().
			Model(&newRoom).
			Exec(ctx)
		if err != nil {
			return
		}

		// Join to the room as well
		newJoinedRoom := models.JoinedRoom{
			UserID: sid,
			RoomID: newRoom.ID,
		}

		_, err = tx.NewInsert().
			Model(&newJoinedRoom).
			Exec(ctx)

		return
	})
	if err != nil {
		return
	}

	roomID = fmt.Sprint(newRoom.ID)
	return
}

func (s *RoomService) Join(ctx context.Context, roomID string) (ok bool, err error) {
	sid := ctx.Value(cctx.SessionID).(string)
	supportPersonnel := ctx.Value(cctx.SupportPersonnel).(bool)

	if !supportPersonnel {
		err = fmt.Errorf("not allowed")
		return
	}

	// Find the room
	var room models.Room
	if room, err = s.findRoom(ctx, roomID); err != nil {
		return
	}

	newJoinedRoom := models.JoinedRoom{
		UserID: sid,
		RoomID: room.ID,
	}

	_, err = s.DB.NewInsert().
		Model(&newJoinedRoom).
		Exec(ctx)

	ok = err == nil
	return
}

func (s *RoomService) Archive(ctx context.Context, roomID string) (ok bool, err error) {
	sid := ctx.Value(cctx.SessionID).(string)
	supportPersonnel := ctx.Value(cctx.SupportPersonnel).(bool)

	// Find the room
	var room models.Room
	var inRoom bool

	err = s.DB.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) (err error) {
		if room, err = s.findRoom(ctx, roomID); err != nil {
			return
		}

		if inRoom, err = s.inRoom(ctx, sid, roomID); err != nil {
			return
		}

		if !inRoom {
			err = fmt.Errorf("not member of given room")
			return
		}

		if !supportPersonnel && room.Owner != sid {
			err = fmt.Errorf("can only interact with your own rooms")
			return
		}

		if room.ArchivedAt != nil {
			err = fmt.Errorf("already archived")
			return
		}

		_, err = tx.NewUpdate().
			Model(&room).
			Where("id = ?", room.ID).
			Set("archived_at = ?", time.Now()).
			Exec(ctx)

		return
	})

	ok = err == nil
	return
}

func (s *RoomService) List(ctx context.Context) (rooms []Room, err error) {
	rooms = make([]Room, 0)
	supportPersonnel := ctx.Value(cctx.SupportPersonnel).(bool)

	if !supportPersonnel {
		err = fmt.Errorf("not allowed")
		return
	}

	var dbRooms []models.Room
	err = s.DB.NewSelect().
		Model(&dbRooms).
		Column("id", "created_at", "archived_at").
		Where("archived_at IS NULL").
		Order("created_at DESC").
		Scan(ctx)

	if err != nil {
		return
	}

	for _, room := range dbRooms {
		rooms = append(rooms, Room{
			ID:        fmt.Sprint(room.ID),
			CreatedAt: room.CreatedAt,
		})
	}

	return
}

func (s *RoomService) NewRooms(ctx context.Context) (sub *rpc.Subscription, err error) {
	supportPersonnel := ctx.Value(cctx.SupportPersonnel).(bool)

	if !supportPersonnel {
		err = fmt.Errorf("not allowed")
		return
	}

	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		err = rpc.ErrNotificationsUnsupported
		return
	}

	sub = notifier.CreateSubscription()

	return
}
