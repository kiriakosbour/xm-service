package handler

import (
	"encoding/json"
	"net/http"
	"xm/internal/core"
	"xm/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	svc *service.CompanyService
}

func NewHandler(svc *service.CompanyService) *Handler {
	return &Handler{svc: svc}
}

// POST /companies
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var input core.Company
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	created, err := h.svc.Create(r.Context(), &input)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "company name already exists" || err.Error() == "name must be 15 characters or fewer" {
			status = http.StatusBadRequest
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

// GET /companies/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	company, err := h.svc.Get(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(company)
}

// PATCH /companies/{id}
func (h *Handler) Patch(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	// Decode into map for partial update
	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	updatedCompany, err := h.svc.Patch(r.Context(), id, updates)
	if err != nil {
		// Determine status code based on error
		status := http.StatusInternalServerError
		if err.Error() == "company not found" {
			status = http.StatusNotFound
		} else if err.Error() == "company name already exists" {
			status = http.StatusBadRequest
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedCompany)
}

// DELETE /companies/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	err = h.svc.Delete(r.Context(), id)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "company not found" {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
