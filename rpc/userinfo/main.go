package main

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"net"

	"Server/proto/common"
	"Server/proto/userinfo"
	"Server/util"

	_ "github.com/go-sql-driver/mysql"
	nsq "github.com/nsqio/go-nsq"
	"golang.org/x/net/context"
)

const (
	femaleType = 0
	maleType   = 1
	saveRate   = 0.03 / (1024.0 * 1024.0)
	signScore  = 50
)

type server struct{}

var db *sql.DB
var w *nsq.Producer

func genUserTip(traffic int64) string {
	traffMb := traffic / (8 * 1024 * 1024)
	save := int64(float64(traffic) * saveRate / 8)
	return fmt.Sprintf("您已节省流量%dM，话费%d元", traffMb, save)
}

func (s *server) GetInfo(ctx context.Context, in *common.CommRequest) (*userinfo.InfoReply, error) {
	util.PubRPCRequest(w, "userinfo", "GetInfo")
	var headurl, nickname string
	var total, save, score int64
	err := db.QueryRow("SELECT headurl, nickname, times, traffic, score FROM user WHERE uid = ?", in.Head.Uid).Scan(&headurl, &nickname, &total, &save, &score)
	if err != nil {
		log.Printf("GetInfo query failed:%v", err)
		return &userinfo.InfoReply{
			Head: &common.Head{Retcode: 1}}, nil
	}
	tip := genUserTip(save)
	save = int64(float64(save) * saveRate)
	util.PubRPCSuccRsp(w, "userinfo", "GetInfo")
	nick, err := base64.StdEncoding.DecodeString(nickname)
	if err != nil {
		log.Printf("GetInfo decode nick failed:%v", err)
	}
	return &userinfo.InfoReply{
		Head: &common.Head{Retcode: 0}, Headurl: headurl, Nickname: string(nick),
		Total: total, Save: save, Tip: tip, Score: score}, nil
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

func getNickMinMax(db *sql.DB) (int, int) {
	min, max := 10, 20
	err := db.QueryRow("SELECT MIN(id), MAX(id) FROM nickname").Scan(&min, &max)
	if err != nil {
		log.Printf("getTotalNick failed:%v", err)
	}
	return min, max
}

func getRandNick(db *sql.DB, uid int64) []string {
	var names []string
	min, max := getNickMinMax(db)
	idx := util.Randn(int32(max - min))
	rows, err := db.Query("SELECT name FROM nickname WHERE id > ? ORDER BY id LIMIT 10",
		idx+int32(min))
	if err != nil {
		log.Printf("getRandNick failed:%v", err)
		return names
	}
	defer rows.Close()
	for rows.Next() {
		var nick string
		err := rows.Scan(&nick)
		if err != nil {
			log.Printf("getRandNick scan failed:%v", err)
			continue
		}
		names = append(names, nick)
	}
	return names
}

func (s *server) GenRandNick(ctx context.Context, in *common.CommRequest) (*userinfo.NickReply, error) {
	util.PubRPCRequest(w, "userinfo", "GetRandNick")
	nicks := getRandNick(db, in.Head.Uid)
	util.PubRPCSuccRsp(w, "userinfo", "GetRandNick")
	return &userinfo.NickReply{
		Head: &common.Head{Retcode: 0}, Nicknames: nicks}, nil
}

func (s *server) ModInfo(ctx context.Context, in *userinfo.InfoRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "userinfo", "ModInfo")
	query := "UPDATE user SET atime = NOW() "
	if in.Headurl != "" {
		query += ", headurl = '" + in.Headurl + "' "
	}
	if in.Nickname != "" {
		query += ", nickname = '" + base64.StdEncoding.EncodeToString([]byte(in.Nickname)) + "' "
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

func getUserScore(db *sql.DB, uid int64) int64 {
	var score int64
	err := db.QueryRow("SELECT score FROM user WHERE uid = ?", uid).
		Scan(&score)
	if err != nil {
		log.Printf("getUserScore query failed:%v", err)
	}
	return score
}

func hasSign(db *sql.DB, uid int64) int64 {
	var cnt int64
	err := db.QueryRow("SELECT COUNT(id) FROM signin_history WHERE ctime >= CURDATE() AND uid = ?", uid).Scan(&cnt)
	if err != nil {
		log.Printf("hasSign query failed:%v", err)
	}
	if cnt > 0 {
		return 1
	}
	return 0
}

func getScoreItems(db *sql.DB, uid int64) []*userinfo.ScoreItem {
	rows, err := db.Query("SELECT i.id, i.img, i.score, IFNULL(u.status,0) FROM score_item i LEFT JOIN user_score_item u ON i.id = u.item AND u.uid = ? AND i.deleted = 0", uid)
	if err != nil {
		log.Printf("getScoreItems query failed:%v", err)
		return nil
	}
	defer rows.Close()
	var items []*userinfo.ScoreItem
	for rows.Next() {
		var item userinfo.ScoreItem
		err = rows.Scan(&item.Id, &item.Img, &item.Score, &item.Status)
		if err != nil {
			log.Printf("getScoreItems scan failed:%v", err)
			continue
		}
		items = append(items, &item)
	}
	return items
}

func (s *server) GetUserScore(ctx context.Context, in *common.CommRequest) (*userinfo.ScoreReply, error) {
	util.PubRPCRequest(w, "userinfo", "GetUserScore")
	score := getUserScore(db, in.Head.Uid)
	sign := hasSign(db, in.Head.Uid)
	items := getScoreItems(db, in.Head.Uid)
	util.PubRPCSuccRsp(w, "userinfo", "GetUserScore")
	return &userinfo.ScoreReply{
		Head: &common.Head{Retcode: 0}, Score: score, Sign: sign,
		Items: items}, nil
}

func (s *server) DailySign(ctx context.Context, in *common.CommRequest) (*userinfo.ScoreReply, error) {
	util.PubRPCRequest(w, "userinfo", "DailySign")
	res, err := db.Exec("INSERT IGNORE INTO signin_history(uid, ctime) VALUES (?, CURDATE())", in.Head.Uid)
	if err != nil {
		log.Printf("DailySign insert failed:%v", err)
		return &userinfo.ScoreReply{
			Head: &common.Head{Retcode: 1}}, nil
	}
	cnt, err := res.RowsAffected()
	if err != nil {
		log.Printf("DailySign get rows affected failed:%v", err)
		return &userinfo.ScoreReply{
			Head: &common.Head{Retcode: 1}}, nil
	}
	if cnt == 0 {
		log.Printf("has signed:%d", in.Head.Uid)
		return &userinfo.ScoreReply{
			Head: &common.Head{Retcode: common.ErrCode_HAS_SIGN}}, nil

	}
	incrUserScore(db, in.Head.Uid, signScore)
	score := getUserScore(db, in.Head.Uid)
	util.PubRPCSuccRsp(w, "userinfo", "DailySign")
	return &userinfo.ScoreReply{
		Head: &common.Head{Retcode: 0}, Score: score}, nil
}

func incrUserScore(db *sql.DB, uid, score int64) {
	_, err := db.Exec("UPDATE user SET score = score + ? WHERE uid = ?",
		score, uid)
	if err != nil {
		log.Printf("incrUserScore failed:%d %d %v", uid, score, err)
	}
}

func descUserScore(db *sql.DB, uid, score int64) {
	_, err := db.Exec("UPDATE user SET score = IF(score > ?, score - ?, 0) WHERE uid = ?",
		score, score, uid)
	if err != nil {
		log.Printf("descUserScore failed:%d %d %v", uid, score, err)
	}
}

func getItemScore(db *sql.DB, id int64) int64 {
	var score int64
	err := db.QueryRow("SELECT score FROM score_item WHERE id = ?", id).Scan(&score)
	if err != nil {
		log.Printf("getItemScore failed:%d %v", id, err)
	}
	return score
}

func recordExchange(db *sql.DB, uid, item, num, score int64) {
	_, err := db.Exec(`INSERT INTO exchange_history(uid, item, num, score, ctime) 
	VALUES (?, ?, ?, ?, NOW())`,
		uid, item, num, score)
	if err != nil {
		log.Printf("recordExchange insert history failed:%d %d %d %v",
			uid, item, num, err)
		return
	}
	_, err = db.Exec(`INSERT INTO user_score_item(uid, item, total, status)
	VALUES (?, ?, ?, 1) ON DUPLICATE KEY UPDATE total = total + ?`,
		uid, item, num, num)
	if err != nil {
		log.Printf("recordExchange insert user_score_item failed:%d %d %d %v",
			uid, item, num, err)
		return

	}
	return
}

func hasExchange(db *sql.DB, uid, item int64) bool {
	var total int64
	err := db.QueryRow("SELECT total FROM user_score_item WHERE uid = ? AND item = ?",
		uid, item).Scan(&total)
	if err != nil {
		log.Printf("hasExchange query failed:%v", err)
	}
	return total > 0
}

func (s *server) ExchangeScore(ctx context.Context, in *common.CommRequest) (*userinfo.ScoreReply, error) {
	util.PubRPCRequest(w, "userinfo", "ExchangeScore")
	if hasExchange(db, in.Head.Uid, in.Id) {
		log.Printf("has exchange:%d %d", in.Head.Uid, in.Id)
		return &userinfo.ScoreReply{
			Head: &common.Head{Retcode: common.ErrCode_HAS_EXCHANGE}}, nil
	}

	itemScore := getItemScore(db, in.Id)
	score := getUserScore(db, in.Head.Uid)
	if score < itemScore*in.Num {
		log.Printf("not enough score:%d %d %d", in.Head.Uid, score, in.Num)
		return &userinfo.ScoreReply{
			Head: &common.Head{Retcode: common.ErrCode_INSUFFICIENT_SCORE}}, nil
	}
	log.Printf("descUserScore uid:%d score:%d", in.Head.Uid, itemScore*in.Num)
	descUserScore(db, in.Head.Uid, itemScore*in.Num)
	recordExchange(db, in.Head.Uid, in.Id, in.Num, itemScore*in.Num)
	score = getUserScore(db, in.Head.Uid)
	util.PubRPCSuccRsp(w, "userinfo", "ExchangeScore")
	return &userinfo.ScoreReply{
		Head: &common.Head{Retcode: 0}, Score: score}, nil
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

	s := util.NewGrpcServer()
	userinfo.RegisterUserinfoServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
