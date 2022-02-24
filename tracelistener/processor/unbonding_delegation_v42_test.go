//go:build sdk_v42

package processor

import (
	"github.com/allinbits/tracelistener/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener/processor/datamarshaler"
)

// This file contains some unbonding delegations test cases which are v42-specific.

func versionSpecificUnbondingDelegationsOwnsKeyTests() []unbondingDelegationsOwnsKeyTest {
	return nil // no-op here
}

func versionSpecificUnbondingDelegationsProcessTests() []unbondingDelegationsProcessTest {
	return []unbondingDelegationsProcessTest{
		{
			"Delete unbonding delegation operation - no error",
			datamarshaler.TestUnbondingDelegation{
				Delegator: "cosmos1xrnner9s783446yz3hhshpr5fpz6wzcwkvwv5j",
				Validator: "cosmosvaloper19xawgvgn887e9gef5vkzkemwh33mtgwa6haa7s",
			},
			tracelistener.TraceOperation{
				Operation:   string(tracelistener.DeleteOp),
				Key:         []byte("QXRkbFY4cUQ2bzZKMnNoc2o5YWNwSSs5T3BkL2U1dVRxWklpN05LNWkzeTk="),
				Value:       []byte("Ci1jb3Ntb3MxeHJubmVyOXM3ODM0NDZ5ejNoaHNocHI1ZnB6Nnd6Y3drdnd2NWoSNGNvc21vc3ZhbG9wZXIxOXhhd2d2Z244ODdlOWdlZjV2a3prZW13aDMzbXRnd2E2aGFhN3MaHAiYIBILCICSuMOY/v///wEaBDEwMDAiBDExMDA="),
				BlockHeight: 0,
			},
			false,
			1,
		},
	}
}
