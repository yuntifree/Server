package main

import (
	"database/sql"
	"log"
	"net"

	"Server/proto/common"
	"Server/proto/userinfo"
	"Server/util"

	_ "github.com/go-sql-driver/mysql"
	nsq "github.com/nsqio/go-nsq"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	saveRate = 0.1 / (1024.0 * 1024.0)
)

type server struct{}

var db *sql.DB
var w *nsq.Producer

func (s *server) GetInfo(ctx context.Context, in *common.CommRequest) (*userinfo.InfoReply, error) {
	util.PubRPCRequest(w, "userinfo", "GetInfo")
	var headurl, nickname string
	var total, save int64
	err := db.QueryRow("SELECT headurl, nickname, times, traffic FROM user WHERE uid = ?", in.Head.Uid).Scan(&headurl, &nickname, &total, &save)
	if err != nil {
		log.Printf("GetInfo query failed:%v", err)
		return &userinfo.InfoReply{
			Head: &common.Head{Retcode: 1}}, nil
	}
	save = int64(float64(save) * saveRate)
	util.PubRPCSuccRsp(w, "userinfo", "GetInfo")
	return &userinfo.InfoReply{
		Head: &common.Head{Retcode: 0}, Headurl: headurl, Nickname: nickname,
		Total: total, Save: save}, nil
}

func main() {
	lis, err := net.Listen("tcp", util.UserinfoServerPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	w = util.NewNsqProducer()
	db, err = util.InitDB(true)
	if err != nil {
		log.Fatalf("failed to init db connection: %v", err)
	}
	db.SetMaxIdleConns(util.MaxIdleConns)
	kv := util.InitRedis()
	go util.ReportHandler(kv, util.UserinfoServerName, util.UserinfoServerPort)
	//cli := util.InitEtcdCli()
	//go util.ReportEtcd(cli, util.UserinfoServerName, util.UserinfoServerPort)

	s := grpc.NewServer()
	userinfo.RegisterUserinfoServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
