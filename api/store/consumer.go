package store

import (
	"context"
	"encoding/json"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/ixcsoft/idp/shared/queue"
)

// Consume registra um consumer em q.api.status e atualiza o store em background.
// O canal deve ser dedicado a este consumer.
func Consume(ctx context.Context, ch *amqp.Channel, st *Store, log *slog.Logger) error {
	msgs, err := ch.ConsumeWithContext(ctx, queue.QueueAPIStatus, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	go func() {
		for d := range msgs {
			var ev queue.StatusEvent
			if err := json.Unmarshal(d.Body, &ev); err != nil {
				log.Warn("invalid status event", "err", err)
				_ = d.Nack(false, false)
				continue
			}
			st.Set(ev)
			log.Info("status updated", "deploy_id", ev.DeployID, "status", ev.Status)
			_ = d.Ack(false)
		}
		log.Info("status consumer stopped")
	}()
	return nil
}
