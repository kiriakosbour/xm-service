package postgres

import (
	"context"
	"database/sql"
	"errors"

	"xm-company-service/internal/core"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Repository implements core.Repository for PostgreSQL
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new PostgreSQL repository
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new company into the database
func (r *Repository) Create(ctx context.Context, c *core.Company) error {
	query := `
		INSERT INTO companies (id, name, description, employees, registered, type)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.db.ExecContext(ctx, query,
		c.ID, c.Name, c.Description, c.Employees, c.Registered, c.Type,
	)

	if err != nil {
		// Check for unique constraint violation
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" { // unique_violation
				return core.ErrDuplicateName
			}
		}
		return err
	}

	return nil
}

// GetByID retrieves a company by its UUID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*core.Company, error) {
	query := `
		SELECT id, name, description, employees, registered, type 
		FROM companies 
		WHERE id = $1`

	var c core.Company
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&c.ID, &c.Name, &c.Description, &c.Employees, &c.Registered, &c.Type,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, core.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &c, nil
}

// GetByName retrieves a company by its name (for uniqueness check)
func (r *Repository) GetByName(ctx context.Context, name string) (*core.Company, error) {
	query := `
		SELECT id, name, description, employees, registered, type 
		FROM companies 
		WHERE name = $1`

	var c core.Company
	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&c.ID, &c.Name, &c.Description, &c.Employees, &c.Registered, &c.Type,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil // Not found is acceptable for uniqueness checks
	}
	if err != nil {
		return nil, err
	}

	return &c, nil
}

// Update modifies an existing company
func (r *Repository) Update(ctx context.Context, c *core.Company) error {
	query := `
		UPDATE companies 
		SET name = $1, description = $2, employees = $3, registered = $4, type = $5
		WHERE id = $6`

	result, err := r.db.ExecContext(ctx, query,
		c.Name, c.Description, c.Employees, c.Registered, c.Type, c.ID,
	)
	if err != nil {
		// Check for unique constraint violation on name update
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" {
				return core.ErrDuplicateName
			}
		}
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return core.ErrNotFound
	}

	return nil
}

// Delete removes a company by ID
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM companies WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return core.ErrNotFound
	}

	return nil
}

// Migrate creates the companies table if it doesn't exist
func (r *Repository) Migrate(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS companies (
			id UUID PRIMARY KEY,
			name VARCHAR(15) NOT NULL UNIQUE,
			description VARCHAR(3000),
			employees INT NOT NULL,
			registered BOOLEAN NOT NULL,
			type VARCHAR(50) NOT NULL CHECK (type IN ('Corporations', 'NonProfit', 'Cooperative', 'Sole Proprietorship'))
		)`

	_, err := r.db.ExecContext(ctx, query)
	return err
}
