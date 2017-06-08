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
	util.PubRPCSuccRsp(w, "inquiry", "GetDoctorInfo")
	return &inquiry.DoctorInfoReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Info: info}, nil
}
