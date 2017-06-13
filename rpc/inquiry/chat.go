package main

import (
	"Server/proto/common"
	"Server/proto/inquiry"
	"Server/util"
	"database/sql"
	"log"

	"golang.org/x/net/context"
)

func (s *server) SendChat(ctx context.Context, in *inquiry.ChatRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "inquiry", "SendChat")
	res, err := db.Exec("INSERT INTO chat(uid, tuid, type, content, ctime) VALUES (?, ?, ?, ?, NOW())",
		in.Head.Uid, in.Tuid, in.Type, in.Content)
	if err != nil {
		log.Printf("SendChat insert failed:%d %d %v", in.Head.Uid,
			in.Tuid, err)
		return &common.CommReply{Head: &common.Head{Retcode: 1}}, nil
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Printf("SendChat get insert id failed:%d %d %v", in.Head.Uid,
			in.Tuid, err)
		return &common.CommReply{Head: &common.Head{Retcode: 1}}, nil
	}
	util.PubRPCSuccRsp(w, "inquiry", "SendChat")
	return &common.CommReply{Head: &common.Head{Retcode: 0},
		Id: id}, nil
}

func getUserChat(db *sql.DB, uid, tuid, seq, num int64) []*inquiry.ChatInfo {
	rows, err := db.Query("SELECT id, uid, tuid, type, content, ctime FROM chat WHERE ((uid = ? AND tuid = ?) OR (uid = ? AND tuid = ?)) AND seq > ? ORDER BY id ASC LIMIT ?",
		uid, tuid, tuid, uid, seq, num)
	if err != nil {
		log.Printf("getUserChat query failed:%d %d %v", uid, tuid, err)
		return nil
	}
	var infos []*inquiry.ChatInfo
	var maxseq int64
	defer rows.Close()
	for rows.Next() {
		var info inquiry.ChatInfo
		err = rows.Scan(&info.Id, &info.Uid, &info.Tuid, &info.Type, &info.Content, &info.Ctime)
		if err != nil {
			log.Printf("getUserChat scan failed:%d %d %v", uid, tuid, err)
			continue
		}
		info.Seq = info.Id
		maxseq = info.Seq
		infos = append(infos, &info)
	}
	_, err = db.Exec("UPDATE chat SET ack = 1, acktime = NOW() WHERE uid = ? AND tuid = ? AND id <= ? AND ack = 0",
		tuid, uid, maxseq)
	if err != nil {
		log.Printf("getUserChat update ack failed:%v", err)
	}
	return infos
}

func (s *server) GetChat(ctx context.Context, in *common.CommRequest) (*inquiry.ChatReply, error) {
	util.PubRPCRequest(w, "inquiry", "GetChat")
	infos := getUserChat(db, in.Head.Uid, in.Id, in.Seq, in.Num)
	util.PubRPCSuccRsp(w, "inquiry", "GetChat")
	return &inquiry.ChatReply{Head: &common.Head{Retcode: 0},
		Infos: infos}, nil
}
