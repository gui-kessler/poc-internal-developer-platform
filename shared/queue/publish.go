package queue

import (
	"context"
	"encoding/json"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/ixcsoft/idp/shared/spec"
)

func PublishWork(ctx context.Context, ch *amqp.Channel, deployID string, payload spec.Intent) error {
	body, err := json.Marshal(WorkMessage{DeployID: deployID, Payload: payload})
	if err != nil {
		return err
	}
	return ch.PublishWithContext(ctx, Exchange, KeyWorkDeploy, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		MessageId:    deployID,
		Timestamp:    time.Now(),
		Body:         body,
	})
}

func PublishStatus(ctx context.Context, ch *amqp.Channel, ev StatusEvent) error {
	if ev.Timestamp.IsZero() {
		ev.Timestamp = time.Now()
	}
	body, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	return ch.PublishWithContext(ctx, Exchange, StatusKey(ev.Status), false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		MessageId:    ev.DeployID,
		Timestamp:    ev.Timestamp,
		Body:         body,
	})
}
