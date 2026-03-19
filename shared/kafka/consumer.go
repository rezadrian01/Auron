// Package kafka provides reusable Kafka producer and consumer helpers for all Auron services.
package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

// ConsumerConfig holds Kafka consumer configuration
type ConsumerConfig struct {
	Brokers       []string
	Topic         string
	GroupID       string
	MinBytes      int
	MaxBytes      int
	MaxWait       time.Duration
	CommitInterval time.Duration
	StartOffset   int64
}

// MessageHandler is a function type for processing Kafka messages
type MessageHandler func(ctx context.Context, msg kafka.Message) error

// Consumer wraps the Kafka reader with error handling and graceful shutdown
type Consumer struct {
	reader  *kafka.Reader
	handler MessageHandler
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(cfg *ConsumerConfig, handler MessageHandler) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Brokers,
		Topic:          cfg.Topic,
		GroupID:        cfg.GroupID,
		MinBytes:       cfg.MinBytes,
		MaxBytes:       cfg.MaxBytes,
		MaxWait:        cfg.MaxWait,
		CommitInterval: cfg.CommitInterval,
		StartOffset:    cfg.StartOffset,
		// Error handler
		// Logger: kafka.LoggerFunc(func(v ...interface{}) {}),
	})

	ctx, cancel := context.WithCancel(context.Background())

	return &Consumer{
		reader:  reader,
		handler: handler,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// NewConsumerWithDefaults creates a new Kafka consumer with default settings
func NewConsumerWithDefaults(brokers []string, topic string, groupID string, handler MessageHandler) *Consumer {
	return NewConsumer(&ConsumerConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       1,
		MaxBytes:       10e6, // 10MB
		MaxWait:        time.Second,
		CommitInterval: time.Second,
		StartOffset:    kafka.LastOffset,
	}, handler)
}

// Start begins consuming messages
func (c *Consumer) Start() {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for {
			// Check if context is cancelled
			select {
			case <-c.ctx.Done():
				return
			default:
			}

			// Read message with context
			msg, err := c.reader.ReadMessage(c.ctx)
			if err != nil {
				// Check if context was cancelled
				if c.ctx.Err() != nil {
					return
				}

				// Log error but continue
				fmt.Printf("Error reading Kafka message: %v\n", err)
				continue
			}

			// Process message
			if err := c.handler(c.ctx, msg); err != nil {
				fmt.Printf("Error handling Kafka message: %v\n", err)
				// Could implement retry logic or DLQ here
			}
		}
	}()
}

// StartWithSync starts the consumer with synchronous message processing
func (c *Consumer) StartWithSync() {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for {
			select {
			case <-c.ctx.Done():
				return
			default:
			}

			msg, err := c.reader.FetchMessage(c.ctx)
			if err != nil {
				if c.ctx.Err() != nil {
					return
				}
				fmt.Printf("Error fetching Kafka message: %v\n", err)
				continue
			}

			if err := c.handler(c.ctx, msg); err != nil {
				fmt.Printf("Error handling Kafka message: %v\n", err)
				continue
			}

			// Commit message after successful processing
			if err := c.reader.CommitMessages(c.ctx, msg); err != nil {
				fmt.Printf("Error committing Kafka message: %v\n", err)
			}
		}
	}()
}

// Stop stops the consumer gracefully
func (c *Consumer) Stop() error {
	c.cancel()
	c.wg.Wait()
	return c.reader.Close()
}

// Pause pauses the consumer
func (c *Consumer) Pause() {
	c.reader.Pause()
}

// Resume resumes the consumer
func (c *Consumer) Resume() {
	c.reader.Resume()
}

// SetOffset sets the offset to read from
func (c *Consumer) SetOffset(offset int64) error {
	return c.reader.SetOffset(offset)
}

// Lag returns the current lag of the consumer
func (c *Consumer) Lag() (int64, error) {
	lag, err := c.reader.Lag()
	if err != nil {
		return 0, fmt.Errorf("failed to get consumer lag: %w", err)
	}
	return lag, nil
}

// Stats returns consumer statistics
func (c *Consumer) Stats() kafka.ReaderStats {
	return c.reader.Stats()
}

// MessageConsumer creates a consumer function that handles specific message types
func MessageConsumer(handler func(ctx context.Context, key []byte, value []byte) error) MessageHandler {
	return func(ctx context.Context, msg kafka.Message) error {
		return handler(ctx, msg.Key, msg.Value)
	}
}

// JSONConsumer creates a consumer function that handles JSON messages
func JSONConsumer(handler func(ctx context.Context, key []byte, value interface{}) error) MessageHandler {
	return func(ctx context.Context, msg kafka.Message) error {
		var value interface{}
		if err := json.Unmarshal(msg.Value, &value); err != nil {
			return fmt.Errorf("failed to unmarshal JSON message: %w", err)
		}
		return handler(ctx, msg.Key, value)
	}
}

// TypedConsumer creates a consumer function that handles typed JSON messages
func TypedConsumer[T any](handler func(ctx context.Context, key []byte, value *T) error) MessageHandler {
	return func(ctx context.Context, msg kafka.Message) error {
		var value T
		if err := json.Unmarshal(msg.Value, &value); err != nil {
			return fmt.Errorf("failed to unmarshal typed message: %w", err)
		}
		return handler(ctx, msg.Key, &value)
	}
}

// MultiTopicConsumer consumes from multiple topics with different handlers
type MultiTopicConsumer struct {
	consumers map[string]*Consumer
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// NewMultiTopicConsumer creates a consumer that handles multiple topics
func NewMultiTopicConsumer(brokers []string, handlers map[string]MessageHandler) *MultiTopicConsumer {
	consumers := make(map[string]*Consumer)

	for topic, handler := range handlers {
		consumer := NewConsumerWithDefaults(brokers, topic, topic+"-consumer", handler)
		consumers[topic] = consumer
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &MultiTopicConsumer{
		consumers: consumers,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start starts all topic consumers
func (m *MultiTopicConsumer) Start() {
	for _, consumer := range m.consumers {
		consumer.Start()
	}
}

// Stop stops all topic consumers
func (m *MultiTopicConsumer) Stop() error {
	m.cancel()
	m.wg.Wait()

	var errs []error
	for _, consumer := range m.consumers {
		if err := consumer.Stop(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors stopping consumers: %v", errs)
	}
	return nil
}

// CreateTopics creates Kafka topics if they don't exist
func CreateTopics(brokers []string, topics []string) error {
	conn, err := kafka.DialLeader(context.Background(), "tcp", brokers[0], "__Aurontopics", 1)
	if err != nil {
		return fmt.Errorf("failed to dial Kafka leader: %w", err)
	}
	defer conn.Close()

	topicConfigs := make([]kafka.TopicConfig, len(topics))
	for i, topic := range topics {
		topicConfigs[i] = kafka.TopicConfig{
			Topic:             topic,
			NumPartitions:     6,
			ReplicationFactor: 1,
		}
	}

	err = conn.CreateTopics(topicConfigs...)
	if err != nil {
		return fmt.Errorf("failed to create topics: %w", err)
	}

	return nil
}

// EnsureTopics ensures all required topics exist
func EnsureTopics(brokers []string, requiredTopics []string) error {
	conn, err := kafka.Dial("tcp", brokers[0])
	if err != nil {
		return fmt.Errorf("failed to dial Kafka: %w", err)
	}
	defer conn.Close()

	// Get existing topics
	existingTopics, err := conn.Topics()
	if err != nil {
		return fmt.Errorf("failed to get topics: %w", err)
	}

	// Create missing topics
	var topicsToCreate []string
	for _, topic := range requiredTopics {
		found := false
		for _, existing := range existingTopics {
			if topic == existing {
				found = true
				break
			}
		}
		if !found {
			topicsToCreate = append(topicsToCreate, topic)
		}
	}

	if len(topicsToCreate) > 0 {
		return CreateTopics(brokers, topicsToCreate)
	}

	return nil
}
