package main

import (
	"Server/proto/common"
	"Server/proto/inquiry"
	"Server/util"
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
