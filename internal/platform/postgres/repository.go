package postgres

import (
	"context"
	"database/sql"
	"errors"
	"xm/internal/core"

	"github.com/google/uuid"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, c *core.Company) error {
	query := `
		INSERT INTO companies (id, name, description, employees, registered, type)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.db.ExecContext(ctx, query,
		c.ID, c.Name, c.Description, c.Employees, c.Registered, c.Type,
	)
	return err
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*core.Company, error) {
	query := `
		SELECT id, name, description, employees, registered, type 
		FROM companies WHERE id = $1`

	var c core.Company
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&c.ID, &c.Name, &c.Description, &c.Employees, &c.Registered, &c.Type,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("company not found")
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *Repository) GetByName(ctx context.Context, name string) (*core.Company, error) {
	query := `SELECT id FROM companies WHERE name = $1`
	var c core.Company
	err := r.db.QueryRowContext(ctx, query, name).Scan(&c.ID)

	if err == sql.ErrNoRows {
		return nil, nil // Not found is fine here
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *Repository) Update(ctx context.Context, c *core.Company) error {
	query := `
		UPDATE companies 
		SET name = $1, description = $2, employees = $3, registered = $4, type = $5
		WHERE id = $6`

	result, err := r.db.ExecContext(ctx, query,
		c.Name, c.Description, c.Employees, c.Registered, c.Type, c.ID,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("company not found")
	}
	return nil
}

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
		return errors.New("company not found")
	}
	return nil
}
