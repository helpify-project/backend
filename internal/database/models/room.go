package models

import (
	"time"

	"github.com/uptrace/bun"
)

type Room struct {
	bun.BaseModel

	ID         uint `bun:",pk,autoincrement"`
	Owner      string
	CreatedAt  time.Time
	ArchivedAt *time.Time
}
