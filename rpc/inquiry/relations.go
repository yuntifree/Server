package main

import (
	"Server/proto/common"
	"Server/util"
	"log"

	"golang.org/x/net/context"
)

func (s *server) BindOp(ctx context.Context, in *common.CommRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "inquiry", "BindOp")
	var err error
	if in.Type == 0 {
		_, err = db.Exec("INSERT INTO relations(doctor, patient, ctime) VALUES (?, ?, NOW()) ON DUPLICATE KEY UPDATE deleted = 0",
			in.Id, in.Head.Uid)
	} else {
		_, err = db.Exec("UPDATE relations SET deleted = 1 WHERE doctor = ? AND patient = ?",
			in.Id, in.Head.Uid)
	}
	if err != nil {
		log.Printf("BindOp failed:%d %d %v", in.Head.Uid, in.Id)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1}}, nil

	}
	if in.Type == 0 {
		_, err = db.Exec("UPDATE users SET hasrelation = 1 WHERE uid = ?", in.Head.Uid)
		if err != nil {
			log.Printf("BindOp update user relation failed:%d %v", in.Head.Uid,
				err)
		}
	}
	util.PubRPCSuccRsp(w, "inquiry", "BindOp")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0}}, nil
}
