#!/bin/bash
IPLIST="10.27.178.90 10.27.168.11"
HTTPDIR=/data/server
RPCDIR=/data/rpc
BACKDIR=/data/backup
HTTPLIST="appserver"
RPCLIST="discover fetch hot modify push verify punch"

function rollback_http()
{
	for ip in $IPLIST; do
        for srv in $HTTPLIST; do
            ssh root@$ip "cp -f $BACKDIR/$srv $HTTPDIR"
            local n=`ssh root@$ip "ps -ef|grep $HTTPDIR/$srv |grep -v grep|gawk -e '{print $2}'|wc -l"`
            if [ $n -eq 0 ]; then
                ssh root@$ip "nohup $HTTPDIR/$srv 1>>$HTTPDIR/$srv.log 2>&1 &"
            else 
                ssh root@$ip "ps -ef|grep $HTTPDIR/$srv |grep -v grep|gawk -e '{print $2}'|xargs kill -SIGUSR2"
            fi
        done
    done
}

function rollback_rpc()
{
	for ip in $IPLIST; do
        for srv in $RPCLIST; do
            ssh root@$ip "cp -f $BACKDIR/$srv $RPCDIR"
        ssh root@$ip "ps -ef|grep $RPCDIR/$srv |grep -v grep|gawk -e '{print $2}'|xargs kill -s SIGTERM"
        ssh root@$ip "nohup $RPCDIR/$srv 1>>$RPCDIR/$srv.log 2>&1 &"
        done
    done
}

function rollback_single_rpc()
{
    for ip in $IPLIST; do
        ssh root@$ip "cp -f $BACKDIR/$1 $RPCDIR"
        ssh root@$ip "ps -ef|grep $RPCDIR/$1 |grep -v grep|gawk -e '{print $2}'|xargs kill -s SIGTERM"
        ssh root@$ip "nohup $RPCDIR/$1 1>>$RPCDIR/$1.log 2>&1 &"
    done
}

function check_http()
{
    if [ "$1" = "appserver" ]; then
        echo 1
    else
        echo 0
    fi
}

function check_rpc()
{
    local n=`echo -n $RPCLIST |grep -w "$1" | wc -l`
    echo $n
}

if [ $# -lt 1 ]; then
    echo "not enough param"
    exit
fi

echo $1

http=$(check_http $1)
echo "after checkHTTP" $http
rpc=$(check_rpc $1)
echo $rpc

if [ "$1" = "all" ]; then
    rollback_http
    rollback_rpc
elif [ $http = 1 ]; then
    rollback_http
elif [ $rpc = 1 ]; then
    rollback_single_rpc $1
fi

