package kafka

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

// Producer implements core.EventProducer for Kafka
type Producer struct {
	writer  *kafka.Writer
	enabled bool
}

// NewProducer creates a new Kafka producer
func NewProducer(brokers []string, topic string, enabled bool) *Producer {
	if !enabled {
		log.Println("Kafka producer disabled")
		return &Producer{enabled: false}
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    1, // Send immediately for this exercise
		BatchTimeout: 10 * time.Millisecond,
		RequiredAcks: kafka.RequireOne,
	}

	log.Printf("Kafka producer initialized: brokers=%v, topic=%s", brokers, topic)
	return &Producer{
		writer:  writer,
		enabled: true,
	}
}

// Event represents a company mutation event
type Event struct {
	Type      string      `json:"type"`
	Payload   interface{} `json:"payload"`
	Timestamp time.Time   `json:"timestamp"`
}

// Publish sends an event to Kafka
func (p *Producer) Publish(ctx context.Context, eventType string, payload interface{}) error {
	if !p.enabled {
		log.Printf("Kafka disabled, skipping event: %s", eventType)
		return nil
	}

	event := Event{
		Type:      eventType,
		Payload:   payload,
		Timestamp: time.Now().UTC(),
	}

	value, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal event: %v", err)
		return err
	}

	msg := kafka.Message{
		Key:   []byte(eventType),
		Value: value,
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		log.Printf("Failed to publish event %s: %v", eventType, err)
		return err
	}

	log.Printf("Event published: %s", eventType)
	return nil
}

// Close closes the Kafka writer
func (p *Producer) Close() error {
	if p.writer != nil {
		return p.writer.Close()
	}
	return nil
}

// NoOpProducer is a no-operation producer for testing
type NoOpProducer struct{}

// NewNoOpProducer creates a producer that does nothing
func NewNoOpProducer() *NoOpProducer {
	return &NoOpProducer{}
}

// Publish does nothing
func (p *NoOpProducer) Publish(ctx context.Context, eventType string, payload interface{}) error {
	log.Printf("NoOp event: %s", eventType)
	return nil
}

// Close does nothing
func (p *NoOpProducer) Close() error {
	return nil
}
