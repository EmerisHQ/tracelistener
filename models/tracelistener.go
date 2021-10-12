package models

import (
	"encoding/json"
	"time"
)

// TracelistenerDatabaseRow contains a list of all the fields each database row must contain in order to be
// inserted correctly.
type TracelistenerDatabaseRow struct {
	ChainName string `db:"chain_name" json:"chain_name"`
	ID        uint64 `db:"id" json:"-"`
}

// DatabaseEntrier is implemented by each object that wants to be inserted in a database.
// It is usually used in conjunction to TracelistenerDatabaseRow.
type DatabaseEntrier interface {
	// WithChainName sets the ChainName field of the TracelistenerDatabaseRow struct.
	WithChainName(cn string) DatabaseEntrier
}

// BalanceRow represents a balance row inserted into the database.
type BalanceRow struct {
	TracelistenerDatabaseRow

	Address     string `db:"address" json:"address"`
	Amount      string `db:"amount" json:"amount"`
	Denom       string `db:"denom" json:"denom"`
	BlockHeight uint64 `db:"height" json:"block_height"`
}

// WithChainName implements the DatabaseEntrier interface.
func (b BalanceRow) WithChainName(cn string) DatabaseEntrier {
	b.ChainName = cn
	return b
}

// DelegationRow represents a delegation row inserted into the database.
type DelegationRow struct {
	TracelistenerDatabaseRow

	Delegator   string `db:"delegator_address" json:"delegator"`
	Validator   string `db:"validator_address" json:"validator"`
	Amount      string `db:"amount" json:"amount"`
	BlockHeight uint64 `db:"height" json:"block_height"`
}

// WithChainName implements the DatabaseEntrier interface.
func (b DelegationRow) WithChainName(cn string) DatabaseEntrier {
	b.ChainName = cn
	return b
}

// IBCChannelRow represents an IBC channel row inserted into the database.
type IBCChannelRow struct {
	TracelistenerDatabaseRow

	ChannelID        string   `db:"channel_id" json:"channel_id"`
	CounterChannelID string   `db:"counter_channel_id" json:"counter_channel_id"`
	Hops             []string `db:"hops" json:"hops"`
	Port             string   `db:"port" json:"port"`
	State            int32    `db:"state" json:"state"`
}

// WithChainName implements the DatabaseEntrier interface.
func (c IBCChannelRow) WithChainName(cn string) DatabaseEntrier {
	c.ChainName = cn
	return c
}

// IBCConnectionRow represents an IBC connection row inserted into the database.
type IBCConnectionRow struct {
	TracelistenerDatabaseRow

	ConnectionID        string `db:"connection_id" json:"connection_id"`
	ClientID            string `db:"client_id" json:"client_id"`
	State               string `db:"state" json:"state"`
	CounterConnectionID string `db:"counter_connection_id" json:"counter_connection_id"`
	CounterClientID     string `db:"counter_client_id" json:"counter_client_id"`
}

// WithChainName implements the DatabaseEntrier interface.
func (c IBCConnectionRow) WithChainName(cn string) DatabaseEntrier {
	c.ChainName = cn
	return c
}

// IBCDenomTraceRow represents an IBC denom trace row inserted into the database.
type IBCDenomTraceRow struct {
	TracelistenerDatabaseRow

	Path      string `json:"path" db:"path"`
	BaseDenom string `json:"base_denom" db:"base_denom"`
	Hash      string `json:"hash" db:"hash"`
}

// WithChainName implements the DatabaseEntrier interface.
func (c IBCDenomTraceRow) WithChainName(cn string) DatabaseEntrier {
	c.ChainName = cn
	return c
}

// PoolRow represents a liquidity pool data inserted into the database.
type PoolRow struct {
	TracelistenerDatabaseRow

	PoolID                uint64   `db:"pool_id"`
	TypeID                uint32   `db:"type_id"`
	ReserveCoinDenoms     []string `db:"reserve_coin_denoms"`
	ReserveAccountAddress string   `db:"reserve_account_address"`
	PoolCoinDenom         string   `db:"pool_coin_denom"`
}

// WithChainName implements the DatabaseEntrier interface.
func (bwp PoolRow) WithChainName(cn string) DatabaseEntrier {
	bwp.ChainName = cn
	return bwp
}

// SwapRow represents a liquidity swap action, inserted into the database.
type SwapRow struct {
	TracelistenerDatabaseRow

	MsgHeight            int64  `db:"msg_height"`
	MsgIndex             uint64 `db:"msg_index"`
	Executed             bool   `db:"executed"`
	Succeeded            bool   `db:"succeeded"`
	ExpiryHeight         int64  `db:"expiry_height"`
	ExchangedOfferCoin   string `db:"exchanged_offer_coin"`
	RemainingOfferCoin   string `db:"remaining_offer_coin"`
	ReservedOfferCoinFee string `db:"reserved_offer_coin_fee"`
	PoolCoinDenom        string `db:"pool_coin_denom"`
	RequesterAddress     string `db:"requester_address"`
	PoolID               uint64 `db:"pool_id"`
	OfferCoin            string `db:"offer_coin"`
	OrderPrice           string `db:"order_price"`
}

// WithChainName implements the DatabaseEntrier interface.
func (bwp SwapRow) WithChainName(cn string) DatabaseEntrier {
	bwp.ChainName = cn
	return bwp
}

// AuthRow represents an account auth row inserted into the database.
type AuthRow struct {
	TracelistenerDatabaseRow

	Address        string `db:"address" json:"address"`
	SequenceNumber uint64 `db:"sequence_number" json:"sequence_number"`
	AccountNumber  uint64 `db:"account_number" json:"account_number"`
}

// WithChainName implements the DatabaseEntrier interface.
func (b AuthRow) WithChainName(cn string) DatabaseEntrier {
	b.ChainName = cn
	return b
}

// BlockTimeRow represents a row containing the last time a chain received a block.
type BlockTimeRow struct {
	TracelistenerDatabaseRow

	BlockTime time.Time `db:"block_time"`
}

// IBCClientStateRow represents the state of client as a row inserted into the database.
type IBCClientStateRow struct {
	TracelistenerDatabaseRow

	ChainID        string `db:"chain_id" json:"chain_id"`
	ClientID       string `db:"client_id" json:"client_id"`
	LatestHeight   uint64 `db:"latest_height" json:"latest_height"`
	TrustingPeriod int64  `db:"trusting_period" json:"trusting_period"`
}

// WithChainName implements the DatabaseEntrier interface.
func (b IBCClientStateRow) WithChainName(cn string) DatabaseEntrier {
	b.ChainName = cn
	return b
}

type UnbondingDelegationRow struct {
	TracelistenerDatabaseRow

	Delegator string                     `db:"delegator_address" json:"delegator"`
	Validator string                     `db:"validator_address" json:"validator"`
	Entries   UnbondingDelegationEntries `db:"entries" json:"entries"`
}

type UnbondingDelegationEntry struct {
	Balance        string `db:"balance" json:"balance"`
	InitialBalance string `db:"initial_balance" json:"initial_balance"`
	CreationHeight int64  `db:"creation_height" json:"creation_height"`
	CompletionTime string `db:"completion_time" json:"completion_time"`
}

type UnbondingDelegationEntries []UnbondingDelegationEntry

// WithChainName implements the DatabaseEntrier interface.
func (b UnbondingDelegationRow) WithChainName(cn string) DatabaseEntrier {
	b.ChainName = cn
	return b
}

func (entries *UnbondingDelegationEntries) Scan(src interface{}) error {
	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return nil // or return some error
	}
	return json.Unmarshal(data, entries)
}

// ValidatorRow represents the state of a validator as a row inserted into the database.
type ValidatorRow struct {
	TracelistenerDatabaseRow

	OperatorAddress      string `db:"operator_address" json:"operator_address"`
	ConsensusPubKeyType  string `db:"consensus_pubkey_type" json:"consensus_pubkey_type"`
	ConsensusPubKeyValue []byte `db:"consensus_pubkey_value" json:"consensus_pubkey_value"`
	Jailed               bool   `db:"jailed" json:"jailed"`
	Status               int32  `db:"status" json:"status"`
	Tokens               string `db:"tokens" json:"tokens"`
	DelegatorShares      string `db:"delegator_shares" json:"delegator_shares"`
	Moniker              string `db:"moniker" json:"moniker,omitempty"`
	Identity             string `db:"identity" json:"identity,omitempty"`
	Website              string `db:"website" json:"website,omitempty"`
	SecurityContact      string `db:"security_contact" json:"security_contact,omitempty"`
	Details              string `db:"details" json:"details,omitempty"`
	UnbondingHeight      int64  `db:"unbonding_height" json:"unbonding_height"`
	UnbondingTime        string `db:"unbonding_time" json:"unbonding_time"`
	CommissionRate       string `db:"commission_rate" json:"commission_rate"`
	MaxRate              string `db:"max_rate" json:"max_rate"`
	MaxChangeRate        string `db:"max_change_rate" json:"max_change_rate"`
	UpdateTime           string `db:"update_time" json:"update_time"`
	MinSelfDelegation    string `db:"min_self_delegation" json:"min_self_delegation"`
}

// WithChainName implements the DatabaseEntrier interface.
func (b ValidatorRow) WithChainName(cn string) DatabaseEntrier {
	b.ChainName = cn
	return b
}
