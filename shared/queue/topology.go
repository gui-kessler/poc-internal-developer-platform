package queue

import amqp "github.com/rabbitmq/amqp091-go"

const (
	Exchange       = "idp.deploys"
	QueueWork      = "q.work"
	QueueAPIStatus = "q.api.status"

	KeyWorkDeploy = "work.deploy.requested"

	keyWorkPattern   = "work.*"
	keyStatusPattern = "status.*"
)

func StatusKey(status string) string {
	return "status." + status
}

func DeclareTopology(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(Exchange, "topic", true, false, false, false, nil); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare(QueueWork, true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.QueueBind(QueueWork, keyWorkPattern, Exchange, false, nil); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare(QueueAPIStatus, true, false, false, false, nil); err != nil {
		return err
	}
	return ch.QueueBind(QueueAPIStatus, keyStatusPattern, Exchange, false, nil)
}
