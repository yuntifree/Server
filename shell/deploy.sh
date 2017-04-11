#!/bin/bash
RPCLIST="discover fetch hot modify verify push advertise config userinfo monitor"
go build ../access/appserver
./release.sh 1 appserver

for srv in $RPCLIST; do
    go build ../rpc/$srv
    ./release.sh 2 $srv
done
