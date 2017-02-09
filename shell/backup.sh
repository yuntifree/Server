#!/bin/bash
IPLIST="10.27.178.90 10.27.168.11"
HTTPDIR=/data/server
RPCDIR=/data/rpc
BACKDIR=/data/backup
HTTPLIST="appserver"
RPCLIST="discover fetch hot modify push verify punch"

function backup_http()
{
	for ip in $IPLIST; do
        for srv in $HTTPLIST; do
            ssh root@$ip "cp -f $HTTPDIR/$srv $BACKDIR"
        done
    done
}

function backup_rpc()
{
	for ip in $IPLIST; do
        for srv in $RPCLIST; do
            ssh root@$ip "cp -f $RPCDIR/$srv $BACKDIR"
        done
    done
}

backup_http
backup_rpc
