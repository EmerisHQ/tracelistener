#!/usr/bin/bash

cd $HOME

export DAEMON_HOME=.gaia_test
export RELAYER_HOME=.relayer_test
export DAEMON=gaiad
export TRACELISTENER_URL=github.com/allinbits/tracelistener
export RELAYER_HOME=~/.relayer_test
export RELAYER_DAEMON=rly


echo "-------stop gaiad---------"

killall $DAEMON

echo "----------stop relayer---------"

killall $RELAYER_DAEMON

echo "------- move application.db to testdata--------------"

mkdir -p "$GOPATH/src/$TRACELISTENER_URL/tracelistener/bulk/testdata"

cp -R "$HOME/$DAEMON_HOME-1/data/application.db" "$GOPATH/src/$TRACELISTENER_URL/tracelistener/bulk/testdata"

echo "-------------remove test folders-------------"

rm -rf $HOME/DAEMON_HOME
rm -rf $HOME/RELAYER_HOME

echo "-----DONE------"

echo