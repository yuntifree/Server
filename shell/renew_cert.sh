#!/bin/bash
IPLIST="10.27.178.90 10.27.168.11"
SERVER=/data/server/appserver

#for wx.yunxingzh.com
cd /data/darren/certbot
./certbot-auto renew
cp -f /etc/letsencrypt/live/wx.yunxingzh.com/fullchain.pem /data/server
cp -f /etc/letsencrypt/live/wx.yunxingzh.com/privkey.pem /data/server
ps -ef|grep $SERVER |grep -v grep|gawk -e '{print $2}'|xargs kill -12

#for api.yunxingzh.com
ssh root@10.27.178.90 "/data/certbot/certbot-auto renew"
scp  root@10.27.178.90:/etc/letsencrypt/live/api.yunxingzh.com/fullchain.pem /tmp/
scp  root@10.27.178.90:/etc/letsencrypt/live/api.yunxingzh.com/privkey.pem /tmp/
for ip in $IPLIST; do
    scp  /tmp/*.pem root@$ip:/data/server
    ssh root@$ip "ps -ef|grep $SERVER |grep -v grep|gawk -e '{print $2}'|xargs kill -12"
    echo "restart $ip appserver..."
done

