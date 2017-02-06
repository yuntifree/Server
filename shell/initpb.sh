#!/bin/bash
RPCLIST="discover fetch hot modify push verify"

function gen_rpc()
{
    for srv in $RPCLIST; do 
        echo $srv
        protoc --go_out=plugins=grpc:../proto/$srv/ ../proto/$srv/$srv.proto -I../.. -I../proto/$srv
     done
}

function gen_comm()
{
    protoc --go_out=../proto/common ../proto/common/common.proto -I../proto/common
}

gen_comm
gen_rpc
