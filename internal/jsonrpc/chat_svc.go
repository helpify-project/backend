package jsonrpc

import (
	"context"
	"time"

	"github.com/uptrace/bun"

	"github.com/helpify-project/backend/internal/cctx"
	"github.com/helpify-project/backend/internal/database/models"
)

func NewChatService(db *bun.DB) *ChatService {
	return &ChatService{
		baseService: baseService{
			DB: db,
		},
	}
}

type ChatService struct {
	baseService
}

func (s *ChatService) Send(ctx context.Context, input InputMessage) (msg Message, err error) {
	sid := ctx.Value(cctx.SessionID).(string)
	supportPersonnel := ctx.Value(cctx.SupportPersonnel).(bool)
	now := time.Now()

	// Find the room
	var room models.Room
	if room, err = s.findRoom(ctx, input.RoomID); err != nil {
		return
	}

	dbMsg := models.Message{
		RoomID:    room.ID,
		Sender:    sid,
		Timestamp: now,
		Message:   input.Message,
		UserType:  0,
	}

	if supportPersonnel {
		dbMsg.UserType = 1
	}

	_, err = s.DB.NewInsert().
		Model(&dbMsg).
		Exec(ctx)
	if err != nil {
		return
	}

	msg.FromModel(dbMsg)
	return
}

func (s *ChatService) History(ctx context.Context, roomID string) (messages []Message, err error) {
	sid := ctx.Value(cctx.SessionID).(string)
	messages = make([]Message, 0)

	// Find the room
	var room models.Room
	if room, err = s.findRoom(ctx, roomID); err != nil {
		return
	}

	_ = sid

	var dbMessages []models.Message
	err = s.DB.NewSelect().
		Model(&dbMessages).
		Where("room_id = ?", room.ID).
		Scan(ctx)
	if err != nil {
		return
	}

	for _, msg := range dbMessages {
		messages = append(messages, MessageFromModel(msg))
	}

	return
}

/*
func (s *ChatService) NewChatMessages(ctx context.Context) (sub *rpc.Subscription, err error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		err = rpc.ErrNotificationsUnsupported
		return
	}

	sub = notifier.CreateSubscription()

	return
}
*/
