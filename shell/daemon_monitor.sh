#!/bin/bash

HTTPSRV=(appserver ossserver)
RPCSRV=(discover fetch hot modify verify)
LOG=/var/log/srv.log
ERR=/var/log/srv.err


HTTP_DIR=/data/server
RPC_DIR=/data/rpc

function log()
{
    echo "[$(date "+%F %T"),000] [agent]$1" >> $LOG
    echo "$1"
}

function err()
{
    echo "[$(date "+%F %T"),000] [agent]$1" >> $ERR
    echo "$1"
}

function pullhttp()
{
    nohup $HTTP_DIR/$1 1>>$HTTP_DIR/$1.log 2>&1 &
}

function pullrpc()
{
    nohup $RPC_DIR/$1 1>>$RPC_DIR/$1.log 2>&1 &
}

function check_http()
{
    for srv in ${HTTPSRV[@]}; do
        sname=$HTTP_DIR/$srv
        if [ -z "$(ps -ef |grep $sname| grep -v grep|grep -v $sname.log)" ]; then
            err "Server $sname not running, restart."
            pullhttp $srv
        fi
    done
}

function check_rpc()
{
    for srv in ${RPCSRV[@]}; do
        sname=$RPC_DIR/$srv
        if [ -z "$(ps -ef |grep $sname| grep -v grep|grep -v $sname.log)" ]; then
            err "Server $sname not running, restart."
            pullrpc $srv
        fi
    done
}

function check_ssdb()
{
    sbase=/usr/local/ssdb/
    sname=ssdb-server
    sconf=ssdb.conf
    if [ -z "$(ps -ef |grep $sbase$sname| grep -v grep)" ]; then
        err "Server $sbase$sname not running, restart."
        nohup $sbase$sname -d $sbase$sconf -s restart
    fi
}

function check_etcd()
{
    ip=`ifconfig  | grep 'inet '|grep -v '127.0.0.1' |grep ' 10.'|gawk '{print $2}'`
    if [ -z "$(ps -ef |grep etcd| grep -v grep)" ]; then
        err "Server etcd not running, restart."
        if [ "$ip" = "10.26.210.175" ]; then
            nohup /usr/local/etcd/etcd --name infra0 --data-dir /data/infra0.etcd --initial-advertise-peer-urls http://10.26.210.175:2380 --listen-peer-urls http://10.26.210.175:2380 --listen-client-urls http://10.26.210.175:2379,http://127.0.0.1:2379 --advertise-client-urls http://10.26.210.175:2379 --initial-cluster-token etcd-cluster-1 1>>/data/etcd.log 2>&1 &
        elif [ "$ip" = "10.27.178.90" ]; then 
            nohup /usr/local/etcd/etcd --name infra1 --data-dir /data/infra1.etcd --initial-advertise-peer-urls http://10.27.178.90:2380 --listen-peer-urls http://10.27.178.90:2380 --listen-client-urls http://10.27.178.90:2379,http://127.0.0.1:2379 --advertise-client-urls http://10.27.178.90:2379 --initial-cluster-token etcd-cluster-1 1>>/data/etcd.log 2>&1 &
        elif [ "$ip" = "10.27.168.11" ]; then
            nohup /usr/local/etcd/etcd --name infra2 --data-dir /data/infra2.etcd --initial-advertise-peer-urls http://10.27.168.11:2380 --listen-peer-urls http://10.27.168.11:2380 --listen-client-urls http://10.27.168.11:2379,http://127.0.0.1:2379 --advertise-client-urls http://10.27.168.11:2379 --initial-cluster-token etcd-cluster-1 1>>/data/etcd.log 2>&1 &
        fi
    fi
}

check_http
check_rpc
check_ssdb
check_etcd
