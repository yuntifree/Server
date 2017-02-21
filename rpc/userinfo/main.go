package main

import (
	"database/sql"
	"fmt"
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
	femaleType = 0
	maleType   = 1
	saveRate   = 0.1 / (1024.0 * 1024.0)
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

func getDefHead(db *sql.DB, stype int64) []*userinfo.HeadInfo {
	var infos []*userinfo.HeadInfo
	rows, err := db.Query("SELECT headurl, description, age FROM default_head WHERE deleted = 0 AND sex = ?", stype)
	if err != nil {
		log.Printf("getDefHead query failed:%v", err)
		return infos
	}

	defer rows.Close()
	for rows.Next() {
		var info userinfo.HeadInfo
		err := rows.Scan(&info.Headurl, &info.Desc, &info.Age)
		if err != nil {
			log.Printf("getDefHead scan failed:%v", err)
			continue
		}
		infos = append(infos, &info)
	}
	return infos
}

func (s *server) GetDefHead(ctx context.Context, in *common.CommRequest) (*userinfo.HeadReply, error) {
	util.PubRPCRequest(w, "userinfo", "GetDefHead")
	male := getDefHead(db, maleType)
	female := getDefHead(db, femaleType)
	util.PubRPCSuccRsp(w, "userinfo", "GetDefHead")
	return &userinfo.HeadReply{
		Head: &common.Head{Retcode: 0}, Male: male,
		Female: female}, nil
}

func (s *server) ModInfo(ctx context.Context, in *userinfo.InfoRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "userinfo", "ModInfo")
	query := "UPDATE user SET atime = NOW() "
	if in.Headurl != "" {
		query += ", headurl = '" + in.Headurl + "' "
	}
	if in.Nickname != "" {
		query += ", nickname = '" + in.Nickname + "' "
	}
	query += fmt.Sprintf(" WHERE uid = %d", in.Head.Uid)
	log.Printf("ModInfo query:%s", query)
	_, err := db.Exec(query)
	if err != nil {
		log.Printf("ModInfo query failed:%v", err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1}}, nil
	}
	util.PubRPCSuccRsp(w, "userinfo", "ModInfo")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0}}, nil
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
