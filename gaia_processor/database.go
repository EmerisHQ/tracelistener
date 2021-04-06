package gaia_processor

const (
	insertBalanceQuery = `
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
)
