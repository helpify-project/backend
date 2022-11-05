package router

import (
	"github.com/gorilla/mux"
)

type Controller interface {
	Register(router *mux.Router)
}
