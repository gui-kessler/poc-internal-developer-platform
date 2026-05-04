package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/ixcsoft/idp/shared/queue"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	rabbitURL := mustEnv(log, "RABBITMQ_URL")

	q, err := queue.Dial(rabbitURL)
	if err != nil {
		log.Error("rabbitmq connect failed", "err", err)
		os.Exit(1)
	}
	defer q.Close()

	log.Info("worker started; pipeline consumers wired in fase 1")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Info("worker shutting down")
}

func mustEnv(log *slog.Logger, k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Error("missing env var", "var", k)
		os.Exit(1)
	}
	return v
}
