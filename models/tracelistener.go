package models

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

	ChannelID string   `db:"channel_id" json:"channel_id"`
	Hops      []string `db:"hops" json:"hops"`
	Port      string   `db:"port" json:"port"`
	State     int32    `db:"state" json:"state"`
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
