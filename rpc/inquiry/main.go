package main

import (
	"Server/proto/common"
	"Server/proto/inquiry"
	"Server/util"
	"Server/weixin"
	"database/sql"
	"encoding/json"
	"errors"
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

func checkSign(skey, rawdata, signature string) bool {
	data := rawdata + skey
	sign := util.Sha1(data)
	return sign == signature
}

func extractUserInfo(skey, encrypted, iv string) (weixin.UserInfo, error) {
	var uinfo weixin.UserInfo
	dst, err := weixin.DecryptData(skey, encrypted, iv)
	if err != nil {
		log.Printf("aes decrypt failed skey:%s", skey)
		return uinfo, err
	}
	err = json.Unmarshal(dst, &uinfo)
	if err != nil {
		log.Printf("parse json failed:%s", string(dst))
		return uinfo, err
	}
	if uinfo.UnionId == "" {
		log.Printf("get unionid failed:%v", err)
		return uinfo, errors.New("get unionid failed")
	}
	return uinfo, nil
}

func (s *server) Login(ctx context.Context, in *inquiry.LoginRequest) (*inquiry.LoginReply, error) {
	log.Printf("Login request:%v", in)
	var skey, unionid, openid string
	err := db.QueryRow("SELECT skey, unionid, openid FROM wx_openid WHERE sid = ?", in.Sid).
		Scan(&skey, &unionid, &openid)
	if err != nil {
		log.Printf("illegal sid:%s", in.Sid)
		return &inquiry.LoginReply{
			Head: &common.Head{Retcode: 1}}, nil
	}
	if !checkSign(skey, in.Rawdata, in.Signature) {
		log.Printf("check signature failed sid:%s", in.Sid)
		return &inquiry.LoginReply{
			Head: &common.Head{Retcode: 1}}, nil
	}
	var uid, role int64
	var phone string
	if unionid == "" { //has login
		uinfo, err := extractUserInfo(skey, in.Encrypteddata, in.Iv)
		if err != nil {
			log.Printf("extract user info failed sid:%s", in.Sid)
			return &inquiry.LoginReply{
				Head: &common.Head{Retcode: 1}}, nil
		}
		_, err = db.Exec("UPDATE wx_openid SET unionid = ? WHERE openid = ?",
			uinfo.UnionId, openid)
		if err != nil {
			log.Printf("update unionid failed:%v", err)
			return &inquiry.LoginReply{
				Head: &common.Head{Retcode: 1}}, nil
		}
		db.QueryRow("SELECT uid, phone, role FROM users WHERE username = ?",
			uinfo.UnionId).
			Scan(&uid, &phone, &role)
		if uid == 0 {
			res, err := db.Exec("INSERT IGNORE INTO user(username, nickname, headurl, gender, ctime) VALUES (?, ?, ?, ?, NOW())",
				uinfo.UnionId, uinfo.NickName, uinfo.AvartarUrl, uinfo.Gender)
			if err != nil {
				log.Printf("create user failed:%v", err)
				return &inquiry.LoginReply{
					Head: &common.Head{Retcode: 1}}, nil
			}
			uid, _ = res.LastInsertId()
		}
	} else {
		db.QueryRow("SELECT uid FROM user WHERE username = ?", unionid).Scan(&uid)
	}
	if uid == 0 {
		log.Printf("select user failed sid:%s", in.Sid)
		return &inquiry.LoginReply{
			Head: &common.Head{Retcode: 1}}, nil
	}
	token := util.GenSalt()
	_, err = db.Exec("UPDATE users SET token = ? WHERE uid = ?", token)
	if err != nil {
		log.Printf("update token failed sid:%s", in.Sid)
		return &inquiry.LoginReply{
			Head: &common.Head{Retcode: 1}}, nil
	}

	return &inquiry.LoginReply{
		Head: &common.Head{Retcode: 0}, Uid: uid, Token: token}, nil
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
