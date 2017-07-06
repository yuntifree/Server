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
	res, err := db.Exec("INSERT INTO inquiry_history(doctor, patient, pid, fee, doctor_fee, form_id, ctime) VALUES (?, ?, ?, ?, ?, ?, NOW())",
		in.Doctor, in.Head.Uid, in.Pid, in.Fee, fee, in.Formid)
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

func finInquiry(db *sql.DB, uid, tuid int64) error {
	var id, hid, status int64
	err := db.QueryRow("SELECT id, hid, status FROM relations WHERE doctor = ? AND patient = ?", uid, tuid).Scan(&id, &hid, &status)
	if err != nil {
		log.Printf("finInquiry query failed:%d %d %v", uid, tuid, err)
		return err
	}
	if status == 2 {
		log.Printf("finInquiry finish closed inquiry:%d %d %d", uid, tuid, hid)
		return nil
	}
	_, err = db.Exec("UPDATE relations SET status = 2 WHERE id = ?", id)
	if err != nil {
		log.Printf("finInquiry update relations failed:%d %d %v", uid,
			tuid, err)
		return err
	}
	_, err = db.Exec("UPDATE inquiry_history SET status = 2, etime = NOW() WHERE id = ?", hid)
	if err != nil {
		log.Printf("finInquiry update inquiry history failed:%d %d %v", uid,
			tuid, err)
		return err
	}
	return nil
}

func (s *server) FinInquiry(ctx context.Context, in *inquiry.FinInquiryRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "inquiry", "FinInquiry")
	var role int64
	err := db.QueryRow("SELECT role FROM users WHERE uid = ?", in.Head.Uid).
		Scan(&role)
	if err != nil {
		log.Printf("FinInquiry query role failed:%d %v", in.Head.Uid, err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	if role != 1 {
		log.Printf("FinInquiry illegal role:%d %d", in.Head.Uid, role)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	err = finInquiry(db, in.Head.Uid, in.Tuid)
	if err != nil {
		log.Printf("finInquiry failed:%d %v", in.Head.Uid, err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	util.PubRPCSuccRsp(w, "inquiry", "FinInquiry")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func (s *server) Feedback(ctx context.Context, in *inquiry.FeedRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "inquiry", "Feedback")
	_, err := db.Exec("INSERT INTO feedback(uid, content, ctime) VALUES (?, ?, NOW())",
		in.Head.Uid, in.Content)
	if err != nil {
		log.Printf("Feedback failed:%d %v", in.Head.Uid, err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	util.PubRPCSuccRsp(w, "inquiry", "Feedback")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}
