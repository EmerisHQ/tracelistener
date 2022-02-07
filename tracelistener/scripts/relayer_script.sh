#!/usr/bin/bash

cd $HOME

export RELAYER_HOME=~/.relayer_test
export CHAINID=test
export DENOM=stake
export GH_URL=https://github.com/cosmos/relayer.git
export CHAIN_VERSION=v0.9.3
export DAEMON=rly
export RLYKEY=key
export DOMAIN=localhost
export ACCOUNT_PREFIX=cosmos
export PTH=rly_test
export RELAYER_DAEMON=rly

CHAINS=2

echo "--------- Install $DAEMON ---------"
git clone $GH_URL && cd $(basename $_ .git)
git fetch && git checkout $CHAIN_VERSION
make install

echo "----- remove relayer home dir if already exists -------"

rm -rf $RELAYER_HOME

SEED1="grief wash cry suggest royal coyote cover payment salute version matter truth science bracket project gasp royal paper menu order wreck polar false tornado"
SEED2="welcome strike afford royal issue wife damage trip aware critic spy caution phone world parent sock flush captain weapon dream bag fame vicious private"

$RELAYER_DAEMON config init --home $RELAYER_HOME

#cd $RELAYER_HOME

mkdir -p test

echo "------ create json files ---------"

for (( a=1; a<=$CHAINS; a++ ))
do
    DIFF=`expr $a - 1`
    INC=`expr $DIFF \* 2`

    RPC=`expr 16657 + $INC` #increment rpc ports

    echo "{\"key\":\"$RLYKEY-"${a}"\",\"chain-id\":\"$CHAINID-"${a}"\",\"rpc-addr\":\"http://$DOMAIN:$RPC\",\"account-prefix\":\"$ACCOUNT_PREFIX\",\"gas-adjustment\": 1.5,\"gas\":200000,\"gas-prices\":\"0$DENOM\",\"default-denom\":\"$DENOM\",\"trusting-period\":\"330h\"}" > test/$CHAINID-$a.json

    echo "------- add chains-------------"
    $RELAYER_DAEMON chains add -f test/$CHAINID-$a.json --home $RELAYER_HOME

    echo "---------restore keys with existing seeds--------"
    if [ $a == 1 ]
    then
        $RELAYER_DAEMON keys restore $CHAINID-$a "$RLYKEY-$a" "$SEED1" --home $RELAYER_HOME

        export SRC=$CHAINID-$a
    else
        $RELAYER_DAEMON keys restore $CHAINID-$a "$RLYKEY-$a" "$SEED2" --home $RELAYER_HOME

        export DST=$CHAINID-$a
    fi

    echo "----------create a light client----------"
    $RELAYER_DAEMON light init $CHAINID-$a -f --home $RELAYER_HOME

done

echo

echo "---------create a test/$PTH.json------------"
echo "{\"src\":{\"chain-id\":\"$SRC\",\"port-id\":\"transfer\",\"order\":\"unordered\",\"version\":\"ics20-1\"},\"dst\":{\"chain-id\":\"$DST\",\"port-id\":\"transfer\",\"order\":\"unordered\",\"version\":\"ics20-1\"},\"strategy\":{\"type\":\"naive\"}}" > test/$PTH.json

echo "------ add a path between $SRC and $DST ----"
$RELAYER_DAEMON pth add $SRC $DST $PTH -f test/$PTH.json --home $RELAYER_HOME

echo "-----show keys----------"
$RELAYER_DAEMON keys show $SRC --home $RELAYER_HOME
$RELAYER_DAEMON keys show $DST --home $RELAYER_HOME

echo "--------link path--------------"
$RELAYER_DAEMON tx link $PTH --home $RELAYER_HOME

sleep 2s

echo "-------start relayer--------"
$RELAYER_DAEMON start $PTH --home $RELAYER_HOME </dev/null &>/dev/null &

sleep 2s

echo "----- run transfer tx from $SRC to $DST---------"
$RELAYER_DAEMON tx transfer $SRC $DST 1000${DENOM} $($RELAYER_DAEMON ch addr $DST) --home $RELAYER_HOME

echo

echo "----- run transfer tx from $DST to $SRC---------"
$RELAYER_DAEMON tx transfer $DST $SRC 1000${DENOM} $($RELAYER_DAEMON ch addr $SRC) --home $RELAYER_HOME
