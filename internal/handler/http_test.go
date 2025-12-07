package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"xm-company-service/internal/core"
	"xm-company-service/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockRepository for testing
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

// MockEventProducer for testing
type MockEventProducer struct {
	mock.Mock
}

func (m *MockEventProducer) Publish(ctx context.Context, eventType string, payload interface{}) error {
	args := m.Called(ctx, eventType, payload)
	return args.Error(0)
}

func (m *MockEventProducer) Close() error {
	return nil
}

func setupTestHandler() (*Handler, *MockRepository, *MockEventProducer) {
	repo := new(MockRepository)
	producer := new(MockEventProducer)
	svc := service.NewCompanyService(repo, producer)
	handler := NewHandler(svc)
	return handler, repo, producer
}

func TestHandler_Create(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		h, repo, producer := setupTestHandler()

		repo.On("GetByName", mock.Anything, "TestCo").Return(nil, nil)
		repo.On("Create", mock.Anything, mock.AnythingOfType("*core.Company")).Return(nil)
		producer.On("Publish", mock.Anything, "CompanyCreated", mock.Anything).Return(nil)

		body := `{"name":"TestCo","employees":10,"registered":true,"type":"Corporations"}`
		req := httptest.NewRequest(http.MethodPost, "/companies", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		h.Create(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)

		var response core.Company
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "TestCo", response.Name)
		assert.NotEqual(t, uuid.Nil, response.ID)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		h, _, _ := setupTestHandler()

		req := httptest.NewRequest(http.MethodPost, "/companies", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		h.Create(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("validation error", func(t *testing.T) {
		h, _, _ := setupTestHandler()

		// Name too long
		body := `{"name":"ThisNameIsTooLongForOurLimit","employees":10,"registered":true,"type":"Corporations"}`
		req := httptest.NewRequest(http.MethodPost, "/companies", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		h.Create(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestHandler_Get(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		h, repo, _ := setupTestHandler()

		id := uuid.New()
		expected := &core.Company{
			ID:         id,
			Name:       "TestCo",
			Employees:  10,
			Registered: true,
			Type:       core.TypeCorporations,
		}

		repo.On("GetByID", mock.Anything, id).Return(expected, nil)

		req := httptest.NewRequest(http.MethodGet, "/companies/"+id.String(), nil)
		rec := httptest.NewRecorder()

		// Setup chi router context
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", id.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		h.Get(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response core.Company
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, expected.Name, response.Name)
	})

	t.Run("not found", func(t *testing.T) {
		h, repo, _ := setupTestHandler()

		id := uuid.New()
		repo.On("GetByID", mock.Anything, id).Return(nil, core.ErrNotFound)

		req := httptest.NewRequest(http.MethodGet, "/companies/"+id.String(), nil)
		rec := httptest.NewRecorder()

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", id.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		h.Get(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("invalid UUID", func(t *testing.T) {
		h, _, _ := setupTestHandler()

		req := httptest.NewRequest(http.MethodGet, "/companies/invalid-uuid", nil)
		rec := httptest.NewRecorder()

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "invalid-uuid")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		h.Get(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestHandler_Delete(t *testing.T) {
	t.Run("successful delete", func(t *testing.T) {
		h, repo, producer := setupTestHandler()

		id := uuid.New()
		existing := &core.Company{ID: id, Name: "ToDelete"}

		repo.On("GetByID", mock.Anything, id).Return(existing, nil)
		repo.On("Delete", mock.Anything, id).Return(nil)
		producer.On("Publish", mock.Anything, "CompanyDeleted", mock.Anything).Return(nil)

		req := httptest.NewRequest(http.MethodDelete, "/companies/"+id.String(), nil)
		rec := httptest.NewRecorder()

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", id.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		h.Delete(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("not found", func(t *testing.T) {
		h, repo, _ := setupTestHandler()

		id := uuid.New()
		repo.On("GetByID", mock.Anything, id).Return(nil, core.ErrNotFound)

		req := httptest.NewRequest(http.MethodDelete, "/companies/"+id.String(), nil)
		rec := httptest.NewRecorder()

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", id.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		h.Delete(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestHandler_Patch(t *testing.T) {
	t.Run("successful patch", func(t *testing.T) {
		h, repo, producer := setupTestHandler()

		id := uuid.New()
		existing := &core.Company{
			ID:         id,
			Name:       "OldName",
			Employees:  10,
			Registered: true,
			Type:       core.TypeCorporations,
		}

		repo.On("GetByID", mock.Anything, id).Return(existing, nil)
		repo.On("GetByName", mock.Anything, "NewName").Return(nil, nil)
		repo.On("Update", mock.Anything, mock.AnythingOfType("*core.Company")).Return(nil)
		producer.On("Publish", mock.Anything, "CompanyUpdated", mock.Anything).Return(nil)

		body := `{"name":"NewName","employees":20}`
		req := httptest.NewRequest(http.MethodPatch, "/companies/"+id.String(), bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", id.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		h.Patch(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response core.Company
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "NewName", response.Name)
		assert.Equal(t, 20, response.Employees)
	})

	t.Run("empty update body", func(t *testing.T) {
		h, _, _ := setupTestHandler()

		id := uuid.New()

		req := httptest.NewRequest(http.MethodPatch, "/companies/"+id.String(), bytes.NewBufferString("{}"))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", id.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		h.Patch(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}
