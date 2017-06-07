package main

import (
	"Server/proto/common"
	"Server/proto/inquiry"
	"Server/util"
	"Server/weixin"
	"database/sql"
	"log"
	"net"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/net/context"
	redis "gopkg.in/redis.v5"
)

type server struct{}

var db *sql.DB
var kv *redis.Client

func (s *server) SubmitCode(ctx context.Context, in *inquiry.CodeRequest) (*inquiry.LoginReply, error) {
	log.Printf("SubmitCode request uid:%d code:%s", in.Head.Uid, in.Code)
	openid, skey, err := weixin.GetInquirySession(in.Code)
	if err != nil {
		log.Printf("SubmitCode GetSession failed:%v", err)
		return &inquiry.LoginReply{
			Head: &common.Head{Retcode: common.ErrCode_ILLEGAL_CODE}}, nil
	}
	sid := util.GenSalt()
	var uid, role int64
	var phone string
	err = db.QueryRow("SELECT u.uid, u.phone, u.role FROM users u, wx_openid x WHERE u.username = x.unionid AND x.openid = ?", openid).
		Scan(&uid, &phone, &role)
	if err != nil {
		_, err = db.Exec("INSERT INTO wx_openid(openid, skey, sid, ctime) VALUES (?, ?, ?, NOW()) ON DUPLICATE KEY UPDATE skey = ?, sid = ?",
			openid, skey, sid, skey, sid)
		if err != nil {
			log.Printf("record failed:%v", err)
			return &inquiry.LoginReply{
				Head: &common.Head{Retcode: 1}}, nil
		}
	}
	if uid == 0 {
		log.Printf("user not found, openid:%s", openid)
		return &inquiry.LoginReply{
			Head: &common.Head{Retcode: 0}, Flag: 0, Sid: sid}, nil
	}

	var hasphone int64
	if phone != "" {
		hasphone = 1
	}
	token := util.GenSalt()
	_, err = db.Exec("UPDATE users SET token = ? WHERE uid = ?", token, uid)
	if err != nil {
		log.Printf("update token failed:%v", err)
		return &inquiry.LoginReply{
			Head: &common.Head{Retcode: 1}}, nil
	}

	return &inquiry.LoginReply{
		Head: &common.Head{Retcode: 0}, Flag: 1, Uid: uid, Token: token,
		Hasphone: hasphone, Role: role}, nil
}

func main() {
	lis, err := net.Listen("tcp", util.InquiryServerPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	db, err = util.InitInquiryDB()
	if err != nil {
		log.Fatalf("failed to init db connection: %v", err)
	}
	db.SetMaxIdleConns(util.MaxIdleConns)

	kv = util.InitRedis()
	go util.ReportHandler(kv, util.InquiryServerName, util.InquiryServerPort)

	s := util.NewGrpcServer()
	inquiry.RegisterInquiryServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
