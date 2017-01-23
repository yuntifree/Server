#!/bin/bash
go build ../access/appserver
./install.sh 1 appserver
go build  ../access/ossserver
./install.sh 1 ossserver
go build ../rpc/discover
./install.sh 2 discover
go build ../rpc/fetch
./install.sh 2 fetch
go build ../rpc/hot
./install.sh 2 hot
go build ../rpc/modify
./install.sh 2 modify
go build ../rpc/verify
./install.sh 2 verify
