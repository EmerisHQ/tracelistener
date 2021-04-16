package gaia_processor

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
	amount integer not null,
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
	unique(pool_id)
)
`
	insertPool = `
UPSERT INTO tracelistener.liquidity_pools
	(chain_name, pool_id, type_id, reserve_coin_denoms, reserve_account_address, pool_coin_denom)
VALUES
	(:chain_name, :pool_id, :type_id, :reserve_coin_denoms, :reserve_account_address, :pool_coin_denom)
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
	unique(msg_index)
)`

	insertSwap = `
UPSERT INTO tracelistener.liquidity_swaps
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
`

	// Account delegations-related queries
	createDelegationsTable = `
CREATE TABLE IF NOT EXISTS tracelistener.delegations (
	id serial unique primary key,
	delegator_address text not null,
	validator_address text not null,
	amount string not null,
	unique(delegator_address, validator_address)
)
`

	insertDelegation = `
INSERT INTO tracelistener.delegations
	(delegator_address, validator_address, amount) 
VALUES 
	(:delegator_address, :validator_address, :amount)  
ON CONFLICT
	(delegator_address, validator_address)
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
)
