package controllers

import (
	"net/http/pprof"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"github.com/helpify-project/backend/internal/router"
)

var _ router.Controller = (*GoDebugController)(nil)

type GoDebugController struct {
}

func (c *GoDebugController) Register(router *mux.Router) {
	zap.L().Warn("enabling /debug/pprof endpoint")
	router.HandleFunc("/debug/pprof/", pprof.Index)
	router.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	router.HandleFunc("/debug/pprof/profile", pprof.Profile)
	router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
}
