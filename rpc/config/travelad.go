package main

import (
	"Server/proto/common"
	"Server/proto/config"
	"Server/util"
	"database/sql"
	"fmt"
	"log"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/net/context"
)

const (
	redirectURL = "http://wx.yunxingzh.com/redirect"
)

func fetchTravelAd(db *sql.DB, stype, seq, num int64) []*config.TravelAdInfo {
	query := "SELECT id, title, img, dst, stime, etime, online FROM travel_ad WHERE deleted = 0 "
	query += fmt.Sprintf(" AND type = %d", stype)
	if seq != 0 {
		query += fmt.Sprintf(" AND id < %d", seq)
	}
	query += fmt.Sprintf(" ORDER BY id DESC LIMIT %d", num)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("fetchTravelAd query failed:%v", err)
		return nil
	}
	defer rows.Close()
	var infos []*config.TravelAdInfo
	for rows.Next() {
		var info config.TravelAdInfo
		err = rows.Scan(&info.Id, &info.Title, &info.Img, &info.Dst,
			&info.Stime, &info.Etime,
			&info.Online)
		if err != nil {
			log.Printf("fetchTravelAd scan failed:%v", err)
			continue
		}
		infos = append(infos, &info)
	}
	return infos
}

func getTotalTravelAd(db *sql.DB, stype int64) int64 {
	var cnt int64
	err := db.QueryRow("SELECT COUNT(id) FROM travel_ad WHERE type = ? AND deleted = 0",
		stype).Scan(&cnt)
	if err != nil {
		log.Printf("getTotalTravelAd failed:%v", err)
	}
	return cnt
}

func (s *server) FetchTravelAd(ctx context.Context, in *common.CommRequest) (*config.TravelAdReply, error) {
	util.PubRPCRequest(w, "config", "FetchTravelAd")
	infos := fetchTravelAd(db, in.Type, in.Seq, in.Num)
	var hasmore int64
	if len(infos) >= int(in.Num) {
		hasmore = 1
	}
	total := getTotalTravelAd(db, in.Type)
	util.PubRPCSuccRsp(w, "config", "FetchTravelAd")
	return &config.TravelAdReply{
		Head:  &common.Head{Retcode: 0, Uid: in.Head.Uid},
		Infos: infos, Hasmore: hasmore, Total: total}, nil
}

func getMaxRedirectType(db *sql.DB) int64 {
	var max int64
	err := db.QueryRow("SELECT MAX(type) FROM redirect").Scan(&max)
	if err != nil {
		log.Printf("getMaxRedirectType failed:%v", err)
	}
	return max
}

func addTravelAd(db *sql.DB, info *config.TravelAdInfo) (int64, error) {
	max := getMaxRedirectType(db)
	max++
	_, err := db.Exec("INSERT INTO redirect(type, title, dst, ctime) VALUES (?, ?, ?, NOW())", max, info.Title, info.Dst)
	if err != nil {
		log.Printf("addTravelAd record redirect failed:%v", err)
		return 0, err
	}
	dst := fmt.Sprintf("%s?type=%d", redirectURL, max)
	res, err := db.Exec("INSERT INTO travel_ad(type, title, img, dst, stime, etime, redirect_type, ctime) VALUES(?, ?, ?, ?, ?, ?, ?, NOW())",
		info.Type, info.Title, info.Img, dst, info.Stime, info.Etime, max)
	if err != nil {
		log.Printf("addTravelAd insert failed:%v", err)
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Printf("addTravelAd get insert id failed:%v", err)
		return 0, err
	}
	return id, nil
}

func (s *server) AddTravelAd(ctx context.Context, in *config.TravelAdRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "config", "AddTravelAd")
	id, err := addTravelAd(db, in.Info)
	if err != nil {
		log.Printf("addTravelAd failed:%+v %v", in, err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid},
		}, nil
	}
	util.PubRPCSuccRsp(w, "config", "AddTravelAd")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid},
		Id:   id}, nil
}

func modTravelAd(db *sql.DB, info *config.TravelAdInfo) error {
	var dst string
	if strings.HasPrefix(info.Dst, redirectURL) {
		dst = info.Dst
	} else {
		var rtype int64
		err := db.QueryRow("SELECT redirect_type FROM travel_ad WHERE id = ?", info.Id).
			Scan(&rtype)
		if err != nil {
			log.Printf("modTravelAd get redirect type failed:%v", err)
		}
		if rtype == 0 {
			max := getMaxRedirectType(db)
			rtype = max + 1
			_, err := db.Exec("INSERT INTO redirect(type, dst, title, ctime) VALUES (?, ?, ?, NOW())", rtype, info.Dst, info.Title)
			if err != nil {
				log.Printf("modTravelAd add redirect failed:%v", err)
				return err
			}
			_, err = db.Exec("UPDATE travel_ad SET redirect_type = ? WHERE id = ?",
				max, info.Id)
			if err != nil {
				log.Printf("modTravelAd update redirect type failed:%v", err)
				return err
			}
		} else {
			_, err := db.Exec("UPDATE redirect SET dst = ? WHERE type = ?",
				info.Dst, rtype)
			if err != nil {
				log.Printf("modTravelAd update redirect failed:%v", err)
				return err
			}
		}
		dst = fmt.Sprintf("%s?type=%d", redirectURL, rtype)
	}
	_, err := db.Exec(`UPDATE travel_ad Set img = ?, dst = ?, title = ?, 
	stime = ?, etime = ?, online = ?, deleted = ? WHERE id = ?`,
		info.Img, dst, info.Title, info.Stime, info.Etime,
		info.Online, info.Deleted, info.Id)
	if err != nil {
		log.Printf("modTravelAd failed:%v", err)
		return err
	}
	return nil
}

func (s *server) ModTravelAd(ctx context.Context, in *config.TravelAdRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "config", "ModTravelAd")
	err := modTravelAd(db, in.Info)
	if err != nil {
		log.Printf("modTravelAd failed:%+v %v", in, err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid},
		}, nil
	}
	util.PubRPCSuccRsp(w, "config", "ModTravelAd")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid},
	}, nil
}
