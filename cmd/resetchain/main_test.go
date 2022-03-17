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
			},
		},
		{
			name:          "negative chunk size",
			expectedError: true,
			flags: Flags{
				db:        "postgres://localhost",
				chain:     "cosmos-hub",
				chunkSize: -2,
			},
		},
		{
			name:          "zero chunk size",
			expectedError: true,
			flags: Flags{
				db:        "postgres://localhost",
				chain:     "cosmos-hub",
				chunkSize: 0,
			},
		},
		{
			name:          "empty connection string",
			expectedError: true,
			flags: Flags{
				db:        "",
				chain:     "cosmos-hub",
				chunkSize: 2,
			},
		},
		{
			name:          "empty chain name",
			expectedError: true,
			flags: Flags{
				db:        "postgres://localhost",
				chain:     "",
				chunkSize: 2,
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
