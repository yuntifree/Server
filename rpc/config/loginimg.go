package main

import (
	"Server/proto/common"
	"Server/proto/config"
	"Server/util"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/net/context"
)

func getLoginImg(db *sql.DB, stype, pos, seq, num int64) []*config.LoginImgInfo {
	query := "SELECT id, img, stime, etime, online FROM login_banner WHERE deleted = 0 "
	query += fmt.Sprintf(" AND type = %d", stype)
	if pos != 0 {
		query += fmt.Sprintf(" AND pos = %d", pos)
	}
	if seq != 0 {
		query += fmt.Sprintf(" AND id < %d", seq)
	}
	query += fmt.Sprintf(" ORDER BY id DESC LIMIT %d", num)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("getLoginImg query failed:%v", err)
		return nil
	}
	defer rows.Close()
	var infos []*config.LoginImgInfo
	for rows.Next() {
		var info config.LoginImgInfo
		err = rows.Scan(&info.Id, &info.Img, &info.Stime, &info.Etime,
			&info.Online)
		if err != nil {
			log.Printf("getLoginImg scan failed:%v", err)
			continue
		}
		infos = append(infos, &info)
	}
	return infos
}

func getTotalLoginImg(db *sql.DB, stype, pos int64) int64 {
	var cnt int64
	query := fmt.Sprintf("SELECT COUNT(id) FROM login_banner WHERE type = %d AND deleted = 0", stype)
	if pos != 0 {
		query += fmt.Sprintf(" AND pos = %d", pos)
	}
	err := db.QueryRow(query).Scan(&cnt)
	if err != nil {
		log.Printf("getTotalLoginImg failed:%v", err)
	}
	return cnt
}

func (s *server) GetLoginImg(ctx context.Context, in *common.CommRequest) (*config.LoginImgReply, error) {
	util.PubRPCRequest(w, "config", "GetLoginImg")
	infos := getLoginImg(db, in.Type, in.Subtype, in.Seq, in.Num)
	var hasmore int64
	if len(infos) >= int(in.Num) {
		hasmore = 1
	}
	total := getTotalLoginImg(db, in.Type, in.Subtype)
	util.PubRPCSuccRsp(w, "config", "GetLoginImg")
	return &config.LoginImgReply{
		Head:  &common.Head{Retcode: 0, Uid: in.Head.Uid},
		Infos: infos, Hasmore: hasmore, Total: total}, nil
}

func addLoginImg(db *sql.DB, info *config.LoginImgInfo) (int64, error) {
	res, err := db.Exec("INSERT INTO login_banner(type, pos, img, stime, etime, ctime) VALUES (?, ?, ?, ?, ?, NOW())",
		info.Type, info.Pos, info.Img, info.Stime, info.Etime)
	if err != nil {
		log.Printf("addLoginImg failed:%+v %v", info, err)
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Printf("addLoginImg get insert id failed:%+v %v", info, err)
		return 0, err
	}
	return id, nil
}

func (s *server) AddLoginImg(ctx context.Context, in *config.LoginImgRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "config", "AddLoginImg")
	id, err := addLoginImg(db, in.Info)
	if err != nil {
		log.Printf("addLoginImg failed:%v", err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	util.PubRPCSuccRsp(w, "config", "AddLoginImg")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid},
		Id:   id}, nil
}

func modLoginImg(db *sql.DB, uid int64, info *config.LoginImgInfo) error {
	_, err := db.Exec("UPDATE login_banner SET img = ?, stime = ?, etime = ?, online = ?, deleted = ? WHERE id = ?",
		info.Img, info.Stime, info.Etime, info.Online, info.Deleted,
		info.Id)
	if err != nil {
		log.Printf("modLoginImg failed:%+v %v", info, err)
		return err
	}

	_, err = db.Exec("INSERT INTO login_banner_history(bid, uid, img, stime, etime, online, deleted, ctime) VALUES (?, ?, ?, ?, ?, ?, ?, NOW())",
		info.Id, uid, info.Img, info.Stime, info.Etime, info.Online,
		info.Deleted)
	if err != nil {
		log.Printf("modLoginImg record failed:%+v %v", info, err)
		return err
	}
	return nil
}

func (s *server) ModLoginImg(ctx context.Context, in *config.LoginImgRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "config", "ModLoginImg")
	err := modLoginImg(db, in.Head.Uid, in.Info)
	if err != nil {
		log.Printf("modLoginImg failed:%v", err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	util.PubRPCSuccRsp(w, "config", "ModLoginImg")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func getOnlineLoginImg(db *sql.DB) []*config.LoginImgInfo {
	rows, err := db.Query("SELECT id, img FROM login_banner WHERE online = 1 AND deleted = 0 AND intranet = 0")
	if err != nil {
		log.Printf("getOnlineLoginImg failed:%v", err)
		return nil
	}
	defer rows.Close()
	var infos []*config.LoginImgInfo
	for rows.Next() {
		var info config.LoginImgInfo
		err = rows.Scan(&info.Id, &info.Img)
		if err != nil {
			log.Printf("getOnlineLoginImg scan failed:%v", err)
			continue
		}
		infos = append(infos, &info)
	}
	return infos
}

func (s *server) GetOnlineLoginImg(ctx context.Context, in *common.CommRequest) (*config.LoginImgReply, error) {
	util.PubRPCRequest(w, "config", "GetOnlineLoginImg")
	infos := getOnlineLoginImg(db)
	util.PubRPCSuccRsp(w, "config", "GetOnlineLoginImg")
	return &config.LoginImgReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Infos: infos}, nil
}

func ackLoginImg(db *sql.DB, id int64) error {
	_, err := db.Exec("UPDATE login_banner SET intranet = 1 WHERE id = ?", id)
	if err != nil {
		return err
	}
	return nil
}

func (s *server) AckLoginImg(ctx context.Context, in *common.CommRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "config", "AckLoginImg")
	err := ackLoginImg(db, in.Id)
	if err != nil {
		log.Printf("AckLoginImg failed:%v", err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	util.PubRPCSuccRsp(w, "config", "AckLoginImg")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}
