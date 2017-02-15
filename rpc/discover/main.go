package main

import (
	"errors"
	"log"
	"net"
	"strconv"
	"strings"

	"github.com/coreos/etcd/clientv3"
	nsq "github.com/nsqio/go-nsq"

	"Server/proto/common"
	"Server/proto/discover"
	"Server/util"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	redis "gopkg.in/redis.v5"
)

const (
	addServer = iota
	dropServer
)

type server struct{}

var kv *redis.Client
var w *nsq.Producer

//Server server ip and port
type Server struct {
	host string
	port int32
}

type serviceMap map[string][]string

var srvMap serviceMap

func parseServer(name string) (Server, error) {
	var srv Server
	vals := strings.Split(name, ":")
	if len(vals) != 2 {
		log.Printf("length:%d", len(vals))
		return srv, errors.New("parse failed")
	}
	port, err := strconv.Atoi(vals[1])
	if err != nil {
		log.Printf("strconv failed, %s:%v", vals[1], err)
		return srv, err
	}
	srv.host = vals[0]
	srv.port = int32(port)
	return srv, nil
}

func updateServiceMap(mp serviceMap, key, srv string, op int64) {
	arr := mp[key]
	switch op {
	case addServer:
		if len(arr) == 0 {
			arr = append(arr, srv)
		} else {
			flag := false
			for i := 0; i < len(arr); i++ {
				if arr[i] == srv {
					flag = true
					break
				}
			}
			if !flag {
				arr = append(arr, srv)
			}
		}
		mp[key] = arr
	case dropServer:
		if len(arr) == 0 {
			break
		}
		idx := 0
		for i := 0; i < len(arr); i++ {
			if arr[i] == srv {
				idx = i
				break
			}
		}
		arr := append(arr[:idx], arr[idx+1:]...)
		mp[key] = arr
	}
}

func extractService(path string) string {
	arr := strings.Split(path, ":")
	if len(arr) != 3 {
		log.Printf("illegal service path:%s", path)
		return ""
	}
	return arr[1]
}

func watcher(cli *clientv3.Client) {
	rch := cli.Watch(context.Background(), "service", clientv3.WithPrefix())
	for wresp := range rch {
		for _, ev := range wresp.Events {
			log.Printf("%s %q : %q\n", ev.Type, ev.Kv.Key, ev.Kv.Value)
			service := extractService(string(ev.Kv.Key))
			if service == "" {
				continue
			}
			if ev.Type.String() == "PUT" {
				updateServiceMap(srvMap, service, string(ev.Kv.Value), addServer)
			} else if ev.Type.String() == "DELETE" {
				updateServiceMap(srvMap, service, string(ev.Kv.Value), dropServer)
			}
		}
	}
}

func fetchServers(name string) []string {
	vals, err := kv.ZRangeByScore(name, redis.ZRangeBy{Min: "-inf", Max: "+inf",
		Offset: 0, Count: 10}).Result()
	if err != nil {
		log.Printf("zrangebyscore failed %s:%v", name, err)
		return nil
	}

	var servers []string
	for i, key := range vals {
		log.Printf("%d:%s", i, key)
		servers = append(servers, key)
		if i >= 10 {
			break
		}
	}

	return servers
}

func isEtcdTestUid(uid int64) bool {
	if uid%2 == 1 {
		return true
	}
	return false
}

func convertServerName(name string) string {
	arr := strings.Split(name, ":")
	var server string
	if len(arr) == 3 {
		server = arr[1] + "-" + arr[2]
	} else if len(arr) == 2 {
		server = arr[1]
	}
	return server
}

func (s *server) Resolve(ctx context.Context, in *discover.ServerRequest) (*discover.ServerReply, error) {
	util.PubRPCRequest(w, "discover", "Resolve")
	log.Printf("resolve request uid:%d server:%s", in.Head.Uid, in.Sname)
	var servers []string
	if !isEtcdTestUid(in.Head.Uid) {
		servers = fetchServers(in.Sname)
	} else {
		name := convertServerName(in.Sname)
		servers = srvMap[name]
	}
	if len(servers) == 0 {
		log.Printf("fetch servers failed:%s", in.Sname)
		return &discover.ServerReply{
			Head: &common.Head{Retcode: common.ErrCode_FETCH_SERVER}}, nil
	}
	host := servers[util.Randn(int32(len(servers)))]
	util.PubRPCSuccRsp(w, "discover", "Resolve")
	return &discover.ServerReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Host: host}, nil
}

func main() {
	lis, err := net.Listen("tcp", util.DiscoverServerPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	srvMap = make(map[string][]string)
	w = util.NewNsqProducer()

	kv = util.InitRedis()
	go util.ReportHandler(kv, util.DiscoverServerName, util.DiscoverServerPort)
	cli := util.InitEtcdCli()
	go watcher(cli)

	s := grpc.NewServer()
	discover.RegisterDiscoverServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
