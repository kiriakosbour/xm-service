package service

import (
	"context"
	"errors"
	"log"

	"xm-company-service/internal/core"

	"github.com/google/uuid"
)

// CompanyService handles business logic for company operations
type CompanyService struct {
	repo     core.Repository
	producer core.EventProducer
}

// NewCompanyService creates a new company service
func NewCompanyService(repo core.Repository, producer core.EventProducer) *CompanyService {
	return &CompanyService{
		repo:     repo,
		producer: producer,
	}
}

// Create creates a new company
func (s *CompanyService) Create(ctx context.Context, c *core.Company) (*core.Company, error) {
	// Validate input
	if err := c.Validate(); err != nil {
		return nil, err
	}

	// Check for duplicate name
	existing, err := s.repo.GetByName(ctx, c.Name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, core.ErrDuplicateName
	}

	// Generate new UUID
	c.ID = uuid.New()

	// Persist
	if err := s.repo.Create(ctx, c); err != nil {
		return nil, err
	}

	// Emit event (don't fail the operation if event fails)
	if err := s.producer.Publish(ctx, "CompanyCreated", c); err != nil {
		log.Printf("Warning: failed to publish CompanyCreated event: %v", err)
	}

	return c, nil
}

// Get retrieves a company by ID
func (s *CompanyService) Get(ctx context.Context, id uuid.UUID) (*core.Company, error) {
	return s.repo.GetByID(ctx, id)
}

// PatchInput represents the fields that can be updated
type PatchInput struct {
	Name        *string           `json:"name,omitempty"`
	Description *string           `json:"description,omitempty"`
	Employees   *int              `json:"employees,omitempty"`
	Registered  *bool             `json:"registered,omitempty"`
	Type        *core.CompanyType `json:"type,omitempty"`
}

// Patch performs a partial update on a company
func (s *CompanyService) Patch(ctx context.Context, id uuid.UUID, updates map[string]interface{}) (*core.Company, error) {
	// Fetch current state
	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if err := applyUpdates(current, updates); err != nil {
		return nil, err
	}

	// Check for duplicate name if name is being changed
	if v, ok := updates["name"].(string); ok && v != current.Name {
		existing, err := s.repo.GetByName(ctx, v)
		if err != nil {
			return nil, err
		}
		if existing != nil && existing.ID != id {
			return nil, core.ErrDuplicateName
		}
	}

	// Validate updated entity
	if err := current.Validate(); err != nil {
		return nil, err
	}

	// Persist
	if err := s.repo.Update(ctx, current); err != nil {
		return nil, err
	}

	// Emit event
	if err := s.producer.Publish(ctx, "CompanyUpdated", current); err != nil {
		log.Printf("Warning: failed to publish CompanyUpdated event: %v", err)
	}

	return current, nil
}

// Delete removes a company by ID
func (s *CompanyService) Delete(ctx context.Context, id uuid.UUID) error {
	// Check existence first
	company, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Delete
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	// Emit event with deleted company info
	event := map[string]interface{}{
		"id":   id.String(),
		"name": company.Name,
	}
	if err := s.producer.Publish(ctx, "CompanyDeleted", event); err != nil {
		log.Printf("Warning: failed to publish CompanyDeleted event: %v", err)
	}

	return nil
}

// applyUpdates applies partial updates to a company
func applyUpdates(c *core.Company, updates map[string]interface{}) error {
	if v, ok := updates["name"]; ok {
		if name, ok := v.(string); ok {
			c.Name = name
		} else {
			return errors.New("name must be a string")
		}
	}

	if v, ok := updates["description"]; ok {
		if v == nil {
			c.Description = nil
		} else if desc, ok := v.(string); ok {
			c.Description = &desc
		} else {
			return errors.New("description must be a string or null")
		}
	}

	if v, ok := updates["employees"]; ok {
		switch emp := v.(type) {
		case float64:
			c.Employees = int(emp)
		case int:
			c.Employees = emp
		default:
			return errors.New("employees must be a number")
		}
	}

	if v, ok := updates["registered"]; ok {
		if reg, ok := v.(bool); ok {
			c.Registered = reg
		} else {
			return errors.New("registered must be a boolean")
		}
	}

	if v, ok := updates["type"]; ok {
		if t, ok := v.(string); ok {
			c.Type = core.CompanyType(t)
		} else {
			return errors.New("type must be a string")
		}
	}

	return nil
}
