#!/bin/bash
RPCLIST="discover fetch hot modify verify push punch"
HTTPLIST="appserver ossserver"
for srv in $HTTPLIST; do
    go build ../access/$srv
    ./install.sh 1 $srv
done

for srv in $RPCLIST; do
    go build ../rpc/$srv
    ./install.sh 2 $srv
done
