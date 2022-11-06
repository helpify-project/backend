package cctx

type ContextKey string

var (
	SessionID        ContextKey = "ha:sid"
	SupportPersonnel ContextKey = "ha:sp"
)
