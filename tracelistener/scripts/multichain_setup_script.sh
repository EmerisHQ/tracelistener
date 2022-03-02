#!/usr/bin/bash

cd $HOME

export DAEMON_HOME=~/.gaia_test
export CHAINID=test
export DENOM=stake
export GH_URL=https://github.com/cosmos/gaia
export CHAIN_VERSION=v6.0.0  
export DAEMON=gaiad
export TRACELISTENER_URL=github.com/allinbits/tracelistener

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

SEED1="grief wash cry suggest royal coyote cover payment salute version matter truth science bracket project gasp royal paper menu order wreck polar false tornado"
SEED2="welcome strike afford royal issue wife damage trip aware critic spy caution phone world parent sock flush captain weapon dream bag fame vicious private"

CHAINS=2

for (( c=1; c<=$CHAINS; c++ ))
do
    export DAEMON_HOME_$c=$DAEMON_HOME-$c
    echo "Deamon path :: $DAEMON_HOME-$c"

    $DAEMON unsafe-reset-all  --home $DAEMON_HOME-$c
    echo "****** here command $DAEMON unsafe-reset-all  --home $DAEMON_HOME-$c ******"

    # remove daemon home directories if already exists
    rm -rf $DAEMON_HOME-$c

    echo "-----Create daemon home directories------"

   
    echo "****** create dir :: $DAEMON_HOME-$c ********"
    mkdir -p "$DAEMON_HOME-$c"
    

    echo "--------Start initializing the chain ($CHAINID-$c)---------"

   
    echo "-------Init chain ${a}--------"
    echo "Deamon home :: $DAEMON_HOME-$c"
    $DAEMON init --chain-id $CHAINID-$c $DAEMON_HOME-${c} --home $DAEMON_HOME-${c}
    

    # add validators
    echo "---------Creating keys-------------"

    if [ $c == 1 ]
    then
        echo $SEED1 | $DAEMON keys add "validator${c}" --keyring-backend test --home $DAEMON_HOME-${c} --recover
    else
        echo $SEED2 | $DAEMON keys add "validator${c}" --keyring-backend test --home $DAEMON_HOME-${c} --recover
    fi

    
    echo "----------Genesis creation---------"
       
    $DAEMON --home $DAEMON_HOME-$c add-genesis-account $($DAEMON keys show validator$c -a --home $DAEMON_HOME-$c --keyring-backend test) 1000000000000$DENOM

    echo "--------Gentx--------"

    $DAEMON gentx validator$c 90000000000$DENOM --chain-id $CHAINID-$c  --keyring-backend test --home $DAEMON_HOME-$c

    echo "----------collect-gentxs------------"

    $DAEMON collect-gentxs --home $DAEMON_HOME-$c

    echo "---------Updating $DAEMON_HOME-$c genesis.json ------------"

    sed -i "s/172800000000000/600000000000/g" $DAEMON_HOME-$c/config/genesis.json
    sed -i "s/172800s/600s/g" $DAEMON_HOME-$c/config/genesis.json
    sed -i "s/stake/$DENOM/g" $DAEMON_HOME-$c/config/genesis.json


    DIFF=`expr $c - 1`
    INC=`expr $DIFF \* 2`

    RPC=`expr 16657 + $INC` #increment rpc ports
    LADDR=`expr 16656 + $INC` #laddr ports
    GRPC=`expr 9090 + $INC` # grpc poprt
    WGRPC=`expr 9091 + $INC` # web grpc port

    echo "PORTS OF RPC :: $RPC , LADDR :: $LADDR , GRPC :: $GRPC , WGRPC :: $WGRPC"

    echo "----------Updating $DAEMON_HOME-$c chain config-----------"

    sed -i 's#tcp://127.0.0.1:26657#tcp://0.0.0.0:'${RPC}'#g' $DAEMON_HOME-$c/config/config.toml
    sed -i 's#tcp://0.0.0.0:26656#tcp://0.0.0.0:'${LADDR}'#g' $DAEMON_HOME-$c/config/config.toml
    #sed -i '/persistent_peers =/c\persistent_peers = "'"$PERSISTENT_PEERS"'"' $DAEMON_HOME-$c/config/config.toml
    sed -i '/allow_duplicate_ip =/c\allow_duplicate_ip = true' $DAEMON_HOME-$c/config/config.toml
    sed -i '/pprof_laddr =/c\# pprof_laddr = "localhost:6060"' $DAEMON_HOME-$c/config/config.toml

    sed -i 's#0.0.0.0:9090#0.0.0.0:'${GRPC}'#g' $DAEMON_HOME-$c/config/app.toml
    sed -i 's#0.0.0.0:9091#0.0.0.0:'${WGRPC}'#g' $DAEMON_HOME-$c/config/app.toml

    sed -i '/max_num_inbound_peers =/c\max_num_inbound_peers = 140' $DAEMON_HOME-$c/config/config.toml
    sed -i '/max_num_outbound_peers =/c\max_num_outbound_peers = 110' $DAEMON_HOME-$c/config/config.toml

    #start chains
   
    $DAEMON start --home $DAEMON_HOME-$c </dev/null &>/dev/null &

    sleep 4s

    echo "Checking $DAEMON_HOME-${c} chain status"

    $DAEMON status --node tcp://localhost:${RPC}

    echo
done