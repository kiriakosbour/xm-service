package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"xm-company-service/internal/core"
	"xm-company-service/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for company operations
type Handler struct {
	svc *service.CompanyService
}

// NewHandler creates a new HTTP handler
func NewHandler(svc *service.CompanyService) *Handler {
	return &Handler{svc: svc}
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// CreateRequest represents the request body for creating a company
type CreateRequest struct {
	Name        string           `json:"name"`
	Description *string          `json:"description,omitempty"`
	Employees   int              `json:"employees"`
	Registered  bool             `json:"registered"`
	Type        core.CompanyType `json:"type"`
}

// Create handles POST /companies
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	company := &core.Company{
		Name:        req.Name,
		Description: req.Description,
		Employees:   req.Employees,
		Registered:  req.Registered,
		Type:        req.Type,
	}

	created, err := h.svc.Create(r.Context(), company)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, created, http.StatusCreated)
}

// Get handles GET /companies/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, "invalid UUID format", http.StatusBadRequest)
		return
	}

	company, err := h.svc.Get(r.Context(), id)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, company, http.StatusOK)
}

// Patch handles PATCH /companies/{id}
func (h *Handler) Patch(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, "invalid UUID format", http.StatusBadRequest)
		return
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		respondError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	// Don't allow updating ID
	delete(updates, "id")

	if len(updates) == 0 {
		respondError(w, "no fields to update", http.StatusBadRequest)
		return
	}

	updated, err := h.svc.Patch(r.Context(), id, updates)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, updated, http.StatusOK)
}

// Delete handles DELETE /companies/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, "invalid UUID format", http.StatusBadRequest)
		return
	}

	err = h.svc.Delete(r.Context(), id)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleServiceError maps service errors to HTTP status codes
func handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, core.ErrNotFound):
		respondError(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, core.ErrDuplicateName):
		respondError(w, err.Error(), http.StatusConflict)
	default:
		// Check for validation errors
		errMsg := err.Error()
		if isValidationError(errMsg) {
			respondError(w, errMsg, http.StatusBadRequest)
			return
		}
		log.Printf("Internal error: %v", err)
		respondError(w, "internal server error", http.StatusInternalServerError)
	}
}

// isValidationError checks if the error message indicates a validation error
func isValidationError(msg string) bool {
	validationPrefixes := []string{
		"name is required",
		"name must be",
		"description must be",
		"employees cannot be",
		"invalid company type",
		"registered",
	}
	for _, prefix := range validationPrefixes {
		if len(msg) >= len(prefix) && msg[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

// respondJSON writes a JSON response
func respondJSON(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// respondError writes an error response
func respondError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}
