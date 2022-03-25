package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFlags_Validate(t *testing.T) {
	tt := []struct {
		name          string
		expectedError bool
		flags         Flags
	}{
		{
			name:          "valid flags",
			expectedError: false,
			flags: Flags{
				db:        "postgres://localhost",
				chain:     "cosmos-hub",
				chunkSize: 1,
				tables:    "auth,clients,validators",
			},
		},
		{
			name:          "negative chunk size",
			expectedError: true,
			flags: Flags{
				db:        "postgres://localhost",
				chain:     "cosmos-hub",
				chunkSize: -2,
				tables:    "auth,clients,validators",
			},
		},
		{
			name:          "zero chunk size",
			expectedError: true,
			flags: Flags{
				db:        "postgres://localhost",
				chain:     "cosmos-hub",
				chunkSize: 0,
				tables:    "auth,clients,validators",
			},
		},
		{
			name:          "empty connection string",
			expectedError: true,
			flags: Flags{
				db:        "",
				chain:     "cosmos-hub",
				chunkSize: 2,
				tables:    "auth,clients,validators",
			},
		},
		{
			name:          "empty chain name",
			expectedError: true,
			flags: Flags{
				db:        "postgres://localhost",
				chain:     "",
				chunkSize: 2,
				tables:    "auth,clients,validators",
			},
		},
	}

	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			err := test.flags.Validate()
			if test.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_GetOverrideTableMap(t *testing.T) {
	tt := []struct {
		name          string
		forceIndexes  string
		expectedTable map[string]string
	}{
		{
			name:          "empty forceIndexes",
			forceIndexes:  "",
			expectedTable: make(map[string]string),
		},
		{
			name:         "one forceIndexes",
			forceIndexes: "auth@1",
			expectedTable: map[string]string{
				"auth": "auth@1",
			},
		},
		{
			name:         "multiple forceIndexes",
			forceIndexes: "auth@1,clients@2,validators@3",
			expectedTable: map[string]string{
				"auth":       "auth@1",
				"clients":    "clients@2",
				"validators": "validators@3",
			},
		},
	}

	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			overrides := getOverrideTableMap(test.forceIndexes)
			require.Equal(t, test.expectedTable, overrides)
		})
	}
}

func Test_ApplyOverride(t *testing.T) {
	tt := []struct {
		name      string
		base      []string
		overrides map[string]string
		expected  []string
	}{
		{
			name:      "empty overrides",
			base:      []string{"a"},
			overrides: make(map[string]string),
			expected:  []string{"a"},
		},
		{
			name:      "one override",
			base:      []string{"a"},
			overrides: map[string]string{"a": "b"},
			expected:  []string{"b"},
		},
		{
			name:      "one non matching override",
			base:      []string{"a"},
			overrides: map[string]string{"x": "b"},
			expected:  []string{"a"},
		},
		{
			name:      "nil overrides",
			base:      []string{"a"},
			overrides: nil,
			expected:  []string{"a"},
		},
		{
			name:      "multiple overrides",
			base:      []string{"a", "b", "c"},
			overrides: map[string]string{"a": "1", "c": "3"},
			expected:  []string{"1", "b", "3"},
		},
	}

	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, applyOverride(test.base, test.overrides))
		})
	}
}
