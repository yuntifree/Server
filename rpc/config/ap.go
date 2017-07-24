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

func getApTotal(db *sql.DB, search string) int64 {
	query := "SELECT COUNT(id) FROM ap_info WHERE deleted = 0"
	if search != "" {
		query += fmt.Sprintf(" AND mac = '%s'", search)
	}
	var cnt int64
	err := db.QueryRow(query).Scan(&cnt)
	if err != nil {
		log.Printf("getApTotal query failed:%v", err)
	}
	return cnt
}

func getApInfo(db *sql.DB, seq, num int64, search string) []*config.ApInfo {
	query := "SELECT a.id, a.mac, a.longitude, a.latitude, a.unid, u.name FROM ap_info a, unit u WHERE a.unid = u.id AND a.deleted = 0 AND u.deleted = 0"
	if search != "" {
		query += fmt.Sprintf(" AND a.mac = '%s'", search)
	}
	if seq != 0 {
		query += fmt.Sprintf(" AND a.id < %d", seq)
	}
	query += fmt.Sprintf(" ORDER BY a.id DESC LIMIT %d", num)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("getApInfo failed:%v", err)
		return nil
	}
	var infos []*config.ApInfo
	defer rows.Close()
	for rows.Next() {
		var info config.ApInfo
		err = rows.Scan(&info.Id, &info.Mac, &info.Longitude, &info.Latitude,
			&info.Unid, &info.Name)
		if err != nil {
			log.Printf("getApInfo scan failed:%v", err)
			continue
		}
		infos = append(infos, &info)
	}
	return infos
}

func (s *server) GetApInfo(ctx context.Context, in *common.CommRequest) (*config.ApInfoReply, error) {
	util.PubRPCRequest(w, "config", "GetApInfo")
	infos := getApInfo(db, in.Seq, in.Num, in.Search)
	total := getApTotal(db, in.Search)
	util.PubRPCSuccRsp(w, "config", "GetApInfo")
	return &config.ApInfoReply{
		Head:  &common.Head{Retcode: 0, Uid: in.Head.Uid},
		Infos: infos, Total: total}, nil
}

func addApInfo(db *sql.DB, info *config.ApInfo) (int64, error) {
	res, err := db.Exec("INSERT INTO ap_info(mac, longitude, latitude, unid) VALUES (?, ?, ?, ?)",
		info.Mac, info.Longitude, info.Latitude, info.Unid)
	if err != nil {
		log.Printf("addApInfo insert failed:%v", err)
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Printf("addApInfo get insert id failed:%v", err)
		return 0, err
	}
	return id, nil
}

func (s *server) AddApInfo(ctx context.Context, in *config.ApInfoRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "config", "AddApInfo")
	id, err := addApInfo(db, in.Info)
	if err != nil {
		log.Printf("AddApInfo addApInfo failed:%v", err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid},
			Id:   id}, nil
	}
	util.PubRPCSuccRsp(w, "config", "AddApInfo")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid},
		Id:   id}, nil
}

func modApInfo(db *sql.DB, info *config.ApInfo) error {
	_, err := db.Exec("UPDATE ap_info SET longitude = ?, latitude = ?, unid = ?, deleted = ? WHERE id = ?",
		info.Longitude, info.Latitude, info.Unid, info.Id, info.Deleted)
	if err != nil {
		log.Printf("addApInfo insert failed:%v", err)
		return err
	}
	return nil
}

func (s *server) ModApInfo(ctx context.Context, in *config.ApInfoRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "config", "ModApInfo")
	err := modApInfo(db, in.Info)
	if err != nil {
		log.Printf("ModApInfo modApInfo failed:%v", err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid},
		}, nil
	}
	util.PubRPCSuccRsp(w, "config", "ModApInfo")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid},
	}, nil
}
