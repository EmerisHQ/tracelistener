# read no.of nodes to be setup

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

echo "--------Gentx--------"

$DAEMON gentx validator 90000000000$DENOM --chain-id $CHAINID  --keyring-backend test

echo "----------collect-gentxs------------"

$DAEMON collect-gentxs

#create system services

echo "---------Creating $DAEMON_HOME system file---------"

echo "[Unit]
Description=${DAEMON} daemon
After=network.target
[Service]
Type=simple
User=$USER
ExecStart=$(which $DAEMON) start
Restart=on-failure
RestartSec=3
LimitNOFILE=4096
[Install]
WantedBy=multi-user.target" | sudo tee "/lib/systemd/system/$DAEMON.service"

echo "-------Start $DAEMON-${a} service-------"

sudo -S systemctl daemon-reload
sudo -S systemctl start $DAEMON.service

sleep 1s

echo
done

echo