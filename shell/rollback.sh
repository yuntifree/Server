#!/bin/bash
IPLIST="10.27.178.90 10.27.168.11"
HTTPDIR=/data/server
RPCDIR=/data/rpc
BACKDIR=/data/backup
HTTPLIST="appserver"
RPCLIST="discover fetch hot modify push verify"

function rollback_http()
{
	for ip in $IPLIST; do
        for srv in $HTTPLIST; do
            ssh root@$ip "cp -f $BACKDIR/$srv $HTTPDIR"
            ssh root@$ip "ps -ef|grep $HTTPDIR/$srv |grep -v grep|gawk -e '{print \$2}'|xargs kill -SIGUSR2"
        done
    done
}

function rollback_rpc()
{
	for ip in $IPLIST; do
        for srv in $RPCLIST; do
            ssh root@$ip "cp -f $BACKDIR/$srv $RPCDIR"
        ssh root@$ip "ps -ef|grep $RPCDIR/$srv |grep -v grep|gawk -e '{print \$2}'|xargs kill -s SIGTERM"
        ssh root@$ip "nohup $RPCDIR/$srv 1>>$RPCDIR/$srv.log 2>&1 &"
        done
    done
}

rollback_http
rollback_rpc
