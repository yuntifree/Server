#!/bin/bash
go build ../access/appserver
./release.sh 1 appserver
go build ../rpc/discover
./release.sh 2 discover
go build ../rpc/fetch
./release.sh 2 fetch
go build ../rpc/hot
./release.sh 2 hot
go build ../rpc/modify
./release.sh 2 modify
go build ../rpc/verify
./release.sh 2 verify
