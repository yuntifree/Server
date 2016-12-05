#!/bin/bash
ROOTDIR=/data/darren/Server
cd $ROOTDIR/access
go build -o appserver app.go
./release.sh 1 appserver
cd $ROOTDIR/rpc/discover
go build
./release.sh 2 discover
cd $ROOTDIR/rpc/fetch
go build
./release.sh 2 fetch
cd $ROOTDIR/rpc/hot
go build
./release.sh 2 hot
cd $ROOTDIR/rpc/modify
go build
./release.sh 2 modify
cd $ROOTDIR/rpc/verify
go build
./release.sh 2 verify
