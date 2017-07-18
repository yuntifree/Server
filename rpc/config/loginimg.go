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

func getLoginImg(db *sql.DB, stype, seq, num int64) []*config.LoginImgInfo {
	query := "SELECT id, img, stime, etime, online FROM login_banner WHERE deleted = 0 "
	query += fmt.Sprintf(" AND type = %d", stype)
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

func (s *server) GetLoginImg(ctx context.Context, in *common.CommRequest) (*config.LoginImgReply, error) {
	util.PubRPCRequest(w, "config", "GetLoginImg")
	infos := getLoginImg(db, in.Type, in.Seq, in.Num)
	var hasmore int64
	if len(infos) >= int(in.Num) {
		hasmore = 1
	}
	util.PubRPCSuccRsp(w, "config", "GetLoginImg")
	return &config.LoginImgReply{
		Head:  &common.Head{Retcode: 0, Uid: in.Head.Uid},
		Infos: infos, Hasmore: hasmore}, nil
}
