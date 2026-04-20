package rabbitmq

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	reconnectDelay = 5 * time.Second
	maxRetries     = 5
)

type Client struct {
	url             string
	conn            *amqp.Connection
	ch              *amqp.Channel
	notifyConnClose chan *amqp.Error
	notifyChanClose chan *amqp.Error
	isConnected     bool
	mu              sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
}

// NewClient creates a new RabbitMQ client that automatically manages connections and channels.
func NewClient(url string) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	client := &Client{
		url:    url,
		ctx:    ctx,
		cancel: cancel,
	}

	go client.handleReconnect()

	return client
}

// handleReconnect handles initial connection and subsequent reconnections with exponential backoff.
func (c *Client) handleReconnect() {
	for {
		c.setConnected(false)
		log.Printf("Attempting to connect to RabbitMQ at %s...", c.url)

		var backoff = 1 * time.Second
		for {
			conn, err := amqp.Dial(c.url)
			if err == nil {
				ch, err := conn.Channel()
				if err == nil {
					c.mu.Lock()
					c.conn = conn
					c.ch = ch
					c.notifyConnClose = make(chan *amqp.Error, 1)
					c.notifyChanClose = make(chan *amqp.Error, 1)
					c.conn.NotifyClose(c.notifyConnClose)
					c.ch.NotifyClose(c.notifyChanClose)
					c.mu.Unlock()
					
					c.setConnected(true)
					log.Println("Successfully connected to RabbitMQ!")
					break
				}
				conn.Close()
			}

			log.Printf("Failed to connect: %v. Retrying in %v...", err, backoff)
			
			select {
			case <-c.ctx.Done():
				return
			case <-time.After(backoff):
				backoff *= 2
				if backoff > 30*time.Second {
					backoff = 30 * time.Second
				}
			}
		}

		if done := c.waitForClose(); done {
			return
		}
	}
}

// waitForClose waits for connection or channel to close or for client shutdown.
func (c *Client) waitForClose() bool {
	select {
	case <-c.ctx.Done():
		return true
	case err := <-c.notifyConnClose:
		log.Printf("RabbitMQ connection closed: %v", err)
	case err := <-c.notifyChanClose:
		log.Printf("RabbitMQ channel closed: %v", err)
	}
	return false
}

func (c *Client) setConnected(status bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.isConnected = status
}

// IsConnected returns the current connection status safely.
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isConnected
}

// ExchangeDeclare declares an exchange on the server.
func (c *Client) ExchangeDeclare(name, kind string, durable, autoDelete bool) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.isConnected {
		return fmt.Errorf("client is not connected to RabbitMQ")
	}

	return c.ch.ExchangeDeclare(
		name,
		kind,
		durable,
		autoDelete,
		false, // internal
		false, // no-wait
		nil,   // args
	)
}

// QueueDeclare declares a queue on the server.
func (c *Client) QueueDeclare(name string, durable, autoDelete bool) (amqp.Queue, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.isConnected {
		return amqp.Queue{}, fmt.Errorf("client is not connected to RabbitMQ")
	}

	return c.ch.QueueDeclare(
		name,
		durable,
		autoDelete,
		false, // exclusive
		false, // no-wait
		nil,   // args
	)
}

// QueueBind binds a queue to an exchange.
func (c *Client) QueueBind(name, key, exchange string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.isConnected {
		return fmt.Errorf("client is not connected to RabbitMQ")
	}

	return c.ch.QueueBind(
		name,
		key,
		exchange,
		false, // no-wait
		nil,   // args
	)
}

// Publish sends a message to the specified exchange and routing key.
func (c *Client) Publish(ctx context.Context,exchange, routingKey string, msg amqp.Publishing) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.isConnected {
		return fmt.Errorf("client is not connected to RabbitMQ")
	}

	return c.ch.PublishWithContext(
		ctx,
		exchange,
		routingKey,
		false, // mandatory
		false, // immediate
		msg,
	)
}

// Consume starts consuming messages from the specified queue.
func (c *Client) Consume(queue, consumer string, autoAck bool) (<-chan amqp.Delivery, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.isConnected {
		return nil, fmt.Errorf("client is not connected to RabbitMQ")
	}

	return c.ch.Consume(
		queue,
		consumer,
		autoAck,
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
}

// Close gracefully closes the RabbitMQ client.
func (c *Client) Close() error {
	c.cancel()
	c.setConnected(false)

	c.mu.Lock()
	defer c.mu.Unlock()

	var err error
	if c.ch != nil {
		if chErr := c.ch.Close(); chErr != nil {
			err = chErr
		}
	}
	if c.conn != nil {
		if connErr := c.conn.Close(); connErr != nil {
			err = connErr
		}
	}
	return err
}
