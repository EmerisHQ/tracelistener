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
		{
			testName: "non ascii",
			name:     "ΩOmega",
			wantErr:  false,
		},
		{
			testName: "DBName.TableName",
			name:     "tracelistener.validators",
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			err := validateName(tt.name)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func Test_validateIndexes(t *testing.T) {
	apple_index := Index{Name: "apple_index", Columns: []string{"apple"}}
	no_name_index := Index{Name: "", Columns: []string{"apple"}}
	apple_pear_index := Index{Name: "apple_pear_index", Columns: []string{"apple", "pear"}}

	testCases := []struct {
		testName string
		t        TableConfig
		names    map[string]bool
		wantErr  bool
	}{
		{
			testName: "happy path",
			t:        TableConfig{Name: "table1", Indexes: []Index{apple_index}},
			names:    map[string]bool{"apple": true},
			wantErr:  false,
		},
		{
			testName: "no index name",
			t:        TableConfig{Name: "table2", Indexes: []Index{no_name_index}},
			names:    map[string]bool{"apple": true},
			wantErr:  true,
		},
		{
			testName: "no matching column",
			t:        TableConfig{Name: "table3", Indexes: []Index{apple_pear_index}},
			names:    map[string]bool{"apple": true},
			wantErr:  true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			err := validateIndexes(tc.t, tc.names)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
