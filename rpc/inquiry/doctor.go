package main

import (
	"Server/proto/common"
	"Server/proto/inquiry"
	"Server/util"
	"database/sql"
	"errors"
	"log"

	"golang.org/x/net/context"
)

func getDoctorInfo(db *sql.DB, uid int64) (*inquiry.DoctorInfo, error) {
	var role, doctor int64
	err := db.QueryRow("SELECT role, doctor FROM users WHERE uid = ?", uid).
		Scan(&role, &doctor)
	if err != nil {
		log.Printf("getDoctorInfo query role failed:%d %v", uid, err)
		return nil, err
	}
	if role == 0 || doctor == 0 {
		log.Printf("getDoctorInfo not doctor, uid:%d role:%d doctor:%d",
			uid, role, doctor)
		return nil, errors.New("not doctor")
	}
	var info inquiry.DoctorInfo
	err = db.QueryRow("SELECT name, headurl, title, hospital, department, fee FROM doctor WHERE id = ?", doctor).
		Scan(&info.Name, &info.Headurl, &info.Title, &info.Hospital,
			&info.Department, &info.Fee)
	if err != nil {
		log.Printf("getDoctorInfo get info failed:%d %v", uid, err)
		return nil, err
	}
	return &info, nil
}

func (s *server) GetDoctorInfo(ctx context.Context, in *common.CommRequest) (*inquiry.DoctorInfoReply, error) {
	util.PubRPCRequest(w, "inquiry", "GetDoctorInfo")
	info, err := getDoctorInfo(db, in.Id)
	if err != nil {
		log.Printf("getDoctorInfo failed:%d %v", in.Id, err)
		return &inquiry.DoctorInfoReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	if in.Head.Uid != in.Id {
		info.Fee = (int64(float64(info.Fee)*feeRate) / 100) * 100
	}
	util.PubRPCSuccRsp(w, "inquiry", "GetDoctorInfo")
	return &inquiry.DoctorInfoReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Info: info}, nil
}

func getPatientInfo(db *sql.DB, uid, tuid int64) (*inquiry.PatientInfo, error) {
	var pid int64
	err := db.QueryRow("SELECT pid FROM inquiry_history WHERE doctor = ? AND patient = ? ORDER BY id DESC LIMIT 1", uid, tuid).Scan(&pid)
	if err != nil {
		log.Printf("getPatientInfo get pid failed:%d %d %v", uid, tuid,
			err)
		return nil, err
	}
	var info inquiry.PatientInfo
	err = db.QueryRow("SELECT name, mcard, phone FROM patient WHERE id = ?", pid).
		Scan(&info.Name, &info.Mcard, &info.Phone)
	if err != nil {
		log.Printf("getPatientInfo query failed:%d %v", uid, err)
		return nil, err
	}
	return &info, nil
}

func (s *server) GetPatientInfo(ctx context.Context, in *common.CommRequest) (*inquiry.PatientInfoReply, error) {
	util.PubRPCRequest(w, "inquiry", "GetPatientInfo")
	info, err := getPatientInfo(db, in.Head.Uid, in.Id)
	if err != nil {
		log.Printf("getPatientInfo failed:%d %v", in.Id, err)
		return &inquiry.PatientInfoReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	util.PubRPCSuccRsp(w, "inquiry", "GetPatientInfo")
	return &inquiry.PatientInfoReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Info: info}, nil
}

func (s *server) SetFee(ctx context.Context, in *inquiry.FeeRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "inquiry", "SetFee")
	var role, doctor int64
	err := db.QueryRow("SELECT role, doctor FROM users WHERE uid = ?",
		in.Head.Uid).Scan(&role, &doctor)
	if err != nil {
		log.Printf("SetFee get user role failed, uid:%d %v", in.Head.Uid, err)
		return &common.CommReply{Head: &common.Head{Retcode: 1}}, nil
	}
	if role == 0 || doctor == 0 {
		log.Printf("SetFee not doctor uid:%d", in.Head.Uid)
		return &common.CommReply{Head: &common.Head{Retcode: 1}}, nil
	}
	_, err = db.Exec("UPDATE doctor SET fee = ? WHERE id = ?", in.Fee, doctor)
	if err != nil {
		log.Printf("SetFee update failed:%d %v", in.Head.Uid, err)
		return &common.CommReply{Head: &common.Head{Retcode: 1}}, nil
	}
	util.PubRPCSuccRsp(w, "inquiry", "SetFee")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0}}, nil
}
