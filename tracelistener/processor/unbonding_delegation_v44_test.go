//go:build sdk_v44

package processor

import (
	"github.com/emerishq/tracelistener/tracelistener"
	"github.com/emerishq/tracelistener/tracelistener/processor/datamarshaler"
)

// This file contains some unbonding delegations test cases which are v44-specific.

func versionSpecificUnbondingDelegationsOwnsKeyTests() []unbondingDelegationsOwnsKeyTest {
	return []unbondingDelegationsOwnsKeyTest{
		{
			"Unbonding delegations by validator key is recognized",
			datamarshaler.UnbondingDelegationByValidatorKey,
			"key",
			false,
		},
	}
}

func versionSpecificUnbondingDelegationsProcessTests() []unbondingDelegationsProcessTest {
	return []unbondingDelegationsProcessTest{
		{
			"Delete unbonding delegation operation - key prefix is not the one which index by validator address",
			datamarshaler.TestUnbondingDelegation{
				Delegator: "cosmos1xrnner9s783446yz3hhshpr5fpz6wzcwkvwv5j",
				Validator: "cosmosvaloper19xawgvgn887e9gef5vkzkemwh33mtgwa6haa7s",
			},
			tracelistener.TraceOperation{
				Operation:   string(tracelistener.DeleteOp),
				Key:         []byte("QXRkbFY4cUQ2bzZKMnNoc2o5YWNwSSs5T3BkL2U1dVRxWklpN05LNWkzeTk="),
				Value:       []byte{},
				BlockHeight: 0,
			},
			false,
			0,
		},
		{
			"Delete unbonding delegation operation - no error",
			datamarshaler.TestUnbondingDelegation{
				Delegator: "delegator",
				Validator: "validator",
			},
			tracelistener.TraceOperation{
				Operation: string(tracelistener.DeleteOp),
				Key: []byte{
					0x33, // prefix
					9, 118, 97, 108, 105, 100, 97, 116, 111, 114, 9, 100, 101, 108, 101, 103, 97, 116, 111, 114,
				},
				Value:       []byte{},
				BlockHeight: 0,
			},
			false,
			1,
		},
	}
}
