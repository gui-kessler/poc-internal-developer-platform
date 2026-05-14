package handler

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/ixcsoft/idp/api/store"
	"github.com/ixcsoft/idp/shared/queue"
	"github.com/ixcsoft/idp/shared/spec"
)

var appNameRe = regexp.MustCompile(`^[a-z][a-z0-9-]{1,30}$`)

var allowedDeps = map[string]struct{}{
	"redis":    {},
	"postgres": {},
	"mongo":    {},
}

type Deploy struct {
	ch         *amqp.Channel
	store      *store.Store
	baseDomain string
	log        *slog.Logger
}

func NewDeploy(ch *amqp.Channel, st *store.Store, baseDomain string, log *slog.Logger) *Deploy {
	return &Deploy{ch: ch, store: st, baseDomain: baseDomain, log: log}
}

func (d *Deploy) Create(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	var p spec.Intent
	if err := dec.Decode(&p); err != nil {
		http.Error(w, "invalid payload: "+err.Error(), http.StatusBadRequest)
		return
	}
	if !appNameRe.MatchString(p.App) {
		http.Error(w, "invalid app name (must match ^[a-z][a-z0-9-]{1,30}$)", http.StatusBadRequest)
		return
	}
	if p.Image == "" {
		http.Error(w, "image is required", http.StatusBadRequest)
		return
	}
	if p.Port < 1024 || p.Port > 65535 {
		http.Error(w, "port must be 1024..65535", http.StatusBadRequest)
		return
	}
	if p.Replicas < 1 {
		p.Replicas = 1
	}
	for _, dep := range p.Dependencies {
		if _, ok := allowedDeps[dep]; !ok {
			http.Error(w, "dependency not allowed: "+dep, http.StatusBadRequest)
			return
		}
	}

	deployID := newDeployID()

	if err := queue.PublishWork(r.Context(), d.ch, deployID, p); err != nil {
		d.log.Error("publish work failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	d.store.Set(queue.StatusEvent{
		DeployID:  deployID,
		Status:    queue.StatusQueued,
		Timestamp: time.Now(),
	})

	resp := map[string]string{
		"deploy_id": deployID,
		"status":    queue.StatusQueued,
		"url":       "https://" + p.App + "." + d.baseDomain,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(resp)
}

func (d *Deploy) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ev, ok := d.store.Get(id)
	if !ok {
		http.Error(w, "deploy not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ev)
}

func newDeployID() string {
	var buf [8]byte
	_, _ = rand.Read(buf[:])
	return fmt.Sprintf("dep_%d_%x", time.Now().Unix(), buf)
}
