// File: OrpheusLabs/OrpheusLABS/internal/services/rabbitmq_service.go
package services

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid" // For correlation IDs
	"github.com/streadway/amqp"
)

// RabbitMQClient handles communication with RabbitMQ for RPC calls.
type RabbitMQClient struct {
	conn         *amqp.Connection
	channel      *amqp.Channel
	replyTo      string                   // Unique queue name for receiving replies
	deliveries   <-chan amqp.Delivery     // Channel to consume replies
	mu           sync.Mutex               // Protects concurrent access to pending calls map
	pendingCalls map[string]chan<- []byte // Maps correlationID to response channel
}

// NewRabbitMQClient creates and initializes a new RabbitMQ client.
func NewRabbitMQClient(amqpURL string) (*RabbitMQClient, error) {
	if amqpURL == "" {
		return nil, errors.New("RabbitMQ URL cannot be empty")
	}
	log.Printf("Attempting to connect to RabbitMQ at %s", amqpURL)

	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		// TODO: Implement reconnection logic here if desired
		log.Printf("Failed initial connection to RabbitMQ: %v", err)
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	log.Println("RabbitMQ connection established.")

	ch, err := conn.Channel()
	if err != nil {
		conn.Close() // Close connection if channel fails
		log.Printf("Failed to open RabbitMQ channel: %v", err)
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}
	log.Println("RabbitMQ channel opened.")

	// Declare a unique, exclusive, auto-delete queue for replies
	replyQueue, err := ch.QueueDeclare(
		"",    // name: let RabbitMQ generate a unique name
		false, // durable
		true,  // delete when unused
		true,  // exclusive (only this connection can use it)
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		log.Printf("Failed to declare reply queue: %v", err)
		return nil, fmt.Errorf("failed to declare reply queue: %w", err)
	}
	log.Printf("Declared reply queue: %s", replyQueue.Name)

	// Start consuming messages from the reply queue
	msgs, err := ch.Consume(
		replyQueue.Name, // queue
		"",              // consumer tag (let RabbitMQ generate one)
		true,            // auto-ack (simpler for RPC, but consider manual ack for reliability)
		true,            // exclusive
		false,           // no-local (not supported by RabbitMQ)
		false,           // no-wait
		nil,             // args
	)
	if err != nil {
		ch.Close()
		conn.Close()
		log.Printf("Failed to start consuming replies: %v", err)
		return nil, fmt.Errorf("failed to register consumer: %w", err)
	}
	log.Println("Consumer started on reply queue.")

	client := &RabbitMQClient{
		conn:         conn,
		channel:      ch,
		replyTo:      replyQueue.Name,
		deliveries:   msgs,
		pendingCalls: make(map[string]chan<- []byte),
	}

	// Start a goroutine to handle incoming replies
	go client.handleReplies()

	// TODO: Add connection/channel close monitoring and reconnection logic if needed

	return client, nil
}

// handleReplies processes incoming messages on the reply queue.
func (c *RabbitMQClient) handleReplies() {
	log.Println("Reply handler started.")
	for d := range c.deliveries {
		log.Printf("Received reply with CorrelationID: %s", d.CorrelationId)
		c.mu.Lock()
		// Find the channel associated with the correlation ID
		responseChan, ok := c.pendingCalls[d.CorrelationId]
		if ok {
			// Send the response body to the waiting channel
			responseChan <- d.Body
			// Remove the pending call entry
			delete(c.pendingCalls, d.CorrelationId)
			log.Printf("Reply routed for CorrelationID: %s", d.CorrelationId)
		} else {
			log.Printf("Warning: Received reply for unknown CorrelationID: %s", d.CorrelationId)
		}
		c.mu.Unlock()
	}
	log.Println("Reply handler stopped (delivery channel closed).")
	// This indicates the connection/channel likely closed. Trigger cleanup/reconnection if implemented.
}

// Call performs an RPC request.
// It publishes a message to the specified queue and waits for a response on the reply queue.
func (c *RabbitMQClient) Call(requestQueue string, requestBody []byte, timeout time.Duration) ([]byte, error) {
	if c.channel == nil || c.conn == nil || c.conn.IsClosed() {
		log.Println("Error: RabbitMQ connection or channel is not available for Call.")
		// TODO: Attempt reconnection here if implemented
		return nil, errors.New("RabbitMQ connection not available")
	}

	correlationID := uuid.New().String()
	responseChan := make(chan []byte) // Unbuffered channel for the response

	// Register the pending call before publishing
	c.mu.Lock()
	c.pendingCalls[correlationID] = responseChan
	c.mu.Unlock()

	// Ensure cleanup if function exits early or times out
	defer func() {
		c.mu.Lock()
		delete(c.pendingCalls, correlationID) // Remove entry on exit
		c.mu.Unlock()
	}()

	log.Printf("Publishing request to queue '%s' with CorrelationID: %s", requestQueue, correlationID)
	err := c.channel.Publish(
		"",           // exchange: use default
		requestQueue, // routing key (queue name)
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType:   "application/json",
			CorrelationId: correlationID,
			ReplyTo:       c.replyTo, // Tell the worker where to send the reply
			Body:          requestBody,
			// DeliveryMode: amqp.Persistent, // Make message persistent if needed
		},
	)
	if err != nil {
		log.Printf("Failed to publish message (CorrelationID: %s): %v", correlationID, err)
		return nil, fmt.Errorf("failed to publish message: %w", err)
	}

	log.Printf("Waiting for reply (CorrelationID: %s) with timeout %v...", correlationID, timeout)

	// Wait for the response or timeout
	select {
	case response := <-responseChan:
		log.Printf("Received reply for CorrelationID: %s", correlationID)
		return response, nil
	case <-time.After(timeout):
		log.Printf("Timeout waiting for reply (CorrelationID: %s)", correlationID)
		return nil, errors.New("timeout waiting for response")
		// Consider context cancellation as well for more complex scenarios
		// case <-ctx.Done():
		//  log.Printf("Context cancelled waiting for reply (CorrelationID: %s)", correlationID)
		// 	return nil, ctx.Err()
	}
}

// Close shuts down the client's channel and connection.
func (c *RabbitMQClient) Close() {
	log.Println("Closing RabbitMQ client...")
	// Close channel first
	if c.channel != nil {
		err := c.channel.Close()
		if err != nil {
			log.Printf("Error closing RabbitMQ channel: %v", err)
		} else {
			log.Println("RabbitMQ channel closed.")
		}
	} else {
		log.Println("RabbitMQ channel was nil, skipping close.")
	}

	// Close connection
	if c.conn != nil && !c.conn.IsClosed() {
		err := c.conn.Close()
		if err != nil {
			log.Printf("Error closing RabbitMQ connection: %v", err)
		} else {
			log.Println("RabbitMQ connection closed.")
		}
	} else {
		log.Println("RabbitMQ connection was nil or already closed, skipping close.")
	}
}
