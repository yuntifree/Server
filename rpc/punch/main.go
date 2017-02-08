package main

import (
	"database/sql"
	"log"
	"net"

	redis "gopkg.in/redis.v5"

	"Server/proto/common"
	"Server/proto/punch"
	"Server/util"
	"Server/weixin"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type server struct{}

var db *sql.DB

var kv *redis.Client

func getApID(db *sql.DB, apmac string) int64 {
	var aid int64
	err := db.QueryRow("SELECT id FROM ap WHERE mac = ?", apmac).Scan(&aid)
	if err != nil {
		log.Printf("getApID failed:%v", err)
	}
	return aid
}

func (s *server) Punch(ctx context.Context, in *punch.PunchRequest) (*common.CommReply, error) {
	log.Printf("punch request uid:%d apmac:%s", in.Head.Uid, in.Apmac)
	aid := getApID(db, in.Apmac)
	if aid == 0 {
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
	aid := getApID(db, in.Apmac)
	if aid == 0 {
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	_, err := db.Exec("INSERT IGNORE INTO punch_praise(aid, uid, ctime) VALUES (?, ?, NOW())",
		aid, in.Head.Uid)
	if err != nil {
		log.Printf("Punch insert record failed:%v", err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func getPraiseTotal(db *sql.DB, aid int64) int64 {
	var total int64
	err := db.QueryRow("SELECT COUNT(id) FROM punch_praise WHERE aid = ?", aid).Scan(&total)
	if err != nil {
		log.Printf("getPraiseTotal failed:%v", err)
	}
	return total
}

func getPunch(db *sql.DB, uid int64) []*punch.PunchInfo {
	var infos []*punch.PunchInfo
	rows, err := db.Query("SELECT a.id, longitude, latitude, address, p.ctime FROM punch p, ap a WHERE p.aid = a.id AND p.uid = ?", uid)
	if err != nil {
		log.Printf("getPunch query failed:%v", err)
		return infos
	}

	defer rows.Close()
	for rows.Next() {
		var info punch.PunchInfo
		err := rows.Scan(&info.Aid, &info.Longitude, &info.Latitude, &info.Address,
			&info.Time)
		if err != nil {
			log.Printf("getPunch scan failed:%v", err)
			continue
		}
		info.Total = getPraiseTotal(db, info.Aid)
		infos = append(infos, &info)
	}
	return infos
}

func (s *server) GetPunch(ctx context.Context, in *common.CommRequest) (*punch.PunchReply, error) {
	log.Printf("GetPunch request uid:%d", in.Head.Uid)
	infos := getPunch(db, in.Head.Uid)
	return &punch.PunchReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Infos: infos}, nil
}

func getApPunch(db *sql.DB, aid int64) punch.PunchUserInfo {
	var info punch.PunchUserInfo
	err := db.QueryRow("SELECT u.uid, u.nickname, u.headurl, p.ctime FROM user u, punch p WHERE p.uid = u.uid AND p.aid = ?", aid).
		Scan(&info.Uid, &info.Nickname, &info.Headurl, &info.Time)
	if err != nil {
		log.Printf("getApPunch failed:%v", err)
	}
	return info
}

func getApPraise(db *sql.DB, aid int64) *punch.PraiseInfo {
	var praise punch.PraiseInfo
	err := db.QueryRow("SELECT COUNT(id) FROM punch_praise WHERE aid = ?", aid).Scan(&praise.Total)
	if err != nil {
		log.Printf("getApPraise get total failed:%v", err)
		return &praise
	}

	rows, err := db.Query("SELECT nickname FROM user u, punch_praise p WHERE p.uid = u.uid AND p.aid = ?", aid)
	if err != nil {
		log.Printf("getApPraise query nickname failed:%v", err)
		return &praise
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		err := rows.Scan(&name)
		if err != nil {
			log.Printf("getApPraise scan nickname failed:%v", err)
			continue
		}
		names = append(names, name)
	}
	praise.Nicknames = names
	return &praise
}

func (s *server) GetStat(ctx context.Context, in *punch.PunchRequest) (*punch.PunchStatReply, error) {
	log.Printf("GetStat request uid:%d apmac:%s", in.Head.Uid, in.Apmac)
	aid := getApID(db, in.Apmac)
	if aid == 0 {
		return &punch.PunchStatReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	info := getApPunch(db, aid)
	if info.Uid == 0 {
		return &punch.PunchStatReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}, Pflag: 0}, nil
	}
	praise := getApPraise(db, aid)
	return &punch.PunchStatReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Pflag: 1,
		Info: &info, Praise: praise}, nil
}

func (s *server) SubmitCode(ctx context.Context, in *punch.CodeRequest) (*punch.LoginReply, error) {
	log.Printf("SubmitCode request uid:%d code:%s", in.Head.Uid, in.Code)
	openid, skey, err := weixin.GetSession(in.Code)
	if err != nil {
		log.Printf("SubmitCode GetSession failed:%v", err)
		return &punch.LoginReply{
			Head: &common.Head{Retcode: common.ErrCode_ILLEGAL_CODE}}, nil
	}
	var uid int64
	err = db.QueryRow("SELECT uid FROM user u, xcx_openid x WHERE u.username = x.unionid AND x.openid = ?", openid).Scan(&uid)
	if err != nil {
		_, err = db.Exec("INSERT INTO xcx_openid(openid, skey, ctime) VALUES (?, ?, NOW()) ON DUPLICATE KEY UPDATE skey = ?", openid, skey, skey)
		if err != nil {
			log.Printf("record failed:%v", err)
			return &punch.LoginReply{
				Head: &common.Head{Retcode: 1}}, nil
		}
	}
	if uid == 0 {
		log.Printf("user not found, openid:%s", openid)
		return &punch.LoginReply{
			Head: &common.Head{Retcode: 0}, Flag: 0}, nil
	}

	token := util.GenSalt()
	privdata := util.GenSalt()
	_, err = db.Exec("UPDATE user SET token = ?, private = ? WHERE uid = ?", token, privdata, err)
	if err != nil {
		log.Printf("SubmitCode update token failed:%v", err)
		return &punch.LoginReply{
			Head: &common.Head{Retcode: 1}}, nil
	}
	util.SetCachedToken(kv, uid, token)
	return &punch.LoginReply{
		Head: &common.Head{Retcode: 0}, Flag: 1, Uid: uid, Token: token}, nil
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
