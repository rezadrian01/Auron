// Package kafka provides reusable Kafka producer and consumer helpers for all Auron services.
package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

// ProducerConfig holds Kafka producer configuration
type ProducerConfig struct {
	Brokers []string
	Topic   string
}

// Producer wraps the Kafka writer with connection pooling and error handling
type Producer struct {
	writer *kafka.Writer
	topic  string
}

// NewProducer creates a new Kafka producer
func NewProducer(cfg *ProducerConfig) *Producer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        cfg.Topic,
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    1,
		BatchTimeout: 10 * time.Millisecond,
		RequiredAcks: kafka.RequireOne,
		Compression:  kafka.Snappy,
		// Retry settings
		MaxRetries:     3,
		RetryBackoff:   time.Millisecond * 100,
		Async:          false,
	}

	return &Producer{
		writer: writer,
		topic:  cfg.Topic,
	}
}

// NewProducerWithConfig creates a new Kafka producer with custom configuration
func NewProducerWithConfig(brokers []string, topic string, balancer kafka.Balancer) *Producer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     balancer,
		BatchSize:    1,
		BatchTimeout: 10 * time.Millisecond,
		RequiredAcks: kafka.RequireOne,
		Compression:  kafka.Snappy,
		MaxRetries:   3,
		RetryBackoff: time.Millisecond * 100,
	}

	return &Producer{
		writer: writer,
		topic:  topic,
	}
}

// Publish publishes a message to the Kafka topic
func (p *Producer) Publish(ctx context.Context, key []byte, value interface{}) error {
	var msgValue []byte
	var err error

	switch v := value.(type) {
	case string:
		msgValue = []byte(v)
	case []byte:
		msgValue = v
	default:
		msgValue, err = json.Marshal(v)
		if err != nil {
			return fmt.Errorf("failed to marshal message value: %w", err)
		}
	}

	msg := kafka.Message{
		Key:   key,
		Value: msgValue,
		Time:  time.Now(),
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

// PublishWithHeaders publishes a message with custom headers
func (p *Producer) PublishWithHeaders(ctx context.Context, key []byte, value interface{}, headers []kafka.Header) error {
	var msgValue []byte
	var err error

	switch v := value.(type) {
	case string:
		msgValue = []byte(v)
	case []byte:
		msgValue = v
	default:
		msgValue, err = json.Marshal(v)
		if err != nil {
			return fmt.Errorf("failed to marshal message value: %w", err)
		}
	}

	msg := kafka.Message{
		Key:    key,
		Value:  msgValue,
		Time:   time.Now(),
		Headers: headers,
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

// PublishJSON publishes a JSON message to the Kafka topic
func (p *Producer) PublishJSON(ctx context.Context, key []byte, value interface{}) error {
	msgValue, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON message: %w", err)
	}

	msg := kafka.Message{
		Key:   key,
		Value: msgValue,
		Time:  time.Now(),
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("failed to publish JSON message: %w", err)
	}

	return nil
}

// PublishToTopic publishes a message to a specific topic
func (p *Producer) PublishToTopic(ctx context.Context, topic string, key []byte, value interface{}) error {
	var msgValue []byte
	var err error

	switch v := value.(type) {
	case string:
		msgValue = []byte(v)
	case []byte:
		msgValue = v
	default:
		msgValue, err = json.Marshal(v)
		if err != nil {
			return fmt.Errorf("failed to marshal message value: %w", err)
		}
	}

	// Create a temporary writer for the specific topic
	writer := &kafka.Writer{
		Addr:         p.writer.Addr,
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		Compression:  kafka.Snappy,
	}
	defer writer.Close()

	msg := kafka.Message{
		Key:   key,
		Value: msgValue,
		Time:  time.Now(),
	}

	if err := writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("failed to publish message to topic %s: %w", topic, err)
	}

	return nil
}

// Close closes the producer
func (p *Producer) Close() error {
	return p.writer.Close()
}

// GetTopic returns the topic name
func (p *Producer) GetTopic() string {
	return p.topic
}
