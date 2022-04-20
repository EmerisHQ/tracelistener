package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_getStructName(t *testing.T) {
	tests := []struct {
		name       string
		tableName  string
		structName string
	}{
		{
			name:       "single word",
			tableName:  "balances",
			structName: "BalancesTable",
		},
		{
			name:       "two words",
			tableName:  "account_transactions",
			structName: "AccountTransactionsTable",
		},
		{
			name:       "initial underscore",
			tableName:  "_private",
			structName: "PrivateTable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			structName := getStructName(tt.tableName)
			require.Equal(t, tt.structName, structName)
		})
	}
}
