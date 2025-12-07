package service

import (
	"context"
	"errors"
	"xm/internal/core"

	"github.com/google/uuid"
)

type CompanyService struct {
	repo     core.Repository
	producer core.EventProducer
}

func NewCompanyService(r core.Repository, p core.EventProducer) *CompanyService {
	return &CompanyService{
		repo:     r,
		producer: p,
	}
}

func (s *CompanyService) Create(ctx context.Context, c *core.Company) (*core.Company, error) {
	// Unique Name Check [cite: 12]
	existing, err := s.repo.GetByName(ctx, c.Name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("company name already exists")
	}

	if err := c.Validate(); err != nil {
		return nil, err
	}

	c.ID = uuid.New() // Generate ID [cite: 11]

	if err := s.repo.Create(ctx, c); err != nil {
		return nil, err
	}

	// Emit Event
	_ = s.producer.Publish(ctx, "CompanyCreated", c)

	return c, nil
}

func (s *CompanyService) Get(ctx context.Context, id uuid.UUID) (*core.Company, error) {
	return s.repo.GetByID(ctx, id)
}

// Patch performs a partial update
func (s *CompanyService) Patch(ctx context.Context, id uuid.UUID, updates map[string]interface{}) (*core.Company, error) {
	// 1. Fetch current state
	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 2. Apply updates safely
	if v, ok := updates["name"].(string); ok {
		// specific check for uniqueness if name changes
		if v != current.Name {
			existing, err := s.repo.GetByName(ctx, v)
			if err != nil {
				return nil, err
			}
			if existing != nil {
				return nil, errors.New("company name already exists")
			}
			current.Name = v
		}
	}

	if v, ok := updates["description"].(string); ok {
		current.Description = &v
	}

	// Handle number types (JSON unmarshals numbers as float64 usually)
	if v, ok := updates["employees"].(float64); ok {
		current.Employees = int(v)
	} else if v, ok := updates["employees"].(int); ok {
		current.Employees = v
	}

	if v, ok := updates["registered"].(bool); ok {
		current.Registered = v
	}

	if v, ok := updates["type"].(string); ok {
		current.Type = core.CompanyType(v)
	}

	// 3. Validate new state
	if err := current.Validate(); err != nil {
		return nil, err
	}

	// 4. Save
	if err := s.repo.Update(ctx, current); err != nil {
		return nil, err
	}

	// 5. Emit Event
	_ = s.producer.Publish(ctx, "CompanyPatched", current)

	return current, nil
}

func (s *CompanyService) Delete(ctx context.Context, id uuid.UUID) error {
	// Check existence
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return err
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	// Emit Event
	payload := map[string]string{"id": id.String()}
	_ = s.producer.Publish(ctx, "CompanyDeleted", payload)

	return nil
}
