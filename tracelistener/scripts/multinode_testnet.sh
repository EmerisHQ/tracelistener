#/bin/sh

cd $HOME

export DAEMON_HOME=~/.gaia
export CHAINID=test
export DENOM=stake
export GH_URL=https://github.com/cosmos/gaia
export CHAIN_VERSION=v5.0.4 
export DAEMON=gaiad
export TRACELISTENER_URL=github.com/allinbits/tracelistener

# read no.of nodes to be setup, if passed argument is empty then default value is 2
NODES=$1
if [ -z $NODES ]
then
    NODES=2
fi

echo "**** Number of nodes to be setup: $NODES ****"

command_exists () {
    type "$1" &> /dev/null ;
}

cd $HOME

if command_exists go ; then
    echo "Golang is already installed"
else
  echo "Install dependencies"
  sudo apt update
  sudo apt-get -y upgrade
  sudo apt install build-essential jq -y

  wget https://dl.google.com/go/go1.17.3.linux-amd64.tar.gz
  tar -xvf go1.17.3.linux-amd64.tar.gz
  sudo mv go /usr/local
  rm go1.17.3.linux-amd64.tar.gz

  echo "------ Update bashrc ---------------"
  export GOPATH=$HOME/go
  export GOROOT=/usr/local/go
  export GOBIN=$GOPATH/bin
  export PATH=$PATH:/usr/local/go/bin:$GOBIN
  echo "" >> ~/.bashrc
  echo 'export GOPATH=$HOME/go' >> ~/.bashrc
  echo 'export GOROOT=/usr/local/go' >> ~/.bashrc
  echo 'export GOBIN=$GOPATH/bin' >> ~/.bashrc
  echo 'export PATH=$PATH:/usr/local/go/bin:$GOBIN' >> ~/.bashrc

  source ~/.bashrc

  mkdir -p "$GOBIN"
  mkdir -p $GOPATH/src/github.com

  go version
fi

echo "--------- Install $DAEMON ---------"
git clone $GH_URL && cd $(basename $_ .git)
git fetch && git checkout $CHAIN_VERSION
make install

cd $HOME

# check version
$DAEMON version --long

# export daemon home paths
for (( a=1; a<=$NODES; a++ ))
do
    export DAEMON_HOME_$a=$DAEMON_HOME-$a
    echo "Deamon path :: $DAEMON_HOME-$a"

    $DAEMON unsafe-reset-all  --home $DAEMON_HOME-$a
    echo "****** here command $DAEMON unsafe-reset-all  --home $DAEMON_HOME-$a ******"
done

# remove daemon home directories if already exists
for (( a=1; a<=$NODES; a++ ))
do
    rm -rf $DAEMON_HOME-$a
done

echo "-----Create daemon home directories------"

for (( a=1; a<=$NODES; a++ ))
do
    echo "****** create dir :: $DAEMON_HOME-$a ********"
    mkdir -p "$DAEMON_HOME-$a"
done

echo "--------Start initializing the chain ($CHAINID)---------"

for (( a=1; a<=$NODES; a++ ))
do
    echo "-------Init chain ${a}--------"
    echo "Deamon home :: $DAEMON_HOME-${a}"
    $DAEMON init --chain-id $CHAINID $DAEMON_HOME-${a} --home $DAEMON_HOME-${a}
done

# add validators
echo "---------Creating $NODES keys-------------"

for (( a=1; a<=$NODES; a++ ))
do
    $DAEMON keys add "validator${a}" --keyring-backend test --home $DAEMON_HOME-${a}
done

# add accounts if second argument is passed
if [ -z $ACCOUNTS ] || [ "$ACCOUNTS" -eq 0 ]
then
    echo "----- Argument for accounts is not present, not creating any additional accounts --------"
else
    echo "---------Creating $ACCOUNTS accounts-------------"

    for (( a=1; a<=$ACCOUNTS; a++ ))
    do
        $DAEMON keys add "account${a}" --keyring-backend test --home $DAEMON_HOME-1
    done
fi

echo "----------Genesis creation---------"

for (( a=1; a<=$NODES; a++ ))
do
    if [ $a == 1 ]
    then
        $DAEMON --home $DAEMON_HOME-$a add-genesis-account validator$a 1000000000000$DENOM  --keyring-backend test
        echo "done $DAEMON_HOME-$a genesis creation "
        continue
    fi
    $DAEMON --home $DAEMON_HOME-$a add-genesis-account validator$a 1000000000000$DENOM  --keyring-backend test
    $DAEMON --home $DAEMON_HOME-1 add-genesis-account $($DAEMON keys show validator$a -a --home $DAEMON_HOME-$a --keyring-backend test) 1000000000000$DENOM
done

echo "----------Genesis creation for accounts---------"

if [ -z $ACCOUNTS ]
then
    echo "Second argument was empty, so not setting up any account\n"
else
    for (( a=1; a<=$ACCOUNTS; a++ ))
    do
        # add accounts
        echo "cmd ::$DAEMON --home $DAEMON_HOME-1 add-genesis-account $($DAEMON keys show account$a -a --home $DAEMON_HOME-1 --keyring-backend test) 1000000000000$DENOM"

        $DAEMON --home $DAEMON_HOME-1 add-genesis-account $($DAEMON keys show account$a -a --home $DAEMON_HOME-1 --keyring-backend test) 1000000000000$DENOM
    done
fi

echo "--------Gentx--------"

for (( a=1; a<=$NODES; a++ ))
do
    $DAEMON gentx validator$a 90000000000$DENOM --chain-id $CHAINID  --keyring-backend test --home $DAEMON_HOME-$a
done

echo "---------Copy all gentxs to $DAEMON_HOME-1----------"

for (( a=2; a<=$NODES; a++ ))
do
    cp $DAEMON_HOME-$a/config/gentx/*.json $DAEMON_HOME-1/config/gentx/
done

echo "----------collect-gentxs------------"

$DAEMON collect-gentxs --home $DAEMON_HOME-1

echo "---------Updating $DAEMON_HOME-1 genesis.json ------------"

sed -i "s/172800000000000/600000000000/g" $DAEMON_HOME-1/config/genesis.json
sed -i "s/172800s/600s/g" $DAEMON_HOME-1/config/genesis.json
sed -i "s/stake/$DENOM/g" $DAEMON_HOME-1/config/genesis.json

echo "---------Distribute genesis.json of $DAEMON_HOME-1 to remaining nodes-------"

for (( a=2; a<=$NODES; a++ ))
do
    cp $DAEMON_HOME-1/config/genesis.json $DAEMON_HOME-$a/config/
done

echo "---------Getting public IP address-----------"

IP="$(dig +short myip.opendns.com @resolver1.opendns.com)"
echo "Public IP address: ${IP}"

if [ -z $IP ]
then
    IP=127.0.0.1
fi

for (( a=1; a<=$NODES; a++ ))
do
    DIFF=`expr $a - 1`
    INC=`expr $DIFF \* 2`
    LADDR=`expr 16656 + $INC` #laddr ports

    echo "----------Get node-id of $DAEMON_HOME-$a ---------"
    nodeID=$("${DAEMON}" tendermint show-node-id --home $DAEMON_HOME-$a)
    echo "** Node ID :: $nodeID **"
    PR="$nodeID@$IP:$LADDR"
    if [ $a == 1 ]
    then
        PERSISTENT_PEERS="${PR}"
        continue
    fi

    PERSISTENT_PEERS="${PERSISTENT_PEERS},${PR}"
    #echo "PERSISTENT_PEERS : $PERSISTENT_PEERS"
done

echo '*** PERSISTENT_PEERS : "'"${PERSISTENT_PEERS}"'" *****'

#update configurations
for (( a=1; a<=$NODES; a++ ))
do

    DIFF=`expr $a - 1`
    INC=`expr $DIFF \* 2`

    RPC=`expr 16657 + $INC` #increment rpc ports
    LADDR=`expr 16656 + $INC` #laddr ports
    GRPC=`expr 9090 + $INC` # grpc poprt
    WGRPC=`expr 9091 + $INC` # web grpc port

    echo "PORTS OF RPC :: $RPC , LADDR :: $LADDR , GRPC :: $GRPC , WGRPC :: $WGRPC"

    echo "----------Updating $DAEMON_HOME-$a chain config-----------"

    sed -i 's#tcp://127.0.0.1:26657#tcp://0.0.0.0:'${RPC}'#g' $DAEMON_HOME-$a/config/config.toml
    sed -i 's#tcp://0.0.0.0:26656#tcp://0.0.0.0:'${LADDR}'#g' $DAEMON_HOME-$a/config/config.toml
    sed -i '/persistent_peers =/c\persistent_peers = "'"$PERSISTENT_PEERS"'"' $DAEMON_HOME-$a/config/config.toml
    sed -i '/allow_duplicate_ip =/c\allow_duplicate_ip = true' $DAEMON_HOME-$a/config/config.toml
    sed -i '/pprof_laddr =/c\# pprof_laddr = "localhost:6060"' $DAEMON_HOME-$a/config/config.toml

    sed -i 's#0.0.0.0:9090#0.0.0.0:'${GRPC}'#g' $DAEMON_HOME-$a/config/app.toml
    sed -i 's#0.0.0.0:9091#0.0.0.0:'${WGRPC}'#g' $DAEMON_HOME-$a/config/app.toml

    sed -i '/max_num_inbound_peers =/c\max_num_inbound_peers = 140' $DAEMON_HOME-$a/config/config.toml
    sed -i '/max_num_outbound_peers =/c\max_num_outbound_peers = 110' $DAEMON_HOME-$a/config/config.toml

done

#start chains
for (( a=1; a<=$NODES; a++ ))
do
    DIFF=`expr $a - 1`
    INC=`expr $DIFF \* 2`

    RPC=`expr 16657 + $INC` #increment rpc ports

    $DAEMON start --home $DAEMON_HOME-$a </dev/null &>/dev/null &

    sleep 4s

    echo "Checking $DAEMON_HOME-${a} chain status"

    $DAEMON status --node tcp://localhost:${RPC}

    echo
done

echo