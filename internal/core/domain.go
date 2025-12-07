package core

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// CompanyType represents the type of company
type CompanyType string

const (
	TypeCorporations       CompanyType = "Corporations"
	TypeNonProfit          CompanyType = "NonProfit"
	TypeCooperative        CompanyType = "Cooperative"
	TypeSoleProprietorship CompanyType = "Sole Proprietorship"
)

// ValidCompanyTypes contains all valid company types
var ValidCompanyTypes = []CompanyType{
	TypeCorporations,
	TypeNonProfit,
	TypeCooperative,
	TypeSoleProprietorship,
}

// Company represents the company entity
type Company struct {
	ID          uuid.UUID   `json:"id"`
	Name        string      `json:"name"`                  // Required, max 15 chars, unique
	Description *string     `json:"description,omitempty"` // Optional, max 3000 chars
	Employees   int         `json:"employees"`             // Required
	Registered  bool        `json:"registered"`            // Required
	Type        CompanyType `json:"type"`                  // Required
}

// Validate enforces business rules
func (c *Company) Validate() error {
	if c.Name == "" {
		return errors.New("name is required")
	}
	if len(c.Name) > 15 {
		return errors.New("name must be 15 characters or fewer")
	}

	if c.Description != nil && len(*c.Description) > 3000 {
		return errors.New("description must be 3000 characters or fewer")
	}

	if c.Employees < 0 {
		return errors.New("employees cannot be negative")
	}

	if !c.Type.IsValid() {
		return fmt.Errorf("invalid company type: %s", c.Type)
	}

	return nil
}

// IsValid checks if the company type is valid
func (ct CompanyType) IsValid() bool {
	switch ct {
	case TypeCorporations, TypeNonProfit, TypeCooperative, TypeSoleProprietorship:
		return true
	default:
		return false
	}
}

// CompanyEvent represents an event emitted on mutations
type CompanyEvent struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}
