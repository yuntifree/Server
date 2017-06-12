package main

import (
	"Server/proto/common"
	"Server/proto/inquiry"
	"Server/util"
	"database/sql"
	"log"

	"golang.org/x/net/context"
)

func addInquiry(db *sql.DB, in *inquiry.InquiryRequest) (int64, error) {
	res, err := db.Exec("INSERT INTO inquiry_history(doctor, patient, pid, fee, ctime) VALUES (?, ?, ?, ?, NOW())", in.Doctor, in.Head.Uid, in.Pid, in.Fee)
	if err != nil {
		log.Printf("addInquiry failed:%v", err)
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Printf("addInquiry get insert id failed:%v", err)
		return 0, err
	}
	_, err = db.Exec("UPDATE relations SET hid = ?, status = 0 WHERE doctor = ? AND patient = ?", id, in.Doctor, in.Head.Uid)
	if err != nil {
		log.Printf("addInquiry upate relation failed:%v", err)
		return 0, err
	}
	return id, nil
}

func (s *server) AddInquiry(ctx context.Context, in *inquiry.InquiryRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "inquiry", "AddInquiry")
	id, err := addInquiry(db, in)
	if err != nil {
		log.Printf("addInquiry failed:%d %v", in.Head.Uid, err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	util.PubRPCSuccRsp(w, "inquiry", "ModPatient")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Id: id}, nil
}
