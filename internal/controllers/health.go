package controllers

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/helpify-project/backend/internal/router"
)

var _ router.Controller = (*HealthController)(nil)

type HealthController struct {
}

func (c *HealthController) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "OK")

	return
}

func (c *HealthController) Register(router *mux.Router) {
	router.HandleFunc("/healthz", c.handleHealthz).
		Methods(http.MethodGet)
}
