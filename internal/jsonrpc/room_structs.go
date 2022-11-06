package jsonrpc

import "time"

type Room struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
}
