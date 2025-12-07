package service

import (
	"context"
	"testing"

	"xm-company-service/internal/core"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockRepository is a mock implementation of core.Repository
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, company *core.Company) error {
	args := m.Called(ctx, company)
	return args.Error(0)
}

func (m *MockRepository) GetByID(ctx context.Context, id uuid.UUID) (*core.Company, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*core.Company), args.Error(1)
}

func (m *MockRepository) GetByName(ctx context.Context, name string) (*core.Company, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*core.Company), args.Error(1)
}

func (m *MockRepository) Update(ctx context.Context, company *core.Company) error {
	args := m.Called(ctx, company)
	return args.Error(0)
}

func (m *MockRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockEventProducer is a mock implementation of core.EventProducer
type MockEventProducer struct {
	mock.Mock
}

func (m *MockEventProducer) Publish(ctx context.Context, eventType string, payload interface{}) error {
	args := m.Called(ctx, eventType, payload)
	return args.Error(0)
}

func (m *MockEventProducer) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestCompanyService_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		repo := new(MockRepository)
		producer := new(MockEventProducer)
		svc := NewCompanyService(repo, producer)

		input := &core.Company{
			Name:       "TestCo",
			Employees:  10,
			Registered: true,
			Type:       core.TypeCorporations,
		}

		// Name uniqueness check returns nil (not found)
		repo.On("GetByName", ctx, "TestCo").Return(nil, nil)
		// Create succeeds
		repo.On("Create", ctx, mock.AnythingOfType("*core.Company")).Return(nil)
		// Event is published
		producer.On("Publish", ctx, "CompanyCreated", mock.AnythingOfType("*core.Company")).Return(nil)

		result, err := svc.Create(ctx, input)

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, result.ID)
		assert.Equal(t, "TestCo", result.Name)
		repo.AssertExpectations(t)
		producer.AssertExpectations(t)
	})

	t.Run("duplicate name", func(t *testing.T) {
		repo := new(MockRepository)
		producer := new(MockEventProducer)
		svc := NewCompanyService(repo, producer)

		existing := &core.Company{ID: uuid.New(), Name: "ExistingCo"}
		input := &core.Company{
			Name:       "ExistingCo",
			Employees:  10,
			Registered: true,
			Type:       core.TypeCorporations,
		}

		repo.On("GetByName", ctx, "ExistingCo").Return(existing, nil)

		result, err := svc.Create(ctx, input)

		require.Error(t, err)
		assert.Equal(t, core.ErrDuplicateName, err)
		assert.Nil(t, result)
	})

	t.Run("validation error - name too long", func(t *testing.T) {
		repo := new(MockRepository)
		producer := new(MockEventProducer)
		svc := NewCompanyService(repo, producer)

		input := &core.Company{
			Name:       "ThisNameIsWayTooLong",
			Employees:  10,
			Registered: true,
			Type:       core.TypeCorporations,
		}

		result, err := svc.Create(ctx, input)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "15 characters")
		assert.Nil(t, result)
	})
}

func TestCompanyService_Get(t *testing.T) {
	ctx := context.Background()

	t.Run("found", func(t *testing.T) {
		repo := new(MockRepository)
		producer := new(MockEventProducer)
		svc := NewCompanyService(repo, producer)

		id := uuid.New()
		expected := &core.Company{
			ID:         id,
			Name:       "TestCo",
			Employees:  10,
			Registered: true,
			Type:       core.TypeCorporations,
		}

		repo.On("GetByID", ctx, id).Return(expected, nil)

		result, err := svc.Get(ctx, id)

		require.NoError(t, err)
		assert.Equal(t, expected, result)
	})

	t.Run("not found", func(t *testing.T) {
		repo := new(MockRepository)
		producer := new(MockEventProducer)
		svc := NewCompanyService(repo, producer)

		id := uuid.New()
		repo.On("GetByID", ctx, id).Return(nil, core.ErrNotFound)

		result, err := svc.Get(ctx, id)

		require.Error(t, err)
		assert.Equal(t, core.ErrNotFound, err)
		assert.Nil(t, result)
	})
}

func TestCompanyService_Patch(t *testing.T) {
	ctx := context.Background()

	t.Run("successful update", func(t *testing.T) {
		repo := new(MockRepository)
		producer := new(MockEventProducer)
		svc := NewCompanyService(repo, producer)

		id := uuid.New()
		existing := &core.Company{
			ID:         id,
			Name:       "OldName",
			Employees:  10,
			Registered: true,
			Type:       core.TypeCorporations,
		}

		updates := map[string]interface{}{
			"name":      "NewName",
			"employees": float64(20),
		}

		repo.On("GetByID", ctx, id).Return(existing, nil)
		repo.On("GetByName", ctx, "NewName").Return(nil, nil)
		repo.On("Update", ctx, mock.AnythingOfType("*core.Company")).Return(nil)
		producer.On("Publish", ctx, "CompanyUpdated", mock.AnythingOfType("*core.Company")).Return(nil)

		result, err := svc.Patch(ctx, id, updates)

		require.NoError(t, err)
		assert.Equal(t, "NewName", result.Name)
		assert.Equal(t, 20, result.Employees)
	})

	t.Run("not found", func(t *testing.T) {
		repo := new(MockRepository)
		producer := new(MockEventProducer)
		svc := NewCompanyService(repo, producer)

		id := uuid.New()
		repo.On("GetByID", ctx, id).Return(nil, core.ErrNotFound)

		result, err := svc.Patch(ctx, id, map[string]interface{}{"name": "NewName"})

		require.Error(t, err)
		assert.Equal(t, core.ErrNotFound, err)
		assert.Nil(t, result)
	})
}

func TestCompanyService_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("successful delete", func(t *testing.T) {
		repo := new(MockRepository)
		producer := new(MockEventProducer)
		svc := NewCompanyService(repo, producer)

		id := uuid.New()
		existing := &core.Company{
			ID:   id,
			Name: "ToDelete",
		}

		repo.On("GetByID", ctx, id).Return(existing, nil)
		repo.On("Delete", ctx, id).Return(nil)
		producer.On("Publish", ctx, "CompanyDeleted", mock.Anything).Return(nil)

		err := svc.Delete(ctx, id)

		require.NoError(t, err)
		repo.AssertExpectations(t)
		producer.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		repo := new(MockRepository)
		producer := new(MockEventProducer)
		svc := NewCompanyService(repo, producer)

		id := uuid.New()
		repo.On("GetByID", ctx, id).Return(nil, core.ErrNotFound)

		err := svc.Delete(ctx, id)

		require.Error(t, err)
		assert.Equal(t, core.ErrNotFound, err)
	})
}
