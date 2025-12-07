package repository

import (
	"database/sql"
	"xm/internal/core"

	"github.com/google/uuid"
)

type PostgresRepo struct {
	DB *sql.DB
}

func NewPostgresRepo(db *sql.DB) *PostgresRepo {
	return &PostgresRepo{DB: db}
}

func (r *PostgresRepo) Create(c *core.Company) error {
	query := `INSERT INTO companies (id, name, description, employees, registered, type) 
              VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.DB.Exec(query, c.ID, c.Name, c.Description, c.Employees, c.Registered, c.Type)
	return err
}

func (r *PostgresRepo) Get(id uuid.UUID) (*core.Company, error) {
	var c core.Company
	query := `SELECT id, name, description, employees, registered, type FROM companies WHERE id = $1`
	row := r.DB.QueryRow(query, id)

	err := row.Scan(&c.ID, &c.Name, &c.Description, &c.Employees, &c.Registered, &c.Type)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// Update handles the PATCH requirement [cite: 2] by expecting the service to merge fields first
func (r *PostgresRepo) Update(c *core.Company) error {
	query := `UPDATE companies SET name=$1, description=$2, employees=$3, registered=$4, type=$5 WHERE id=$6`
	_, err := r.DB.Exec(query, c.Name, c.Description, c.Employees, c.Registered, c.Type, c.ID)
	return err
}

func (r *PostgresRepo) Delete(id uuid.UUID) error {
	_, err := r.DB.Exec(`DELETE FROM companies WHERE id = $1`, id)
	return err
}

func (r *PostgresRepo) GetByName(name string) (*core.Company, error) {
	// Implemented to check uniqueness [cite: 2]
	var c core.Company
	query := `SELECT id FROM companies WHERE name = $1`
	err := r.DB.QueryRow(query, name).Scan(&c.ID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &c, err
}
