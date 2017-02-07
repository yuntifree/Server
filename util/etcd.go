package util

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/coreos/etcd/clientv3"
)

//InitEtcdCli return etcd client
func InitEtcdCli() *clientv3.Client {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints: []string{"10.26.210.175:2379", "10.27.178.90:2379",
			"10.27.168.11:2379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}
	return cli
}

func report(cli *clientv3.Client, key, val string) {
	resp, err := cli.Grant(context.TODO(), 5)
	if err != nil {
		log.Fatal(err)
	}
	_, err = cli.Put(context.TODO(), key, val,
		clientv3.WithLease(resp.ID))
	if err != nil {
		log.Fatal(err)
	}
}

//ReportEtcd service report host and port to etcd
func ReportEtcd(cli *clientv3.Client, server, port string) {
	host, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}
	ip := GetInnerIP()
	addr := ip + port
	var name string
	if ip == DebugHost {
		name = server + "-debug:" + host
	} else {
		name = server + ":" + host
	}
	for {
		time.Sleep(time.Second * 2)
		report(cli, name, addr)
	}
}
