package kafka

import (
	"context"
	"encoding/json"
	"time"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(brokers []string, topic string) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Topic:    topic,
			Balancer: &kafka.LeastBytes{},
		},
	}
}

func (p *Producer) Publish(ctx context.Context, eventType string, payload interface{}) error {
	value, err := json.Marshal(map[string]interface{}{
		"type":      eventType,
		"payload":   payload,
		"timestamp": time.Now().UTC(),
	})
	if err != nil {
		return err
	}

	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(eventType), // Partition by event type or entity ID
		Value: value,
	})
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
