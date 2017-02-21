package main

import (
	"database/sql"
	"log"
	"net"

	"Server/proto/common"
	"Server/proto/monitor"
	"Server/util"

	_ "github.com/go-sql-driver/mysql"
	nsq "github.com/nsqio/go-nsq"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	redis "gopkg.in/redis.v5"
)

type server struct{}

var db *sql.DB
var kv *redis.Client
var w *nsq.Producer

func getApi(db *sql.DB) []*monitor.ApiInfo {
	var infos []*monitor.ApiInfo
	rows, err := db.Query("SELECT id, name, description FROM api WHERE deleted = 0")
	if err != nil {
		log.Printf("getApi query failed:%v", err)
		return infos
	}

	defer rows.Close()
	for rows.Next() {
		var info monitor.ApiInfo
		err := rows.Scan(&info.Id, &info.Name, &info.Desc)
		if err != nil {
			log.Printf("getApi scan failed:%v", err)
			continue
		}
		infos = append(infos, &info)
	}
	return infos
}

func (s *server) GetApi(ctx context.Context, in *common.CommRequest) (*monitor.ApiReply, error) {
	util.PubRPCRequest(w, "monitor", "GetApi")
	infos := getApi(db)
	util.PubRPCSuccRsp(w, "monitor", "GetApi")
	return &monitor.ApiReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Infos: infos}, nil
}

func main() {
	lis, err := net.Listen("tcp", util.MonitorServerPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	w = util.NewNsqProducer()

	db, err = util.InitMonitorDB()
	if err != nil {
		log.Fatalf("failed to init db connection: %v", err)
	}
	db.SetMaxIdleConns(util.MaxIdleConns)
	kv = util.InitRedis()
	go util.ReportHandler(kv, util.MonitorServerName, util.MonitorServerPort)
	//cli := util.InitEtcdCli()
	//go util.ReportEtcd(cli, util.ConfigServerName, util.ConfigServerPort)

	s := grpc.NewServer()
	monitor.RegisterMonitorServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
