package queue

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

type Client struct {
	conn *amqp.Connection
}

func Dial(url string) (*Client, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn}, nil
}

// Channel opens a fresh AMQP channel. Cada finalidade (publicar trabalho,
// publicar status, consumir status, consumir trabalho) deve usar um canal
// dedicado — canais AMQP têm restrições de concorrência.
func (c *Client) Channel() (*amqp.Channel, error) {
	return c.conn.Channel()
}

func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
