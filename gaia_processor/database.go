package gaia_processor

const (
	insertBalance = `
INSERT INTO tracelistener.balances 
	(address, amount, denom, height) 
VALUES 
	(:address, :amount, :denom, :height) 
ON CONFLICT
	(address, denom)
DO UPDATE SET 
	address=EXCLUDED.address,
	denom=EXCLUDED.denom,
	amount=EXCLUDED.amount,
	height=EXCLUDED.height;
`

	createBalancesTable = `
CREATE TABLE IF NOT EXISTS tracelistener.balances (
	id serial unique primary key,
	address text not null,
	amount integer not null,
	denom text not null,
	height integer not null,
	unique(address, denom)
)
`

	createConnectionsTable = `
CREATE TABLE IF NOT EXISTS tracelistener.connections (
	id serial unique primary key,
	connection_id text not null,
	client_id text not null,
	state text not null,
	counter_connection_id text not null,
	counter_client_id text not null,
	unique(connection_id, client_id)
)
`

	insertConnection = `
INSERT INTO tracelistener.connections 
	(connection_id, client_id, state, counter_connection_id, counter_client_id) 
VALUES 
	(:connection_id, :client_id, :state, :counter_connection_id, :counter_client_id) 
ON CONFLICT
	(connection_id, client_id)
DO UPDATE SET
	state=EXCLUDED.state,
	counter_connection_id=EXCLUDED.counter_connection_id,
	counter_client_id=EXCLUDED.counter_client_id
`
)
