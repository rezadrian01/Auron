package cmd

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"auron/payment-service/internal/domain"
	"auron/payment-service/internal/events"

	"github.com/segmentio/kafka-go"
)

var paymentTopics = []string{
	domain.TopicPaymentCreated,
	domain.TopicPaymentCompleted,
	domain.TopicPaymentFailed,
}

func setupKafkaPublisher(kafkaBrokers string) domain.EventPublisher {
	brokers := parseBrokers(kafkaBrokers)
	ensureTopics(brokers, paymentTopics)

	writers := make(map[string]*kafka.Writer, len(paymentTopics))
	for _, topic := range paymentTopics {
		writers[topic] = &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{},
			RequiredAcks: kafka.RequireOne,
			BatchTimeout: 10 * time.Millisecond,
		}
	}

	return events.NewKafkaPublisher(writers)
}

func setupKafkaConsumer(kafkaBrokers string, svc domain.PaymentService) *events.KafkaConsumer {
	brokers := parseBrokers(kafkaBrokers)
	return events.NewKafkaConsumer(brokers, domain.TopicOrderCreated, "payment-service", svc)
}

func startKafkaConsumer(ctx context.Context, consumer *events.KafkaConsumer) {
	consumer.Start(ctx)
	slog.Info("kafka consumer started", "topic", domain.TopicOrderCreated, "group", "payment-service")
}

func parseBrokers(kafkaBrokers string) []string {
	parts := strings.Split(kafkaBrokers, ",")
	brokers := make([]string, 0, len(parts))
	for _, b := range parts {
		if trimmed := strings.TrimSpace(b); trimmed != "" {
			brokers = append(brokers, trimmed)
		}
	}
	if len(brokers) == 0 {
		return []string{"localhost:9092"}
	}
	return brokers
}

func ensureTopics(brokers []string, topics []string) {
	if len(brokers) == 0 || len(topics) == 0 {
		return
	}

	conn, err := kafka.Dial("tcp", brokers[0])
	if err != nil {
		slog.Warn("kafka topic init skipped: cannot connect", "broker", brokers[0], "error", err)
		return
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		slog.Warn("kafka topic init skipped: cannot get controller", "error", err)
		return
	}

	controllerConn, err := kafka.Dial("tcp", controller.Host+":"+strconv.Itoa(controller.Port))
	if err != nil {
		slog.Warn("kafka topic init skipped: cannot connect to controller", "error", err)
		return
	}
	defer controllerConn.Close()

	configs := make([]kafka.TopicConfig, 0, len(topics))
	for _, topic := range topics {
		configs = append(configs, kafka.TopicConfig{
			Topic:             topic,
			NumPartitions:     3,
			ReplicationFactor: 1,
		})
	}

	if err := controllerConn.CreateTopics(configs...); err != nil {
		slog.Warn("kafka topic init failed", "topics", topics, "error", err)
		return
	}

	slog.Info("kafka topics ensured", "topics", topics)
}

func closeKafkaPublisher(publisher domain.EventPublisher) {
	if err := publisher.Close(); err != nil {
		slog.Warn("failed to close kafka publisher", "error", err)
	}
}
