package main

import (
	"Server/proto/common"
	"Server/proto/inquiry"
	"Server/util"
	"database/sql"
	"log"

	"golang.org/x/net/context"
)

func addInquiry(db *sql.DB, fee int64, in *inquiry.InquiryRequest) (int64, error) {
	res, err := db.Exec("INSERT INTO inquiry_history(doctor, patient, pid, fee, doctor_fee, ctime) VALUES (?, ?, ?, ?, ?, NOW())",
		in.Doctor, in.Head.Uid, in.Pid, in.Fee, fee)
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

func getDoctorFee(db *sql.DB, uid int64) (int64, error) {
	var fee int64
	err := db.QueryRow("SELECT d.fee FROM doctor d, users u WHERE d.id = u.doctor AND u.uid = ?", uid).Scan(&fee)
	if err != nil {
		log.Printf("getDoctorFee query failed:%v", err)
		return 0, err
	}
	return fee, nil
}

func (s *server) AddInquiry(ctx context.Context, in *inquiry.InquiryRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "inquiry", "AddInquiry")
	fee, err := getDoctorFee(db, in.Doctor)
	if err != nil {
		log.Printf("AddInquiry getDoctorFee failed:%v", err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	id, err := addInquiry(db, fee, in)
	if err != nil {
		log.Printf("addInquiry failed:%d %v", in.Head.Uid, err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	util.PubRPCSuccRsp(w, "inquiry", "ModPatient")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Id: id}, nil
}
