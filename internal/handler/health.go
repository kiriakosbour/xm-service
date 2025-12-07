package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	db *sql.DB
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(db *sql.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status   string            `json:"status"`
	Services map[string]string `json:"services,omitempty"`
}

// Live handles GET /health/live - basic liveness check
func (h *HealthHandler) Live(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(HealthResponse{Status: "ok"})
}

// Ready handles GET /health/ready - readiness check including dependencies
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	services := make(map[string]string)
	status := http.StatusOK
	overallStatus := "ok"

	// Check database connectivity
	if err := h.db.PingContext(r.Context()); err != nil {
		services["database"] = "unhealthy: " + err.Error()
		status = http.StatusServiceUnavailable
		overallStatus = "unhealthy"
	} else {
		services["database"] = "healthy"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(HealthResponse{
		Status:   overallStatus,
		Services: services,
	})
}
