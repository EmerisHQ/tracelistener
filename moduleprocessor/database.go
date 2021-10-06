package moduleprocessor

const (

	// Balance-related queries
	insertBalance = `
INSERT INTO tracelistener.balances 
	(chain_name, address, amount, denom, height) 
VALUES 
	(:chain_name, :address, :amount, :denom, :height) 
ON CONFLICT
	(chain_name, address, denom)
DO UPDATE SET 
	chain_name=EXCLUDED.chain_name,
	address=EXCLUDED.address,
	denom=EXCLUDED.denom,
	amount=EXCLUDED.amount,
	height=EXCLUDED.height;
`

	createBalancesTable = `
CREATE TABLE IF NOT EXISTS tracelistener.balances (
	id serial unique primary key,
	chain_name text not null,
	address text not null,
	amount text not null,
	denom text not null,
	height integer not null,
	unique(chain_name, address, denom)
)
`

	// Connection-related queries
	createConnectionsTable = `
CREATE TABLE IF NOT EXISTS tracelistener.connections (
	id serial unique primary key,
	chain_name text not null,
	connection_id text not null,
	client_id text not null,
	state text not null,
	counter_connection_id text not null,
	counter_client_id text not null,
	unique(chain_name, connection_id, client_id)
)
`

	insertConnection = `
INSERT INTO tracelistener.connections 
	(chain_name, connection_id, client_id, state, counter_connection_id, counter_client_id) 
VALUES 
	(:chain_name, :connection_id, :client_id, :state, :counter_connection_id, :counter_client_id) 
ON CONFLICT
	(chain_name, connection_id, client_id)
DO UPDATE SET
	chain_name=EXCLUDED.chain_name,
	state=EXCLUDED.state,
	counter_connection_id=EXCLUDED.counter_connection_id,
	counter_client_id=EXCLUDED.counter_client_id
`

	// Liquidity pool-related queries
	createPoolsTable = `
CREATE TABLE IF NOT EXISTS tracelistener.liquidity_pools (
	id serial unique primary key,
	chain_name text not null,
	pool_id bigint not null,
	type_id bigint not null,
	reserve_coin_denoms text[] not null,
	reserve_account_address text not null,
	pool_coin_denom text not null,
	unique(chain_name, pool_id)
)
`
	insertPool = `
INSERT INTO tracelistener.liquidity_pools
	(chain_name, pool_id, type_id, reserve_coin_denoms, reserve_account_address, pool_coin_denom)
VALUES
	(:chain_name, :pool_id, :type_id, :reserve_coin_denoms, :reserve_account_address, :pool_coin_denom)
ON CONFLICT
	(chain_name, pool_id)
DO UPDATE SET
	chain_name=EXCLUDED.chain_name,
	pool_id=EXCLUDED.pool_id,
	type_id=EXCLUDED.type_id,
	reserve_coin_denoms=EXCLUDED.reserve_coin_denoms,
	reserve_account_address=EXCLUDED.reserve_account_address,
	pool_coin_denom=EXCLUDED.pool_coin_denom
`

	// Liquidity swaps-related queries
	createSwapsTable = `
CREATE TABLE IF NOT EXISTS tracelistener.liquidity_swaps (
	id serial unique primary key,
	chain_name text not null,
	msg_height bigint not null,
	msg_index bigint not null,
	executed bool not null,
	succeded bool not null,
	expiry_height bigint not null,
	exchange_offer_coin text not null,
	remaining_offer_coin_fee text not null,
	reserved_offer_coin_fee text not null,
	pool_coin_denom text not null,
	requester_address text not null,
	pool_id bigint not null,
	offer_coin text not null,
	order_price string not null,
	unique(chain_name, msg_index)
)`

	insertSwap = `
INSERT INTO tracelistener.liquidity_swaps
	(
		chain_name, 
		msg_height,
		msg_index,
		executed,
		succeded,
		expiry_height,
		exchange_offer_coin,
		remaining_offer_coin_fee,
		reserved_offer_coin_fee,
		pool_coin_denom,
		requester_address,
		pool_id,
		offer_coin,
		demand_coin,
		order_price
	)
VALUES
	(
		:chain_name,
		:msg_height,
		:msg_index,
		:executed,
		:succeded,
		:expiry_height,
		:exchange_offer_coin,
		:remaining_offer_coin_fee,
		:reserved_offer_coin_fee,
		:pool_coin_denom,
		:requester_address,
		:pool_id,
		:offer_coin,
		:demand_coin,
		:order_price
	)
ON CONFLICT
	(chain_name, msg_index)
DO UPDATE SET
		chain_name=EXCLUDED.chain_name, 
		msg_height=EXCLUDED.msg_height,
		msg_index=EXCLUDED.msg_index,
		executed=EXCLUDED.executed,
		succeded=EXCLUDED.succeded,
		expiry_height=EXCLUDED.expiry_height,
		exchange_offer_coin=EXCLUDED.exchange_offer_coin,
		remaining_offer_coin_fee=EXCLUDED.remaining_offer_coin_fee,
		reserved_offer_coin_fee=EXCLUDED.reserved_offer_coin_fee,
		pool_coin_denom=EXCLUDED.pool_coin_denom,
		requester_address=EXCLUDED.requester_address,
		pool_id=EXCLUDED.pool_id,
		offer_coin=EXCLUDED.offer_coin,
		demand_coin=EXCLUDED.demand_coin,
		order_price=EXCLUDED.order_price
`

	// Account delegations-related queries
	createDelegationsTable = `
CREATE TABLE IF NOT EXISTS tracelistener.delegations (
	id serial unique primary key,
	chain_name text not null,
	delegator_address text not null,
	validator_address text not null,
	amount string not null,
	unique(chain_name, delegator_address, validator_address)
)
`

	insertDelegation = `
INSERT INTO tracelistener.delegations
	(delegator_address, validator_address, amount, chain_name) 
VALUES 
	(:delegator_address, :validator_address, :amount, :chain_name)  
ON CONFLICT
	(chain_name, delegator_address, validator_address)
DO UPDATE SET
	amount=EXCLUDED.amount
`

	deleteDelegation = `
DELETE FROM tracelistener.delegations
WHERE
	delegator_address=:delegator_address
AND
	validator_address=:validator_address
`

	// Denom traces-related queries
	createDenomTracesTable = `
CREATE TABLE IF NOT EXISTS tracelistener.denom_traces (
	id serial unique primary key,
	chain_name text not null,
	path text not null,
	base_denom text not null,
	hash text not null,
	unique(chain_name, hash)
)
`

	insertDenomTrace = `
INSERT INTO tracelistener.denom_traces
	(path, base_denom, hash, chain_name) 
VALUES 
	(:path, :base_denom, :hash, :chain_name)
ON CONFLICT
	(chain_name, hash)
DO UPDATE SET
	base_denom=EXCLUDED.base_denom,
	hash=EXCLUDED.hash,
	path=EXCLUDED.path
`

	// IBC channels-related queries
	createChannelsTable = `
CREATE TABLE IF NOT EXISTS tracelistener.channels (
	id serial unique primary key,
	chain_name text not null,
	channel_id text not null,
	counter_channel_id text not null,
	port text not null,
	state integer not null,
	hops text[] not null,
	unique(chain_name, channel_id, port)
)
`

	insertChannel = `
INSERT INTO tracelistener.channels
	(channel_id, counter_channel_id, port, state, hops, chain_name) 
VALUES 
	(:channel_id, :counter_channel_id, :port, :state, :hops, :chain_name)
ON CONFLICT
	(chain_name, channel_id, port)
DO UPDATE SET
	state=EXCLUDED.state,
	counter_channel_id=EXCLUDED.counter_channel_id,
	hops=EXCLUDED.hops,
	port=EXCLUDED.port,
	channel_id=EXCLUDED.channel_id
`

	// Auth-related queries
	createAuthTable = `
CREATE TABLE IF NOT EXISTS tracelistener.auth (
	id serial unique primary key,
	chain_name text not null,
	address text not null,
	sequence_number numeric not null,
	account_number numeric not null,
	unique(chain_name, address, account_number)
)
`

	insertAuth = `
INSERT INTO tracelistener.auth 
	(chain_name, address, sequence_number, account_number) 
VALUES 
	(:chain_name, :address, :sequence_number, :account_number) 
ON CONFLICT
	(chain_name, address, account_number)
DO UPDATE SET 
	chain_name=EXCLUDED.chain_name,
	address=EXCLUDED.address,
	sequence_number=EXCLUDED.sequence_number,
	account_number=EXCLUDED.account_number
`

	createClientsTable = `
CREATE TABLE IF NOT EXISTS tracelistener.clients (
	id serial unique primary key,
	chain_name text not null,
	chain_id text not null,
	client_id text not null,
	latest_height numeric not null,
	trusting_period numeric not null,
	unique(chain_name, chain_id, client_id)
)
`

	insertClient = `
INSERT INTO tracelistener.clients
	(chain_name, chain_id, client_id, latest_height, trusting_period) 
VALUES 
	(:chain_name, :chain_id, :client_id, :latest_height, :trusting_period)
ON CONFLICT
	(chain_name, chain_id, client_id)
DO UPDATE SET
	chain_id=EXCLUDED.chain_id,
	client_id=EXCLUDED.client_id,
	latest_height=EXCLUDED.latest_height,
	trusting_period=EXCLUDED.trusting_period
`
)
