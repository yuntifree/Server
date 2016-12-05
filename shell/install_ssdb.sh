#!/bin/bash
wget --no-check-certificate https://github.com/ideawu/ssdb/archive/master.zip
unzip master
cd ssdb-master
make && make install
cd /usr/local/ssdb
./ssdb-server -d ssdb.conf
