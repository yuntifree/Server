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
	query := "SELECT a.id, a.longitude, a.latitude, a.unid, u.name FROM ap_info a, unit u WHERE a.unid = u.id AND a.deleted = 0 AND u.deleted = 0"
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
		err = rows.Scan(&info.Id, &info.Longitude, &info.Latitude,
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
