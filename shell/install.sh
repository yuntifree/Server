#!/bin/bash
SRV_DST=/data/server
RPC_DST=/data/rpc

function install_server()
{
    install $1 $SRV_DST
    killall $1
    ulimit -c unlimited

    cd $SRV_DST
    nohup $SRV_DST/$1 >> $SRV_DST/$1.log 2>&1 &
    cd -
}

function install_rpc()
{
    install $1 $RPC_DST
    killall $1
    ulimit -c unlimited

    cd $RPC_DST
    nohup $RPC_DST/$1 >> $RPC_DST/$1.log 2>&1 &
    cd -
}

if [ $# -lt 2 ]; then
    echo "not enough param"
    exit
fi

arr=$*
args=${arr[@]:2}

for arg in $args
do
    if [ $1 -eq 1 ]; then
        install_server $arg
    elif [ $1 -eq 2 ]; then
        install_rpc $arg
    fi
done
