package main

import (
	"Server/proto/common"
	"Server/proto/config"
	"Server/util"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/net/context"
)

func getAdBanner(db *sql.DB, mtype, seq, num int64) []*config.AdBannerInfo {
	query := "SELECT id, stype, img, dst, online FROM ad_banner WHERE deleted = 0 "
	query += fmt.Sprintf(" AND type = %d", mtype)
	if seq != 0 {
		query += fmt.Sprintf(" AND id < %d", seq)
	}
	query += fmt.Sprintf(" ORDER BY id DESC LIMIT %d", num)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("getAdBanner query failed:%v", err)
		return nil
	}
	defer rows.Close()
	var infos []*config.AdBannerInfo
	for rows.Next() {
		var info config.AdBannerInfo
		err = rows.Scan(&info.Id, &info.Stype, &info.Img, &info.Dst,
			&info.Online)
		if err != nil {
			log.Printf("getAdBanner scan failed:%v", err)
			continue
		}
		infos = append(infos, &info)
	}
	return infos
}

func getTotalAdBanner(db *sql.DB, mtype int64) int64 {
	var cnt int64
	err := db.QueryRow("SELECT COUNT(id) FROM ad_banner WHERE type = ? AND deleted = 0", mtype).Scan(&cnt)
	if err != nil {
		log.Printf("getTotalAdBanner failed:%v", err)
	}
	return cnt
}

func (s *server) GetAdBanner(ctx context.Context, in *common.CommRequest) (*config.AdBannerReply, error) {
	util.PubRPCRequest(w, "config", "GetAdBanner")
	infos := getAdBanner(db, in.Type, in.Seq, in.Num)
	total := getTotalAdBanner(db, in.Type)
	util.PubRPCSuccRsp(w, "config", "GetAdBanner")
	return &config.AdBannerReply{
		Head:  &common.Head{Retcode: 0, Uid: in.Head.Uid},
		Infos: infos, Total: total}, nil
}

func addAdBanner(db *sql.DB, info *config.AdBannerInfo) (int64, error) {
	res, err := db.Exec("INSERT INTO ad_banner(type, stype, img, dst, ctime) VALUES (?, ?, ?, ?, NOW())",
		info.Type, info.Stype, info.Img, info.Dst)
	if err != nil {
		log.Printf("addAdBanner failed:%+v %v", info, err)
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Printf("addAdBanner get insert id failed:%+v %v", info, err)
		return 0, err
	}
	return id, nil
}

func (s *server) AddAdBanner(ctx context.Context, in *config.AdBannerRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "config", "AddAdBanner")
	id, err := addAdBanner(db, in.Info)
	if err != nil {
		log.Printf("AddAdBanner addAdBanner failed:%+v %v", in, err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid},
		}, nil
	}
	util.PubRPCSuccRsp(w, "config", "AddAdBanner")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid},
		Id:   id}, nil
}

func modAdBanner(db *sql.DB, uid int64, info *config.AdBannerInfo) error {
	_, err := db.Exec("UPDATE ad_banner SET stype = ?, img = ?, dst = ?, online = ?, deleted = ? WHERE id = ?",
		info.Stype, info.Img, info.Dst, info.Online, info.Deleted,
		info.Id)
	if err != nil {
		log.Printf("modAdBanner failed:%+v %v", info, err)
		return err
	}

	_, err = db.Exec("INSERT INTO ad_banner_history(bid, uid, img, dst, online, deleted, ctime) VALUES (?, ?, ?, ?, ?, ?, NOW())",
		info.Id, uid, info.Img, info.Dst, info.Online,
		info.Deleted)
	if err != nil {
		log.Printf("modAdBanner record failed:%+v %v", info, err)
		return err
	}
	return nil
}

func (s *server) ModAdBanner(ctx context.Context, in *config.AdBannerRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "config", "ModAdBanner")
	log.Printf("ModAdBanner in:%+v", in)
	err := modAdBanner(db, in.Head.Uid, in.Info)
	if err != nil {
		log.Printf("ModAdBanner modAdBanner failed:%+v %v", in, err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid},
		}, nil
	}
	util.PubRPCSuccRsp(w, "config", "ModAdBanner")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid},
	}, nil
}

func genViewTable() string {
	now := time.Now()
	return fmt.Sprintf("banner_view_%4d%02d", now.Year(), now.Month())
}

func createViewTable(db *sql.DB, tname string) error {
	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s LIKE banner_view", tname)
	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("crateViewTable %s failed:%v", tname, err)
	}
	return nil
}

func (s *server) ReportAdView(ctx context.Context, in *config.AdViewRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "config", "AdViewRequest")
	table := genViewTable()
	err := createViewTable(db, table)
	if err != nil {
		log.Printf("ReportAdView createViewTable failed:%v", err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid},
		}, nil
	}
	query := fmt.Sprintf("INSERT INTO %s (usermac, apmac, bid, ctime) VALUES(?, ?, ?, NOW())",
		table)
	_, err = db.Exec(query, in.Usermac, in.Apmac, in.Id)
	if err != nil {
		log.Printf("ReportAdView record view failed:%s %s %d %v",
			in.Usermac, in.Apmac, in.Id, err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1},
		}, nil
	}
	_, err = db.Exec(`INSERT INTO banner_view_stat(bid, view_cnt, ctime) 
		VALUES(?, 1, CURDATE()) ON DUPLICATE KEY UPDATE view_cnt = view_cnt + 1`,
		in.Id)
	if err != nil {
		log.Printf("ReportAdView record stat failed:%d %v", in.Id, err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1},
		}, nil
	}
	util.PubRPCSuccRsp(w, "config", "AdViewRequest")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0},
	}, nil
}
