package handler

import (
	"log/slog"
	"net/http"

	"github.com/ixcsoft/idp/shared/queue"
)

type Deploy struct {
	q   *queue.Client
	log *slog.Logger
}

func NewDeploy(q *queue.Client, log *slog.Logger) *Deploy {
	return &Deploy{q: q, log: log}
}

func (d *Deploy) Create(w http.ResponseWriter, r *http.Request) {
	d.log.Info("deploy create called", "remote", r.RemoteAddr)
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (d *Deploy) Get(w http.ResponseWriter, r *http.Request) {
	d.log.Info("deploy get called", "id", r.PathValue("id"))
	http.Error(w, "not implemented", http.StatusNotImplemented)
}
