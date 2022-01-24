//go:build sdk_v44

package datamarshaler

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	models "github.com/allinbits/demeris-backend-models/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingTypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	gaia "github.com/cosmos/gaia/v6/app"
	transferTypes "github.com/cosmos/ibc-go/v2/modules/apps/transfer/types"
	ibcConnectionTypes "github.com/cosmos/ibc-go/v2/modules/core/03-connection/types"
	channelTypes "github.com/cosmos/ibc-go/v2/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v2/modules/core/24-host"
	"github.com/cosmos/ibc-go/v2/modules/core/exported"
	tmIBCTypes "github.com/cosmos/ibc-go/v2/modules/light-clients/07-tendermint/types"
)

var (
	cdc     codec.Codec = nil
	cdcOnce sync.Once
)

const (
	transferMsgType = "transfer"
)

func initCodec() {
	c := gaia.MakeEncodingConfig().Marshaler
	cdc = c
}

func getCodec() codec.Codec {
	cdcOnce.Do(initCodec)

	return cdc
}

func (d DataMarshaler) Bank(data tracelistener.TraceOperation) (models.BalanceRow, error) {
	// How's a length-prefixed data.Key is made you ask?
	// 0x02<length prefix><address bytes>
	//
	// AddressFromBalancesStore requires the key data without the store prefix
	// so we simply reslice data.Key to get rid of it.
	//
	// If data.Operation == "delete", the trace that's been observed has a different data.Key:
	// 0x02<length prefix><address bytes><denom>
	//
	// This different schema is used when the balance associated to <denom> is being set to zero.
	// So, to obtain this denom one must subslice rawAddress to the length of <address bytes> + 1
	// to bypass the length prefix byte.
	rawAddress := data.Key[1:]
	addrBytes, err := bankTypes.AddressFromBalancesStore(rawAddress)
	if err != nil {
		return models.BalanceRow{}, fmt.Errorf("cannot parse address from balance store key, %w", err)
	}

	hAddr := hex.EncodeToString(addrBytes)

	coins := sdk.Coin{
		Amount: sdk.NewInt(0),
	}

	if err := getCodec().Unmarshal(data.Value, &coins); err != nil {
		return models.BalanceRow{}, err
	}

	// Since SDK 0.44.x x/bank now deletes keys from store when the balance is 0
	// (picture someone who sends all their balance to another address).
	// To work around this issue, we don't return when coin is invalid when data.Operation is "delete",
	// and we set balance == 0 instead.
	if !coins.IsValid() {
		if data.Operation == tracelistener.DeleteOp.String() {
			// rawAddress still contains the lenght prefix, so we have to jump it by
			// reading 1 byte after len(addrBytes)
			denom := rawAddress[len(addrBytes)+1:]
			coins.Denom = string(denom)
		} else {
			return models.BalanceRow{}, nil
		}
	}

	d.l.Debugw("new bank store write",
		"operation", data.Operation,
		"address", hAddr,
		"new_balance", coins.String(),
		"height", data.BlockHeight,
		"txHash", data.TxHash,
	)

	return models.BalanceRow{
		Address:     hAddr,
		Amount:      coins.String(),
		Denom:       coins.Denom,
		BlockHeight: data.BlockHeight,
	}, nil
}

func (d DataMarshaler) Auth(data tracelistener.TraceOperation) (models.AuthRow, error) {
	d.l.Debugw("auth processor entered", "key", string(data.Key), "value", string(data.Value))

	if len(data.Key) > address.MaxAddrLen+1 {
		d.l.Debugw("auth got key that isn't supposed to")
		// key len must be len(account bytes) + 1
		return models.AuthRow{}, nil
	}

	var acc authTypes.AccountI

	if err := getCodec().UnmarshalInterface(data.Value, &acc); err != nil {
		// HACK: since slashing and auth use the same prefix for two different things,
		// let's ignore "no concrete type registered for type URL *" errors.
		// This is ugly, but frankly this is the only way to do it.
		// Frojdi please bless us with the new SDK ASAP.

		if strings.HasPrefix(err.Error(), "no concrete type registered for type URL") {
			d.l.Debugw("exiting because value isnt accountI")
			return models.AuthRow{}, nil
		}

		return models.AuthRow{}, err
	}

	if _, ok := acc.(*authTypes.ModuleAccount); ok {
		// ignore moduleaccounts
		d.l.Debugw("exiting because moduleaccount")
		return models.AuthRow{}, nil
	}

	baseAcc, ok := acc.(*authTypes.BaseAccount)
	if !ok {
		return models.AuthRow{}, fmt.Errorf("cannot cast account to BaseAccount, type %T, account object type %T", baseAcc, acc)
	}

	if err := baseAcc.Validate(); err != nil {
		d.l.Debugw("found invalid base account", "account", baseAcc, "error", err)
		return models.AuthRow{}, fmt.Errorf("non compliant auth account, %w", err)
	}

	_, bz, err := bech32.DecodeAndConvert(baseAcc.Address)
	if err != nil {
		return models.AuthRow{}, fmt.Errorf("cannot parse %s as bech32, %w", baseAcc.Address, err)
	}

	hAddr := hex.EncodeToString(bz)
	d.l.Debugw("new auth store write",
		"operation", data.Operation,
		"address", hAddr,
		"sequence_number", acc.GetSequence(),
		"account_number", acc.GetAccountNumber(),
		"height", data.BlockHeight,
		"txHash", data.TxHash,
	)

	return models.AuthRow{
		Address:        hAddr,
		SequenceNumber: acc.GetSequence(),
		AccountNumber:  acc.GetAccountNumber(),
	}, nil
}

func (d DataMarshaler) Delegations(data tracelistener.TraceOperation) (models.DelegationRow, error) {
	if data.Operation == tracelistener.DeleteOp.String() {
		if len(data.Key) < 41 { // 20 bytes by address, 1 prefix = 2*20 + 1
			return models.DelegationRow{}, nil // found probably liquidity stuff being deleted
		}

		delegatorAddr := hex.EncodeToString(data.Key[1:21])
		validatorAddr := hex.EncodeToString(data.Key[21:41])
		d.l.Debugw("new delegation delete", "delegatorAddr", delegatorAddr, "validatorAddr", validatorAddr)
		return models.DelegationRow{
			Delegator: delegatorAddr,
			Validator: validatorAddr,
		}, nil
	}

	delegation := stakingTypes.Delegation{}

	if err := getCodec().Unmarshal(data.Value, &delegation); err != nil {
		return models.DelegationRow{}, err
	}

	delegator, err := b32Hex(delegation.DelegatorAddress)
	if err != nil {
		return models.DelegationRow{}, fmt.Errorf("cannot convert delegator address from bech32 to hex, %w", err)
	}

	validator, err := b32Hex(delegation.ValidatorAddress)
	if err != nil {
		return models.DelegationRow{}, fmt.Errorf("cannot convert validator address from bech32 to hex, %w", err)
	}

	d.l.Debugw("new delegation write",
		"operation", data.Operation,
		"delegator", delegator,
		"validator", "validator",
		"amount", delegation.Shares.String(),
		"height", data.BlockHeight,
		"txHash", data.TxHash,
	)

	return models.DelegationRow{
		Delegator:   delegator,
		Validator:   validator,
		Amount:      delegation.Shares.String(),
		BlockHeight: data.BlockHeight,
	}, nil
}

func (d DataMarshaler) IBCChannels(data tracelistener.TraceOperation) (models.IBCChannelRow, error) {
	d.l.Debugw("ibc channel key", "key", string(data.Key), "raw value", string(data.Value))
	var result channelTypes.Channel
	if err := getCodec().Unmarshal(data.Value, &result); err != nil {
		return models.IBCChannelRow{}, err
	}

	if err := result.ValidateBasic(); err != nil {
		d.l.Debugw("found non-compliant channel", "channel", result, "error", err)
		return models.IBCChannelRow{}, fmt.Errorf("cannot validate ibc channel, %w", err)
	}

	if result.Ordering != channelTypes.UNORDERED {
		return models.IBCChannelRow{}, nil
	}

	d.l.Debugw("ibc channel data", "result", result)

	portID, channelID, err := host.ParseChannelPath(string(data.Key))
	if err != nil {
		return models.IBCChannelRow{}, err
	}

	return models.IBCChannelRow{
		ChannelID:        channelID,
		CounterChannelID: result.Counterparty.ChannelId,
		Hops:             result.GetConnectionHops(),
		Port:             portID,
		State:            int32(result.State),
	}, nil
}

func (d DataMarshaler) IBCClients(data tracelistener.TraceOperation) (models.IBCClientStateRow, error) {
	d.l.Debugw("ibc client key", "key", string(data.Key), "raw value", string(data.Value))
	var result exported.ClientState
	var dest *tmIBCTypes.ClientState
	if err := getCodec().UnmarshalInterface(data.Value, &result); err != nil {
		return models.IBCClientStateRow{}, err
	}

	if res, ok := result.(*tmIBCTypes.ClientState); !ok {
		return models.IBCClientStateRow{}, nil
	} else {
		dest = res
	}

	if err := result.Validate(); err != nil {
		d.l.Debugw("found non-compliant ibc connection", "connection", dest, "error", err)
		return models.IBCClientStateRow{}, fmt.Errorf("cannot validate ibc connection, %w", err)
	}

	keySplit := strings.Split(string(data.Key), "/")
	clientID := keySplit[1]

	return models.IBCClientStateRow{
		ChainID:        dest.ChainId,
		ClientID:       clientID,
		LatestHeight:   dest.LatestHeight.RevisionHeight,
		TrustingPeriod: int64(dest.TrustingPeriod),
	}, nil
}

func (d DataMarshaler) IBCConnections(data tracelistener.TraceOperation) (models.IBCConnectionRow, error) {
	keyFields := strings.FieldsFunc(string(data.Key), func(r rune) bool {
		return r == '/'
	})

	d.l.Debugw("ibc store key", "fields", keyFields, "raw key", string(data.Key))

	// IBC keys are mostly strings
	if len(keyFields) == 2 {
		if keyFields[0] == IBCConnectionsKey { // this is a ConnectionEnd
			ce := ibcConnectionTypes.ConnectionEnd{}
			if err := getCodec().Unmarshal(data.Value, &ce); err != nil {
				return models.IBCConnectionRow{}, fmt.Errorf("cannot unmarshal connection end, %w", err)
			}

			if err := ce.ValidateBasic(); err != nil {
				d.l.Debugw("found non-compliant connection end", "connection end", ce, "error", err)
				return models.IBCConnectionRow{}, fmt.Errorf("connection end validation failed, %w", err)
			}

			d.l.Debugw("connection end", "data", ce)
			return models.IBCConnectionRow{
				ConnectionID:        keyFields[1],
				ClientID:            ce.ClientId,
				State:               ce.State.String(),
				CounterConnectionID: ce.Counterparty.ConnectionId,
				CounterClientID:     ce.Counterparty.ClientId,
			}, nil
		}
	}

	return models.IBCConnectionRow{}, nil
}

func (d DataMarshaler) IBCDenomTraces(data tracelistener.TraceOperation) (models.IBCDenomTraceRow, error) {
	d.l.Debugw("beginning denom trace processor", "key", string(data.Key), "value", string(data.Value))

	dt := transferTypes.DenomTrace{}
	if err := getCodec().Unmarshal(data.Value, &dt); err != nil {
		return models.IBCDenomTraceRow{}, err
	}

	if err := dt.Validate(); err != nil {
		d.l.Debugw("found a denom trace that isn't ICS20 compliant", "denom trace", dt, "error", err)
		return models.IBCDenomTraceRow{}, fmt.Errorf("denom trace validation failed, %w", err)
	}

	if dt.BaseDenom == "" {
		d.l.Debugw("ignoring since it's not a denom trace")
		return models.IBCDenomTraceRow{}, nil
	}

	hash := hex.EncodeToString(dt.Hash())

	newObj := models.IBCDenomTraceRow{
		Path:      dt.Path,
		BaseDenom: dt.BaseDenom,
		Hash:      hash,
	}

	d.l.Debugw("denom trace unmarshaled", "object", newObj)

	return newObj, nil
}

func (d DataMarshaler) UnbondingDelegations(data tracelistener.TraceOperation) (models.UnbondingDelegationRow, error) {
	if data.Operation == tracelistener.DeleteOp.String() {
		if len(data.Key) < 41 { // 20 bytes by address, 1 prefix = 2*20 + 1
			return models.UnbondingDelegationRow{}, nil // found probably liquidity stuff being deleted
		}
		delegatorAddr := hex.EncodeToString(data.Key[1:21])
		validatorAddr := hex.EncodeToString(data.Key[21:41])
		d.l.Debugw("new unbonding_delegation delete", "delegatorAddr", delegatorAddr, "validatorAddr", validatorAddr)

		return models.UnbondingDelegationRow{
			Delegator: delegatorAddr,
			Validator: validatorAddr,
		}, nil
	}

	unbondingDelegation := stakingTypes.UnbondingDelegation{}

	if err := getCodec().Unmarshal(data.Value, &unbondingDelegation); err != nil {
		return models.UnbondingDelegationRow{}, err
	}

	delegator, err := b32Hex(unbondingDelegation.DelegatorAddress)
	if err != nil {
		return models.UnbondingDelegationRow{}, fmt.Errorf("cannot convert delegator address from bech32 to hex, %w", err)
	}

	validator, err := b32Hex(unbondingDelegation.ValidatorAddress)
	if err != nil {
		return models.UnbondingDelegationRow{}, fmt.Errorf("cannot convert validator address from bech32 to hex, %w", err)
	}

	entries, err := json.Marshal(unbondingDelegation.Entries)

	if err != nil {
		return models.UnbondingDelegationRow{}, fmt.Errorf("cannot convert unbonding delegation entries to string")
	}
	d.l.Debugw("new unbondingDelegation write",
		"operation", data.Operation,
		"delegator", delegator,
		"validator", validator,
		"entries", string(entries),
		"height", data.BlockHeight,
		"txHash", data.TxHash,
	)

	var entriesStore models.UnbondingDelegationEntries

	err = json.Unmarshal(entries, &entriesStore)

	if err != nil {
		return models.UnbondingDelegationRow{}, fmt.Errorf("unable to unmarshal unbonding delegation entries")
	}

	return models.UnbondingDelegationRow{
		Delegator: delegator,
		Validator: validator,
		Entries:   entriesStore,
	}, err
}

func (d DataMarshaler) Validators(data tracelistener.TraceOperation) (models.ValidatorRow, error) {
	if data.Operation == tracelistener.DeleteOp.String() {
		if len(data.Key) < 21 {
			return models.ValidatorRow{}, nil
		}

		operatorAddress := hex.EncodeToString(data.Key[1:21])
		d.l.Debugw("new validator delete", "operator address", operatorAddress)

		return models.ValidatorRow{
			OperatorAddress: operatorAddress,
		}, nil

	}

	v := stakingTypes.Validator{}

	if err := getCodec().Unmarshal(data.Value, &v); err != nil {
		return models.ValidatorRow{}, err
	}

	val := string(v.ConsensusPubkey.GetValue())

	k := hex.EncodeToString(data.Key)

	d.l.Debugw("new validator write",
		"operator_address", v.OperatorAddress,
		"height", data.BlockHeight,
		"txHash", data.TxHash,
		"cons pub key type", data.TxHash,
		"cons pub key", val,
		"key", k,
	)

	return models.ValidatorRow{
		OperatorAddress:      v.OperatorAddress,
		ConsensusPubKeyType:  v.ConsensusPubkey.GetTypeUrl(),
		ConsensusPubKeyValue: v.ConsensusPubkey.Value,
		Jailed:               v.Jailed,
		Status:               int32(v.Status),
		Tokens:               v.Tokens.String(),
		DelegatorShares:      v.DelegatorShares.String(),
		Moniker:              v.Description.Moniker,
		Identity:             v.Description.Identity,
		Website:              v.Description.Website,
		SecurityContact:      v.Description.SecurityContact,
		Details:              v.Description.Details,
		UnbondingHeight:      v.UnbondingHeight,
		UnbondingTime:        v.UnbondingTime.String(),
		CommissionRate:       v.Commission.CommissionRates.Rate.String(),
		MaxRate:              v.Commission.CommissionRates.MaxRate.String(),
		MaxChangeRate:        v.Commission.CommissionRates.MaxChangeRate.String(),
		UpdateTime:           v.Commission.UpdateTime.String(),
		MinSelfDelegation:    v.MinSelfDelegation.String(),
	}, nil
}
