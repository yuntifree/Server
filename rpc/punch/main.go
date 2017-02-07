package main

import (
	"database/sql"
	"log"
	"net"

	redis "gopkg.in/redis.v5"

	"Server/proto/common"
	"Server/proto/punch"
	"Server/util"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type server struct{}

var db *sql.DB

var kv *redis.Client

func (s *server) Punch(ctx context.Context, in *punch.PunchRequest) (*common.CommReply, error) {
	log.Printf("punch request uid:%d apmac:%s", in.Head.Uid, in.Apmac)
	var aid int64
	err := db.QueryRow("SELECT id FROM ap WHERE mac = ?", in.Apmac).Scan(&aid)
	if err != nil {
		log.Printf("Punch query failed:%v", err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	res, err := db.Exec("INSERT IGNORE INTO punch(aid, uid, ctime) VALUES (?, ?, NOW())",
		aid, in.Head.Uid)
	if err != nil {
		log.Printf("Punch insert record failed:%v", err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Printf("Punch insert record failed:%v", err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	if id == 0 {
		log.Printf("Punch insert id zero, aid:%d, uid:%d", aid, in.Head.Uid)
		return &common.CommReply{
			Head: &common.Head{Retcode: common.ErrCode_HAS_PUNCH,
				Uid: in.Head.Uid}}, nil
	}
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func (s *server) Praise(ctx context.Context, in *punch.PunchRequest) (*common.CommReply, error) {
	log.Printf("praise request uid:%d apmac:%s", in.Head.Uid, in.Apmac)
	var aid int64
	err := db.QueryRow("SELECT id FROM ap WHERE mac = ?", in.Apmac).Scan(&aid)
	if err != nil {
		log.Printf("Punch query failed:%v", err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	_, err = db.Exec("INSERT IGNORE INTO punch_praise(aid, uid, ctime) VALUES (?, ?, NOW())",
		aid, in.Head.Uid)
	if err != nil {
		log.Printf("Punch insert record failed:%v", err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func main() {
	lis, err := net.Listen("tcp", util.PunchServerPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	db, err = util.InitDB(true)
	if err != nil {
		log.Fatalf("failed to init db connection: %v", err)
	}
	db.SetMaxIdleConns(util.MaxIdleConns)

	kv = util.InitRedis()
	go util.ReportHandler(kv, util.PunchServerName, util.PunchServerPort)
	cli := util.InitEtcdCli()
	go util.ReportEtcd(cli, util.PunchServerName, util.PunchServerPort)

	s := grpc.NewServer()
	punch.RegisterPunchServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
