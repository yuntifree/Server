package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"time"

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

func getStartTime(num int64, interval int64) time.Time {
	tt := util.TruncTime(time.Now(), int(interval))
	secs := -60 * num * interval
	return tt.Add(time.Duration(secs) * time.Second)
}

func getApiStat(db *sql.DB, name string, num int64) *monitor.ApiStat {
	start := getStartTime(num, 3)
	stime := start.Format(util.TimeFormat)
	var stat monitor.ApiStat
	stat.Name = name
	infos := make([]*monitor.ApiStatInfo, num+1)
	query := fmt.Sprintf("SELECT req, succrsp, FLOOR(TIMESTAMPDIFF(MINUTE, '%s', ctime)/3) FROM api_stat WHERE name = '%s' AND ctime > '%s' ORDER BY id DESC LIMIT %d", stime, name, stime, num)
	log.Printf("getApiStat query:%s", query)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("getApiStat query failed:%v", err)
		return &stat
	}

	defer rows.Close()
	for rows.Next() {
		var info monitor.ApiStatInfo
		var idx int64
		err := rows.Scan(&info.Req, &info.Succrsp, &idx)
		if err != nil {
			log.Printf("getApiStat scan failed:%v", err)
			continue
		}
		info.Ctime = start.Add(time.Duration(idx*3*60) * time.Second).Format(util.TimeFormat)
		infos[idx] = &info
	}
	stat.Records = infos[1:]
	return &stat
}

func getBatchApiStat(db *sql.DB, names []string, num int64) []*monitor.ApiStat {
	var infos []*monitor.ApiStat
	for i := 0; i < len(names); i++ {
		info := getApiStat(db, names[i], num)
		infos = append(infos, info)
	}
	return infos
}

func (s *server) GetBatchApiStat(ctx context.Context, in *monitor.BatchApiStatRequest) (*monitor.BatchApiStatReply, error) {
	util.PubRPCRequest(w, "monitor", "GetBatchApiStat")
	infos := getBatchApiStat(db, in.Names, in.Num)
	util.PubRPCSuccRsp(w, "monitor", "GetBatchApiStat")
	return &monitor.BatchApiStatReply{
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
