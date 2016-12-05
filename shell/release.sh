#!/bin/bash
# Need some system function.
. /etc/init.d/functions

HTTPDIR=/data/server
RPCDIR=/data/rpc
LOG=/var/log/release.log


function usage()
{
	echo "Usage: $0 [ TYPE ] SRC"
	echo ""
}

function abort()
{
	echo -e -n "$1" && failure && echo
	echo "[$(date +%F-%T)] [FAILED] $1" >> $LOG
	[ -n "$2" ] && exit "$2" || exit 1
}

# $1 -- ret
# $2 -- cmd
function check_ret()
{
	if [ "$1" == "0" ]; then
		ok "$2"
	else
		abort "$2"
	fi
}

function run()
{
	local ret
	local cmd=$*

	$cmd 
	ret=$?
	check_ret $ret "$cmd"
}

function install_http()
{
	for ip in $IPLIST; do
        scp $1 root@$ip:/tmp
        ssh root@$ip "mv -f /tmp/$1 $HTTPDIR"
        ssh root@$ip "ps -ef|grep $HTTPDIR/$1 |grep -v grep|gawk -e '{print \$2}'|xargs kill -SIGUSR2"
    done
    rm -f $1 
}

function install_rpc()
{
	for ip in $IPLIST; do
        scp $1 root@$ip:/tmp
        ssh root@$ip "mv -f /tmp/$1 $RPCDIR"
        ssh root@$ip "ps -ef|grep $RPCDIR/$1 |grep -v grep|gawk -e '{print \$2}'|xargs kill -s SIGTERM"
        ssh root@$ip "nohup $RPCDIR/$1 1>>$RPCDIR/$1.log 2>&1 &"
    done
    rm -f $1 
}

IPLIST="10.27.178.90 10.27.168.11"

if [ $# -lt 2 ]; then
    echo "not enough param"
    exit
fi

arr=$*
args=${arr[@]:2}

for arg in $args
do
    if [ $1 -eq 1 ]; then
        install_http $arg
    elif [ $1 -eq 2 ]; then
        install_rpc $arg
    else
        echo "illegal type"
    fi
done
