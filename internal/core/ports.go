package core

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// Repository defines the contract for data persistence
type Repository interface {
	Create(ctx context.Context, company *Company) error
	GetByID(ctx context.Context, id uuid.UUID) (*Company, error)
	GetByName(ctx context.Context, name string) (*Company, error)
	Update(ctx context.Context, company *Company) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// EventProducer defines the contract for publishing events
type EventProducer interface {
	Publish(ctx context.Context, eventType string, payload interface{}) error
	Close() error
}

// ErrNotFound is returned when a company is not found
var ErrNotFound = errors.New("company not found")

// ErrDuplicateName is returned when a company name already exists
var ErrDuplicateName = errors.New("company name already exists")
