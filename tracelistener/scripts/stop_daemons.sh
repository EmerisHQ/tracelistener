#!/bin/sh

cd $HOME

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

cp -R "$HOME/.gaia/data/application.db" "$GOPATH/src/$TRACELISTENER_URL/tracelistener/bulk/testdata"

done

echo