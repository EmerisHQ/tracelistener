#!/bin/sh

cd $HOME

export DAEMON_HOME=~/.gaiad
export CHAINID=test
export DENOM=stake
export GH_URL=https://github.com/cosmos/gaia
export CHAIN_VERSION=v4.0.0 
export DAEMON=gaiad

echo "--------- Install $DAEMON ---------"
git clone $GH_URL && cd $(basename $_ .git)
git fetch && git checkout $CHAIN_VERSION
make install

cd $HOME

# check version
$DAEMON version --long


echo "--------Start initializing the chain ($CHAINID)---------"

$DAEMON init --chain-id $CHAINID $DAEMON_HOME

# add validators

$DAEMON keys add "validator" --keyring-backend test
$DAEMON keys add "delegator" --keyring-backend test

echo "----------Genesis creation---------"

$DAEMON add-genesis-account $($DAEMON keys show validator -a --keyring-backend test) 1000000000000$DENOM
$DAEMON add-genesis-account $($DAEMON keys show delegator -a --keyring-backend test) 1000000000000$DENOM

echo "--------gentx--------"

$DAEMON gentx validator 90000000000$DENOM --chain-id $CHAINID  --keyring-backend test

echo "----------collect-gentxs------------"

$DAEMON collect-gentxs

#start chain

$DAEMON start </dev/null &>/dev/null &

sleep 2s

echo

# get validator address
validator=$("${DAEMON}" keys show "validator" --bech val --keyring-backend test --output json)
valAddress=$(echo "${validator}" | jq -r '.address')

export valAddress="${valAddress}"

echo "-----------run delegation txs-----------"

dTx=$("${DAEMON}" tx staking delegate "${valAddress}" 10000"${DENOM}" --from delegator --fees 1000"${DENOM}" --chain-id "${CHAINID}" --keyring-backend test --output json -y)

sleep 6s

dtxHash=$(echo "${dTx}" | jq -r '.txhash')
echo "** TX HASH :: $dtxHash **"

# query the txhash and check the code
txResult=$("${DAEMON}" q tx "${dtxHash}" --output json)
dTxCode=$(echo "${txResult}"| jq -r '.code')

echo "Code is : $dTxCode"
echo
if [ "$dTxCode" -eq 0 ];
then
    echo "****** Delegation tx is successfull! *******"
else
    echo "****** Delegation tx is failed!! ******"
fi

echo "--------- Unbond txs -----------"

ubTx=$("${DAEMON}" tx staking unbond "${valAddress}" 10000"${DENOM}" --from "validator" --fees 1000"${DENOM}" --chain-id "${CHAINID}" --keyring-backend test --output json -y)

sleep 6s
    
ubtxHash=$(echo "${ubTx}" | jq -r '.txhash')
echo "** TX HASH :: $ubtxHash **"

# query the txhash and check the code
txResult=$("${DAEMON}" q tx "${ubtxHash}" --output json)
ubTxCode=$(echo "${txResult}"| jq -r '.code')

echo "Code is : $ubTxCode"
echo
if [ "$ubTxCode" -eq 0 ];
then
    echo "****** Unbond tx is successfull! ******"
else
    echo "****** Ubond tx is failed !!! ******"
fi

echo "------ run send tx -------"

# get delegator address
delegator=$("${DAEMON}" keys show "delegator" --bech val --keyring-backend test --output json)
delAddress=$(echo "${delegator}" | jq -r '.address')

export delAddress=${delAddress}

sendTx=$("${DAEMON}" tx bank send "validator" "${delAddress}" 10000"${DENOM}" --fees 1000"${DENOM}" --chain-id "${CHAINID}" --keyring-backend test --output json -y)

sleep 6s
    
sendtxHash=$(echo "${sendTx}" | jq -r '.txhash')
echo "** TX HASH :: $sendtxHash **"

# query the txhash and check the code
txResult=$("${DAEMON}" q tx "${sendtxHash}" --output json)
sendTxCode=$(echo "${txResult}"| jq -r '.code')

echo "Code is : $sendTxCode"
echo
if [ "$sendTxCode" -eq 0 ];
then
    echo "****** Send tx is successfull! ******"
else
    echo "****** Send tx is failed !!! ******"
fi

echo "-------stop gaiad---------"

killall $DAEMON

done

echo