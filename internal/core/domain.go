package core

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// CompanyType constants as defined in requirements [cite: 16]
type CompanyType string

const (
	TypeCorporations       CompanyType = "Corporations"
	TypeNonProfit          CompanyType = "NonProfit"
	TypeCooperative        CompanyType = "Cooperative"
	TypeSoleProprietorship CompanyType = "Sole Proprietorship"
)

// Company represents the data structure [cite: 10]
type Company struct {
	ID          uuid.UUID   `json:"id"`
	Name        string      `json:"name"`        // Required, 15 chars max, Unique [cite: 12]
	Description *string     `json:"description"` // Optional, 3000 chars max [cite: 13]
	Employees   int         `json:"employees"`   // Required [cite: 14]
	Registered  bool        `json:"registered"`  // Required [cite: 15]
	Type        CompanyType `json:"type"`        // Required [cite: 16]
}

// Validate enforces business rules regarding length and types
func (c *Company) Validate() error {
	if c.Name == "" {
		return errors.New("name is required")
	}
	if len(c.Name) > 15 {
		return errors.New("name must be 15 characters or fewer") // [cite: 12]
	}

	if c.Description != nil && len(*c.Description) > 3000 {
		return errors.New("description must be 3000 characters or fewer") // [cite: 13]
	}

	if c.Employees < 0 {
		return errors.New("amount of employees cannot be negative")
	}

	switch c.Type {
	case TypeCorporations, TypeNonProfit, TypeCooperative, TypeSoleProprietorship:
		// valid
	default:
		return fmt.Errorf("invalid company type: %s", c.Type) // [cite: 16]
	}

	return nil
}
