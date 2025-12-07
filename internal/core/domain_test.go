package core

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompany_Validate(t *testing.T) {
	validDesc := "A valid description"

	tests := []struct {
		name    string
		company Company
		wantErr string
	}{
		{
			name: "valid company - all fields",
			company: Company{
				ID:          uuid.New(),
				Name:        "TestCompany",
				Description: &validDesc,
				Employees:   100,
				Registered:  true,
				Type:        TypeCorporations,
			},
			wantErr: "",
		},
		{
			name: "valid company - no description",
			company: Company{
				ID:         uuid.New(),
				Name:       "TestCompany",
				Employees:  50,
				Registered: false,
				Type:       TypeNonProfit,
			},
			wantErr: "",
		},
		{
			name: "valid company - max name length",
			company: Company{
				ID:         uuid.New(),
				Name:       "123456789012345", // exactly 15 chars
				Employees:  1,
				Registered: true,
				Type:       TypeCooperative,
			},
			wantErr: "",
		},
		{
			name: "invalid - empty name",
			company: Company{
				ID:         uuid.New(),
				Name:       "",
				Employees:  10,
				Registered: true,
				Type:       TypeCorporations,
			},
			wantErr: "name is required",
		},
		{
			name: "invalid - name too long",
			company: Company{
				ID:         uuid.New(),
				Name:       "1234567890123456", // 16 chars
				Employees:  10,
				Registered: true,
				Type:       TypeCorporations,
			},
			wantErr: "name must be 15 characters or fewer",
		},
		{
			name: "invalid - description too long",
			company: Company{
				ID:          uuid.New(),
				Name:        "TestCompany",
				Description: strPtr(strings.Repeat("a", 3001)),
				Employees:   10,
				Registered:  true,
				Type:        TypeCorporations,
			},
			wantErr: "description must be 3000 characters or fewer",
		},
		{
			name: "invalid - negative employees",
			company: Company{
				ID:         uuid.New(),
				Name:       "TestCompany",
				Employees:  -1,
				Registered: true,
				Type:       TypeCorporations,
			},
			wantErr: "employees cannot be negative",
		},
		{
			name: "invalid - invalid type",
			company: Company{
				ID:         uuid.New(),
				Name:       "TestCompany",
				Employees:  10,
				Registered: true,
				Type:       "InvalidType",
			},
			wantErr: "invalid company type: InvalidType",
		},
		{
			name: "valid - zero employees",
			company: Company{
				ID:         uuid.New(),
				Name:       "NewStartup",
				Employees:  0,
				Registered: true,
				Type:       TypeSoleProprietorship,
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.company.Validate()

			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestCompanyType_IsValid(t *testing.T) {
	tests := []struct {
		companyType CompanyType
		want        bool
	}{
		{TypeCorporations, true},
		{TypeNonProfit, true},
		{TypeCooperative, true},
		{TypeSoleProprietorship, true},
		{"Invalid", false},
		{"", false},
		{"corporation", false}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(string(tt.companyType), func(t *testing.T) {
			got := tt.companyType.IsValid()
			assert.Equal(t, tt.want, got)
		})
	}
}

func strPtr(s string) *string {
	return &s
}
