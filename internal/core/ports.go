package core

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines database operations
type Repository interface {
	Create(ctx context.Context, company *Company) error
	GetByID(ctx context.Context, id uuid.UUID) (*Company, error)
	GetByName(ctx context.Context, name string) (*Company, error) // For uniqueness check
	Update(ctx context.Context, company *Company) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// EventProducer defines the contract for sending events (Kafka)
type EventProducer interface {
	Publish(ctx context.Context, eventType string, payload interface{}) error
	Close() error
}
