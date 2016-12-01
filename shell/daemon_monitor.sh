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

check_http
check_rpc
check_ssdb
