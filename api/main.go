package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ixcsoft/idp/api/handler"
	"github.com/ixcsoft/idp/api/middleware"
	"github.com/ixcsoft/idp/shared/queue"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	token := mustEnv(log, "IDP_API_TOKEN")
	rabbitURL := mustEnv(log, "RABBITMQ_URL")
	port := envOr("PORT", "8080")

	q, err := queue.Dial(rabbitURL)
	if err != nil {
		log.Error("rabbitmq connect failed", "err", err)
		os.Exit(1)
	}
	defer q.Close()
	log.Info("rabbitmq connected")

	deploy := handler.NewDeploy(q, log)
	authed := middleware.Bearer(token)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handler.Health)
	mux.Handle("POST /deploy", authed(http.HandlerFunc(deploy.Create)))
	mux.Handle("GET /deploy/{id}", authed(http.HandlerFunc(deploy.Get)))

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info("api listening", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Info("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
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
