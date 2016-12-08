#!/bin/bash
cd /data/darren/certbot
./certbot-auto renew

IPLIST="10.27.178.90 10.27.168.11"
SERVER=/data/server/appserver
for ip in $IPLIST; do
    scp /etc/letsencrypt/live/yunxingzh.com/* root@$ip:/etc/letsencrypt/live/yunxingzh.com/
    ssh root@$ip "ps -ef|grep $SERVER |grep -v grep|gawk -e '{print \$2}'|xargs kill -SIGUSR2"
    echo "restart $ip appserver..."
done

