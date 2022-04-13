package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_validateName(t *testing.T) {
	tests := []struct {
		testName string
		name     string
		wantErr  bool
	}{
		{
			testName: "empty",
			name:     "",
			wantErr:  true,
		},
		{
			testName: "starts with number",
			name:     "1name",
			wantErr:  true,
		},
		{
			testName: "starts with underscore",
			name:     "_1name",
			wantErr:  false,
		},
		{
			testName: "starts with letter",
			name:     "name42",
			wantErr:  false,
		},
		{
			testName: "very long name",
			name:     "namenamenamenamenamenamenamenamenamenamenamenamenamenamenamenamenamenamenamenamenamenamenamenamenamenamenamenamenamenamenamenamename",
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			err := validateName(tt.name)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
