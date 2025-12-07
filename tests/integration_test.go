//go:build integration
// +build integration

package tests

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"xm-company-service/internal/core"
	"xm-company-service/internal/handler"
	"xm-company-service/internal/middleware"
	"xm-company-service/internal/platform/kafka"
	"xm-company-service/internal/platform/postgres"
	"xm-company-service/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// IntegrationTestSuite contains all integration tests
type IntegrationTestSuite struct {
	suite.Suite
	db      *sql.DB
	repo    *postgres.Repository
	svc     *service.CompanyService
	handler *handler.Handler
	router  *chi.Mux
}

func (s *IntegrationTestSuite) SetupSuite() {
	// Get database URL from environment or use default
	dbURL := os.Getenv("TEST_DB_URL")
	if dbURL == "" {
		dbURL = "postgres://xm_user:xm_password@localhost:5432/xm_test?sslmode=disable"
	}

	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	require.NoError(s.T(), err)

	// Wait for database to be ready
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for {
		if err := db.PingContext(ctx); err == nil {
			break
		}
		select {
		case <-ctx.Done():
			s.T().Fatal("Database not ready")
		case <-time.After(time.Second):
			continue
		}
	}

	s.db = db
	s.repo = postgres.NewRepository(db)

	// Run migrations
	err = s.repo.Migrate(context.Background())
	require.NoError(s.T(), err)

	// Create service with no-op producer
	producer := kafka.NewNoOpProducer()
	s.svc = service.NewCompanyService(s.repo, producer)
	s.handler = handler.NewHandler(s.svc)

	// Setup router
	s.router = chi.NewRouter()
	s.router.Get("/companies/{id}", s.handler.Get)
	s.router.Group(func(r chi.Router) {
		r.Use(middleware.JWTAuth)
		r.Post("/companies", s.handler.Create)
		r.Patch("/companies/{id}", s.handler.Patch)
		r.Delete("/companies/{id}", s.handler.Delete)
	})
}

func (s *IntegrationTestSuite) TearDownSuite() {
	if s.db != nil {
		s.db.Close()
	}
}

func (s *IntegrationTestSuite) SetupTest() {
	// Clean database before each test
	_, err := s.db.Exec("DELETE FROM companies")
	require.NoError(s.T(), err)
}

func (s *IntegrationTestSuite) TestCreateCompany() {
	body := `{
		"name": "TestCompany",
		"description": "A test company",
		"employees": 100,
		"registered": true,
		"type": "Corporations"
	}`

	req := httptest.NewRequest(http.MethodPost, "/companies", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()

	s.router.ServeHTTP(rec, req)

	assert.Equal(s.T(), http.StatusCreated, rec.Code)

	var company core.Company
	err := json.Unmarshal(rec.Body.Bytes(), &company)
	require.NoError(s.T(), err)

	assert.NotEqual(s.T(), uuid.Nil, company.ID)
	assert.Equal(s.T(), "TestCompany", company.Name)
	assert.Equal(s.T(), 100, company.Employees)
	assert.True(s.T(), company.Registered)
	assert.Equal(s.T(), core.TypeCorporations, company.Type)
}

func (s *IntegrationTestSuite) TestCreateCompanyDuplicateName() {
	// Create first company
	body := `{"name": "UniqueName", "employees": 10, "registered": true, "type": "NonProfit"}`

	req := httptest.NewRequest(http.MethodPost, "/companies", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()

	s.router.ServeHTTP(rec, req)
	require.Equal(s.T(), http.StatusCreated, rec.Code)

	// Try to create duplicate
	req = httptest.NewRequest(http.MethodPost, "/companies", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")
	rec = httptest.NewRecorder()

	s.router.ServeHTTP(rec, req)
	assert.Equal(s.T(), http.StatusConflict, rec.Code)
}

func (s *IntegrationTestSuite) TestGetCompany() {
	// Create a company first
	company := &core.Company{
		Name:       "GetTest",
		Employees:  50,
		Registered: true,
		Type:       core.TypeCooperative,
	}

	created, err := s.svc.Create(context.Background(), company)
	require.NoError(s.T(), err)

	// Get the company
	req := httptest.NewRequest(http.MethodGet, "/companies/"+created.ID.String(), nil)
	rec := httptest.NewRecorder()

	s.router.ServeHTTP(rec, req)

	assert.Equal(s.T(), http.StatusOK, rec.Code)

	var result core.Company
	err = json.Unmarshal(rec.Body.Bytes(), &result)
	require.NoError(s.T(), err)

	assert.Equal(s.T(), created.ID, result.ID)
	assert.Equal(s.T(), "GetTest", result.Name)
}

func (s *IntegrationTestSuite) TestGetCompanyNotFound() {
	id := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/companies/"+id.String(), nil)
	rec := httptest.NewRecorder()

	s.router.ServeHTTP(rec, req)

	assert.Equal(s.T(), http.StatusNotFound, rec.Code)
}

func (s *IntegrationTestSuite) TestPatchCompany() {
	// Create a company first
	company := &core.Company{
		Name:       "PatchTest",
		Employees:  10,
		Registered: false,
		Type:       core.TypeNonProfit,
	}

	created, err := s.svc.Create(context.Background(), company)
	require.NoError(s.T(), err)

	// Patch the company
	body := `{"employees": 25, "registered": true}`
	req := httptest.NewRequest(http.MethodPatch, "/companies/"+created.ID.String(), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()

	s.router.ServeHTTP(rec, req)

	assert.Equal(s.T(), http.StatusOK, rec.Code)

	var result core.Company
	err = json.Unmarshal(rec.Body.Bytes(), &result)
	require.NoError(s.T(), err)

	assert.Equal(s.T(), "PatchTest", result.Name) // unchanged
	assert.Equal(s.T(), 25, result.Employees)     // updated
	assert.True(s.T(), result.Registered)         // updated
}

func (s *IntegrationTestSuite) TestDeleteCompany() {
	// Create a company first
	company := &core.Company{
		Name:       "DeleteTest",
		Employees:  5,
		Registered: true,
		Type:       core.TypeSoleProprietorship,
	}

	created, err := s.svc.Create(context.Background(), company)
	require.NoError(s.T(), err)

	// Delete the company
	req := httptest.NewRequest(http.MethodDelete, "/companies/"+created.ID.String(), nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()

	s.router.ServeHTTP(rec, req)

	assert.Equal(s.T(), http.StatusNoContent, rec.Code)

	// Verify it's deleted
	req = httptest.NewRequest(http.MethodGet, "/companies/"+created.ID.String(), nil)
	rec = httptest.NewRecorder()

	s.router.ServeHTTP(rec, req)
	assert.Equal(s.T(), http.StatusNotFound, rec.Code)
}

func (s *IntegrationTestSuite) TestUnauthorizedAccess() {
	body := `{"name": "Unauthorized", "employees": 10, "registered": true, "type": "Corporations"}`

	// No Authorization header
	req := httptest.NewRequest(http.MethodPost, "/companies", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	s.router.ServeHTTP(rec, req)

	assert.Equal(s.T(), http.StatusUnauthorized, rec.Code)
}

func (s *IntegrationTestSuite) TestValidationErrors() {
	testCases := []struct {
		name     string
		body     string
		expected string
	}{
		{
			name:     "name too long",
			body:     `{"name": "ThisNameIsWayTooLong", "employees": 10, "registered": true, "type": "Corporations"}`,
			expected: "15 characters",
		},
		{
			name:     "invalid type",
			body:     `{"name": "ValidName", "employees": 10, "registered": true, "type": "InvalidType"}`,
			expected: "invalid company type",
		},
		{
			name:     "negative employees",
			body:     `{"name": "ValidName", "employees": -5, "registered": true, "type": "Corporations"}`,
			expected: "negative",
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/companies", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer test-token")
			rec := httptest.NewRecorder()

			s.router.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Contains(t, rec.Body.String(), tc.expected)
		})
	}
}

func (s *IntegrationTestSuite) TestAllCompanyTypes() {
	types := []core.CompanyType{
		core.TypeCorporations,
		core.TypeNonProfit,
		core.TypeCooperative,
		core.TypeSoleProprietorship,
	}

	for i, companyType := range types {
		s.T().Run(string(companyType), func(t *testing.T) {
			body := fmt.Sprintf(`{"name": "Type%d", "employees": 10, "registered": true, "type": "%s"}`, i, companyType)

			req := httptest.NewRequest(http.MethodPost, "/companies", bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer test-token")
			rec := httptest.NewRecorder()

			s.router.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusCreated, rec.Code)
		})
	}
}

func TestIntegrationSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	suite.Run(t, new(IntegrationTestSuite))
}
