package main

import (
	"Server/proto/common"
	"Server/proto/inquiry"
	"Server/util"
	"database/sql"
	"fmt"
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

type chatInfo struct {
	cid     int64
	ctype   int64
	content string
	ctime   string
	ack     int64
}

func getLastChat(db *sql.DB, doctor, patient int64) (*chatInfo, error) {
	var info chatInfo
	err := db.QueryRow("SELECT id, type, content, ctime, ack FROM chat WHERE uid = ? AND tuid = ? ORDER BY id DESC LIMIT 1", patient, doctor).
		Scan(&info.cid, &info.ctype, &info.content, &info.ctime,
			&info.ack)
	if err != nil {
		log.Printf("getLastChat failed:%d %d %v", doctor, patient, err)
		return nil, err
	}
	return &info, nil
}

func getUserChatSession(db *sql.DB, doctor, seq, num int64) []*inquiry.ChatSessionInfo {
	query := fmt.Sprintf("SELECT r.id, r.patient, u.headurl, u.nickname FROM relations r, users u WHERE r.doctor = %d AND r.flag = 1 AND r.deleted = 0 AND u.deleted = 0", doctor)
	if seq != 0 {
		query += fmt.Sprintf(" AND r.id < %d", seq)
	}
	query += fmt.Sprintf(" ORDER BY r.id DESC LIMIT %d", num)
	log.Printf("getUserChatSession query:%s", query)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("getUserChatSession query failed:%v", err)
		return nil
	}

	var infos []*inquiry.ChatSessionInfo
	defer rows.Close()
	for rows.Next() {
		var info inquiry.ChatSessionInfo
		err = rows.Scan(&info.Id, &info.Uid, &info.Headurl,
			&info.Nickname)
		if err != nil {
			log.Printf("getUserChatSession scan failed:%v", err)
			continue
		}
		cinfo, err := getLastChat(db, doctor, info.Uid)
		if err != nil {
			log.Printf("getUserChatSession getLastChat failed:%v", err)
		} else {
			info.Cid = cinfo.cid
			info.Type = cinfo.ctype
			info.Content = cinfo.content
			info.Ctime = cinfo.ctime
			if cinfo.ack == 0 {
				info.Reddot = 1
			}
		}

		infos = append(infos, &info)
	}
	return infos
}

func (s *server) GetChatSession(ctx context.Context, in *common.CommRequest) (*inquiry.ChatSessionReply, error) {
	util.PubRPCRequest(w, "inquiry", "GetChatSession")
	infos := getUserChatSession(db, in.Head.Uid, in.Seq, in.Num)
	util.PubRPCSuccRsp(w, "inquiry", "GetChatSession")
	var hasmore int64
	if len(infos) >= int(in.Num) {
		hasmore = 1
	}
	return &inquiry.ChatSessionReply{Head: &common.Head{Retcode: 0},
		Infos: infos, Hasmore: hasmore}, nil
}
