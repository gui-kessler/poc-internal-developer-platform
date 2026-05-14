package pipeline

import (
	"context"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/ixcsoft/idp/shared/queue"
	"github.com/ixcsoft/idp/worker/runtime"
)

type Runner struct {
	ch  *amqp.Channel
	rt  *runtime.Docker
	log *slog.Logger
}

func New(ch *amqp.Channel, rt *runtime.Docker, log *slog.Logger) *Runner {
	return &Runner{ch: ch, rt: rt, log: log}
}

func (r *Runner) Run(ctx context.Context, work queue.WorkMessage) {
	log := r.log.With("deploy_id", work.DeployID, "app", work.Payload.App)
	log.Info("pipeline start")

	r.emit(ctx, log, work.DeployID, queue.StatusQueued, "", "")

	if err := r.security(ctx, work); err != nil {
		r.emit(ctx, log, work.DeployID, queue.StatusFailed, "security: "+err.Error(), "")
		return
	}
	r.emit(ctx, log, work.DeployID, queue.StatusSecurityPassed, "", "")

	if err := r.policy(ctx, work); err != nil {
		r.emit(ctx, log, work.DeployID, queue.StatusFailed, "policy: "+err.Error(), "")
		return
	}
	r.emit(ctx, log, work.DeployID, queue.StatusPolicyPassed, "", "")

	url, err := r.rt.DeployApp(ctx, work.DeployID, work.Payload)
	if err != nil {
		log.Error("deploy failed", "err", err)
		r.emit(ctx, log, work.DeployID, queue.StatusFailed, "deploy: "+err.Error(), "")
		return
	}
	r.emit(ctx, log, work.DeployID, queue.StatusRunning, "", url)
	log.Info("pipeline done", "url", url)
}

func (r *Runner) emit(ctx context.Context, log *slog.Logger, deployID, status, msg, url string) {
	ev := queue.StatusEvent{DeployID: deployID, Status: status, Message: msg, URL: url}
	if err := queue.PublishStatus(ctx, r.ch, ev); err != nil {
		log.Error("publish status failed", "status", status, "err", err)
	}
}

// Stubs — Trivy real entra na Fase 3, policy engine real na Fase 2.

func (r *Runner) security(_ context.Context, _ queue.WorkMessage) error {
	time.Sleep(200 * time.Millisecond)
	return nil
}

func (r *Runner) policy(_ context.Context, _ queue.WorkMessage) error {
	time.Sleep(200 * time.Millisecond)
	return nil
}
