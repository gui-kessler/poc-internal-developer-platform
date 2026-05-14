package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/ixcsoft/idp/shared/queue"
	"github.com/ixcsoft/idp/worker/pipeline"
	"github.com/ixcsoft/idp/worker/runtime"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	rabbitURL := mustEnv(log, "RABBITMQ_URL")
	baseDomain := mustEnv(log, "IDP_BASE_DOMAIN")
	traefikNet := envOr("IDP_TRAEFIK_NETWORK", "traefik")

	qc, err := queue.Dial(rabbitURL)
	if err != nil {
		log.Error("rabbitmq connect failed", "err", err)
		os.Exit(1)
	}
	defer qc.Close()
	log.Info("rabbitmq connected")

	declCh, err := qc.Channel()
	if err != nil {
		log.Error("open channel failed", "err", err)
		os.Exit(1)
	}
	if err := queue.DeclareTopology(declCh); err != nil {
		log.Error("declare topology failed", "err", err)
		os.Exit(1)
	}
	_ = declCh.Close()

	publishCh, err := qc.Channel()
	if err != nil {
		log.Error("open publish channel failed", "err", err)
		os.Exit(1)
	}
	defer publishCh.Close()

	consumeCh, err := qc.Channel()
	if err != nil {
		log.Error("open consume channel failed", "err", err)
		os.Exit(1)
	}
	defer consumeCh.Close()

	if err := consumeCh.Qos(1, 0, false); err != nil {
		log.Error("set qos failed", "err", err)
		os.Exit(1)
	}

	rt, err := runtime.NewDocker(baseDomain, traefikNet, log)
	if err != nil {
		log.Error("docker client failed", "err", err)
		os.Exit(1)
	}
	defer rt.Close()
	log.Info("docker runtime ready", "traefik_network", traefikNet)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runner := pipeline.New(publishCh, rt, log)

	msgs, err := consumeCh.ConsumeWithContext(ctx, queue.QueueWork, "", false, false, false, false, nil)
	if err != nil {
		log.Error("consume failed", "err", err)
		os.Exit(1)
	}

	go consume(ctx, msgs, runner, log)
	log.Info("worker consuming", "queue", queue.QueueWork)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Info("worker shutting down")
	cancel()
}

func consume(ctx context.Context, msgs <-chan amqp.Delivery, runner *pipeline.Runner, log *slog.Logger) {
	for d := range msgs {
		var work queue.WorkMessage
		if err := json.Unmarshal(d.Body, &work); err != nil {
			log.Warn("invalid work message", "err", err)
			_ = d.Nack(false, false)
			continue
		}
		runner.Run(ctx, work)
		_ = d.Ack(false)
	}
	log.Info("work consumer stopped")
}

func mustEnv(log *slog.Logger, k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Error("missing env var", "var", k)
		os.Exit(1)
	}
	return v
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
